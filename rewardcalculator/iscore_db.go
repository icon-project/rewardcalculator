package rewardcalculator

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
)


const (
	NumDelegate              = 10
)

type IconDB interface {
	ID() []byte
	Bytes() []byte
	String() string
	SetBytes([]byte) error
}

type IScoreDB struct {
	// Info. for service
	info *DBInfo

	// DB instance
	Global db.Database

	accountLock   sync.RWMutex
	claim         db.Database
	claimSnapshot db.Snapshot
	Account0      []db.Database
	Account1      []db.Database
}

func (idb *IScoreDB) getQueryDBList() []db.Database {
	idb.accountLock.RLock()
	defer idb.accountLock.RUnlock()
	if idb.info.QueryDBIsZero {
		return idb.Account0
	} else {
		return idb.Account1
	}
}

func (idb *IScoreDB) getCalcDBList() []db.Database {
	idb.accountLock.RLock()
	defer idb.accountLock.RUnlock()
	return idb._getCalcDBList()
}

func (idb *IScoreDB) _getCalcDBList() []db.Database {
	if idb.info.QueryDBIsZero {
		return idb.Account1
	} else {
		return idb.Account0
	}
}

func (idb *IScoreDB) toggleAccountDB() {
	idb.accountLock.Lock()
	idb.info.QueryDBIsZero = !idb.info.QueryDBIsZero
	idb.accountLock.Unlock()

	// write to DB
	idb.writeToDB()
}

func (idb *IScoreDB) getAccountDBIndex(address common.Address) int {
	prefix := int(address.ID()[0])
	return prefix % idb.info.DBCount
}

func (idb *IScoreDB) GetCalculateDB(address common.Address) db.Database {
	aDB := idb.getCalcDBList()
	return aDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) GetQueryDB(address common.Address) db.Database {
	aDB := idb.getQueryDBList()
	return aDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) GetClaimDB() db.Database {
	return idb.claim
}

func (idb *IScoreDB) SetClaimDBSnapshot() {
	snapshot, err := idb.claim.GetSnapshot()
	if err != nil {
		log.Printf("Failed to get snapshot of claim DB. err=%+v\n", err)
		return
	}

	snapshot.New()
	idb.claimSnapshot = snapshot
}

func (idb *IScoreDB) listClaimDBSnapshot(head string) {
	log.Printf("%s : list up claim db snapshot entries\n", head)
	// get iterator of claim DB snapshot
	idb.claimSnapshot.NewIterator(nil, nil)
	for idb.claimSnapshot.IterNext() {
		// read
		key := idb.claimSnapshot.IterKey()[len(db.PrefixIScore):]
		claim, err := NewClaimFromBytes(idb.claimSnapshot.IterValue())
		if err != nil {
			log.Printf("Can't read data with claim snapshot iterator\n")
			continue
		}
		claim.Address = *common.NewAddress(key)
		log.Printf("Claim Snapshot : %s\n", claim.String())
	}
	idb.claimSnapshot.ReleaseIterator()
}

func (idb *IScoreDB) GetClaimDBSnapshot() db.Snapshot {
	return idb.claimSnapshot
}

func (idb *IScoreDB) resetCalcDB() {
	idb.accountLock.Lock()
	defer idb.accountLock.Unlock()

	calcDBList := idb._getCalcDBList()
	var calcDBPostFix = 0
	if idb.info.QueryDBIsZero {
		calcDBPostFix = 1
	}

	newDBList := make([]db.Database, len(calcDBList))
	for i, calcDB := range calcDBList {
		calcDB.Close()
		dbName := fmt.Sprintf("%d_%d_%d", i+1, idb.info.DBCount, calcDBPostFix)
		os.RemoveAll(idb.info.DBRoot + "/" + dbName)
		newDBList[i] = db.Open(idb.info.DBRoot, idb.info.DBType, dbName)
	}

	if idb.info.QueryDBIsZero {
		idb.Account1 = newDBList
	} else {
		idb.Account0 = newDBList
	}
}

func (idb *IScoreDB) SetBlockHeight(blockHeight uint64) {
	idb.info.BlockHeight = blockHeight
	idb.writeToDB()
}
func (idb *IScoreDB) writeToDB() {
	bucket, _ := idb.Global.GetBucket(db.PrefixManagement)
	value, _ := idb.info.Bytes()
	bucket.Set(idb.info.ID(), value)
}

type GlobalOptions struct {
	BlockHeight     uint64

	db              *IScoreDB
	PRepCandidates  map[common.Address]*PRepCandidate
	GV              []*GovernanceVariable
}

