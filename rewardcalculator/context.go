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

type IScoreDB struct {
	// Info. for service
	info *DBInfo

	// DB instance
	management    db.Database

	claim         db.Database

	accountLock   sync.RWMutex
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
		if i != (gvLen - 1) &&gv.BlockHeight < ctx.db.info.BlockHeight {
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

	// Init prCommit
	ctx.preCommit = new(preCommit)

	// TODO find IISS data and load

	return ctx, nil
}

func CloseIScoreDB(isDB *IScoreDB) {
	log.Printf("Close 1 global DB and %d account DBs\n", len(isDB.Account0) + len(isDB.Account1))

	// close management DB
	isDB.management.Close()

	// close account DBs
	for _, aDB := range isDB.Account0 {
		aDB.Close()
	}
	isDB.Account0 = nil
	for _, aDB := range isDB.Account1 {
		aDB.Close()
	}
	isDB.Account1 = nil

	// close claim DB
	isDB.claim.Close()
}
