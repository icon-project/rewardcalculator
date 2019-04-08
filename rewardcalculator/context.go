package rewardcalculator

import (
	"fmt"
	"log"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
)


const (
	NumDelegate              = 10
)

type IScoreDB struct {
	// Info. for service
	info *DBInfo

	// DB instance
	management  db.Database
	claim       db.Database

	accountLock sync.RWMutex
	calculate   []db.Database
	query       []db.Snapshot
}

func (idb *IScoreDB) getAccountDBIndex(address common.Address) int {
	prefix := int(address.ID()[0])
	return prefix % idb.info.DBCount
}

func (idb *IScoreDB) setQueryDB() {
	idb.accountLock.Lock()
	defer idb.accountLock.Unlock()

	if idb.query == nil {
		idb.query = make([]db.Snapshot, idb.info.DBCount)
	}

	calcDBList := idb.getCalculateDBList()

	for i, calcDB := range calcDBList {
		snapshot, err := calcDB.GetSnapshot()
		if err != nil {
			log.Printf("Failed to get snapshot of calculation DB(%d). err=%+v\n", i, err)
			return
		}

		// release old snapshot
		if idb.query[i] != nil {
			idb.query[i].Release()
		}

		// make new snapshot
		snapshot.New()
		idb.query[i] = snapshot
	}
}

func (idb *IScoreDB) getQueryDBList() []db.Snapshot{
	return idb.query
}

func (idb *IScoreDB) getFromQueryDB(address common.Address) ([]byte, error) {
	idb.accountLock.RLock()
	defer idb.accountLock.RUnlock()
	qDB := idb.getQueryDBList()
	snapshot := qDB[idb.getAccountDBIndex(address)]
	return snapshot.Get(address.Bytes())
}

func (idb *IScoreDB) getCalculateDBList() []db.Database {
	return idb.calculate
}

func (idb *IScoreDB) getCalculateDB(address common.Address) db.Database {
	aDB := idb.getCalculateDBList()
	return aDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) getClaimDB() db.Database {
	return idb.claim
}

func (idb *IScoreDB) setBlockHeight(blockHeight uint64) {
	idb.info.BlockHeight = blockHeight
	idb.writeToDB()
}

func (idb *IScoreDB) writeToDB() {
	bucket, _ := idb.management.GetBucket(db.PrefixManagement)
	value, _ := idb.info.Bytes()
	bucket.Set(idb.info.ID(), value)
}

type Context struct {
	db              *IScoreDB

	PRepCandidates  map[common.Address]*PRepCandidate
	GV              []*GovernanceVariable

	preCommit       *preCommit
}

// Update Governance variable with IISS data
func (ctx *Context) UpdateGovernanceVariable(gvList []*IISSGovernanceVariable) {
	bucket, _ := ctx.db.management.GetBucket(db.PrefixGovernanceVariable)

	// Update GV
	for _, gvIISS := range gvList {
		// there is new GV
		if  len(ctx.GV) == 0 || ctx.GV[len(ctx.GV)-1].BlockHeight < gvIISS.BlockHeight {
			gv :=  NewGVFromIISS(gvIISS)

			// write to memory
			ctx.GV = append(ctx.GV, gv)

			// write to global DB
			value, _ := gv.Bytes()
			bucket.Set(gv.ID(), value)
		}
	}

	// delete old value
	gvLen := len(ctx.GV)
	for i, gv := range ctx.GV {
		// FIXME check delete logic
		if i < (gvLen - 1) && gv.BlockHeight < ctx.db.info.BlockHeight {
			// delete from global DB
			bucket.Delete(gv.ID())

			// delete from memory
			ctx.GV = ctx.GV[i:]
			break
		}
	}
}

// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
func (ctx *Context) UpdatePRepCandidate(txList []*IISSTX) {
	for _, tx := range txList {
		switch tx.DataType {
		case TXDataTypeDelegate:
		case TXDataTypePrepReg:
			pRep := ctx.PRepCandidates[tx.Address]
			if pRep == nil {
				p := new(PRepCandidate)
				p.Address = tx.Address
				p.Start = tx.BlockHeight
				p.End = 0

				// write to memory
				ctx.PRepCandidates[tx.Address] = p

				// write to global DB
				bucket, _ := ctx.db.management.GetBucket(db.PrefixPrepCandidate)
				data, _ := p.Bytes()
				bucket.Set(p.ID(), data)
			} else {
				log.Printf("P-Rep : '%s' was registered already\n", tx.Address.String())
				continue
			}
		case TXDataTypePrepUnReg:
			pRep := ctx.PRepCandidates[tx.Address]
			if pRep != nil {
				if pRep.End != 0 {
					log.Printf("P-Rep : %s was unregistered already\n", tx.Address.String())
					continue
				}

				// write to memory
				pRep.End = tx.BlockHeight

				// write to global DB
				bucket, _ := ctx.db.management.GetBucket(db.PrefixPrepCandidate)
				data, _ := pRep.Bytes()
				bucket.Set(pRep.ID(), data)
			} else {
				log.Printf("P-Rep :  %s was not registered\n", tx.Address.String())
				continue
			}
		}
	}
}

func (ctx *Context) Print() {
	log.Printf("============================================================================")
	log.Printf("Context\n")
	log.Printf("Database Info.: %s\n", ctx.db.info.String())
	log.Printf("Governance Variable: %d\n", len(ctx.GV))
	for i, v := range ctx.GV {
		log.Printf("\t%d: %s\n", i, v.String())
	}
	log.Printf("P-Rep candidate count : %d\n", len(ctx.PRepCandidates))
	log.Printf("============================================================================")
}

func NewContext(dbPath string, dbType string, dbName string, dbCount int) (*Context, error) {
	ctx := new(Context)
	isDB := new(IScoreDB)
	ctx.db = isDB
	var err error

	// Open management DB
	mngDB := db.Open(dbPath, dbType, dbName)
	isDB.management = mngDB

	// read DB Info.
	isDB.info, err = NewDBInfo(mngDB, dbPath, dbType, dbName, dbCount)
	if err != nil {
		log.Panicf("Failed to load DB Information\n")
		return nil, err
	}

	// read Governance variable
	ctx.GV, err = LoadGovernanceVariable(mngDB, ctx.db.info.BlockHeight)
	if err != nil {
		log.Printf("Failed to load GV structure\n")
		return nil, err
	}

	// read P-Rep candidate list
	ctx.PRepCandidates, err = LoadPRepCandidate(mngDB)
	if err != nil {
		log.Printf("Failed to load P-Rep candidate structure\n")
		return nil, err
	}

	// Open DB for Calculate
	isDB.calculate = make([]db.Database, isDB.info.DBCount)
	for i := 0; i < isDB.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf("calculate_%d_%d", i + 1, isDB.info.DBCount)
		isDB.calculate[i] = db.Open(isDB.info.DBRoot, isDB.info.DBType, dbNameTemp)
	}

	// Set snapshot of Calculate DBs as query DB
	isDB.setQueryDB()

	// Open claim DB
	isDB.claim = db.Open(isDB.info.DBRoot, isDB.info.DBType, "claim")

	// Init preCommit
	ctx.preCommit = new(preCommit)

	// TODO find IISS data and load

	return ctx, nil
}

func CloseIScoreDB(isDB *IScoreDB) {
	log.Printf("Close DBs\n")

	// close management DB
	isDB.management.Close()

	// close account DBs
	for _, aDB := range isDB.calculate {
		aDB.Close()
	}
	isDB.calculate = nil

	for _, snaphost := range isDB.query {
		snaphost.Release()
	}
	isDB.query = nil

	// close claim DB
	isDB.claim.Close()
}