// Update Governance variable with IISS data
func (opts *GlobalOptions) UpdateGovernanceVariable(gvList []*IISSGovernanceVariable) {
	bucket, _ := opts.db.Global.GetBucket(db.PrefixGovernanceVariable)

	// Update GV
	for _, gvIISS := range gvList {
		// there is new GV
		if  len(opts.GV) == 0 || opts.GV[len(opts.GV)-1].BlockHeight < gvIISS.BlockHeight {
			gv :=  NewGVFromIISS(gvIISS)

			// write to memory
			opts.GV = append(opts.GV, gv)

			// write to global DB
			value, _ := gv.Bytes()
			bucket.Set(gv.ID(), value)
		}
	}

	// delete old value
	gvLen := len(opts.GV)
	for i, gv := range opts.GV {
		if i != (gvLen - 1) &&gv.BlockHeight < opts.db.info.BlockHeight {
			// delete from global DB
			bucket.Delete(gv.ID())

			// delete from memory
			opts.GV = opts.GV[i:]
			break
		}
	}
}

// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
func (opts *GlobalOptions) UpdatePRepCandidate(txList []*IISSTX) {
	for _, tx := range txList {
		switch tx.DataType {
		case TXDataTypeDelegate:
		case TXDataTypePrepReg:
			pRep := opts.PRepCandidates[tx.Address]
			if pRep == nil {
				p := new(PRepCandidate)
				p.Address = tx.Address
				p.Start = tx.BlockHeight
				p.End = 0

				// write to memory
				opts.PRepCandidates[tx.Address] = p

				// write to global DB
				bucket, _ := opts.db.Global.GetBucket(db.PrefixPrepCandidate)
				data, _ := p.Bytes()
				bucket.Set(p.ID(), data)
			} else {
				log.Printf("P-Rep : '%s' was registered already\n", tx.Address.String())
				continue
			}
		case TXDataTypePrepUnReg:
			pRep := opts.PRepCandidates[tx.Address]
			if pRep != nil {
				if pRep.End != 0 {
					log.Printf("P-Rep : %s was unregistered already\n", tx.Address.String())
					continue
				}

				// write to memory
				pRep.End = tx.BlockHeight

				// write to global DB
				bucket, _ := opts.db.Global.GetBucket(db.PrefixPrepCandidate)
				data, _ := pRep.Bytes()
				bucket.Set(pRep.ID(), data)
			} else {
				log.Printf("P-Rep :  %s was not registered\n", tx.Address.String())
				continue
			}
		}
	}
}

func (opts *GlobalOptions) Print() {
	log.Printf("============================================================================")
	log.Printf("Global Option\n")
	log.Printf("Management Info.: %s\n", opts.db.info.String())
	log.Printf("Governance Variable: %d\n", len(opts.GV))
	for i, v := range opts.GV {
		log.Printf("\t%d: %s\n", i, v.String())
	}
	log.Printf("P-Rep candidate count : %d\n", len(opts.PRepCandidates))
	log.Printf("============================================================================")
}

func InitIScoreDB(dbPath string, dbType string, dbName string, worker int) (*GlobalOptions, error) {
	gOpts := new(GlobalOptions)
	isDB := new(IScoreDB)
	gOpts.db = isDB
	var err error

	// Open global DB
	globalDB := db.Open(dbPath, dbType, dbName)
	isDB.Global = globalDB

	// read management Info.
	isDB.info, err = NewDBInfo(globalDB, dbPath, dbType, dbName, worker)
	if err != nil {
		log.Panicf("Failed to load management Information\n")
		return nil, err
	}

	// read Governance variable
	gOpts.GV, err = LoadGovernanceVariable(globalDB, gOpts.db.info.BlockHeight)
	if err != nil {
		log.Printf("Failed to load GV structure\n")
		return nil, err
	}

	// read P-Rep candidate list
	gOpts.PRepCandidates, err = LoadPRepCandidate(globalDB)
	if err != nil {
		log.Printf("Failed to load P-Rep candidate structure\n")
		return nil, err
	}

	// Open account DBs for Query and Calculate
	isDB.Account0 = make([]db.Database, isDB.info.DBCount)
	for i := 0; i < isDB.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf("%d_%d_0", i + 1, isDB.info.DBCount)
		isDB.Account0[i] = db.Open(isDB.info.DBRoot, isDB.info.DBType, dbNameTemp)
	}
	isDB.Account1 = make([]db.Database, isDB.info.DBCount)
	for i := 0; i < isDB.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf("%d_%d_1", i + 1, isDB.info.DBCount)
		isDB.Account1[i] = db.Open(isDB.info.DBRoot, isDB.info.DBType, dbNameTemp)
	}

	// Open claim DB
	isDB.claim = db.Open(isDB.info.DBRoot, isDB.info.DBType, "claim")

	// TODO find IISS data and load

	return gOpts, nil
}

func CloseIScoreDB(isDB *IScoreDB) {
	log.Printf("Close 1 global DB and %d account DBs\n", len(isDB.Account0) + len(isDB.Account1))

	// close global DB
	isDB.Global.Close()

	// close account DBs
	for _, aDB := range isDB.Account0 {
		aDB.Close()
	}
	isDB.Account0 = nil
	for _, aDB := range isDB.Account1 {
		aDB.Close()
	}
	isDB.Account1 = nil
}
