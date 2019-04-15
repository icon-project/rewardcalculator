package rewardcalculator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
)


const (
	NumDelegate           = 10
	CalculateDBNameFormat = "calculate_%d_%d_%d"
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

func (idb *IScoreDB) getCalculateDB(address common.Address) db.Database {
	cDB := idb.getCalcDBList()
	return cDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) getQueryDB(address common.Address) db.Database {
	qDB := idb.getQueryDBList()
	return qDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) getClaimDB() db.Database {
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
		dbName := fmt.Sprintf(CalculateDBNameFormat, i+1, idb.info.DBCount, calcDBPostFix)
		os.RemoveAll(filepath.Join(idb.info.DBRoot, dbName))
		newDBList[i] = db.Open(idb.info.DBRoot, idb.info.DBType, dbName)
	}

	if idb.info.QueryDBIsZero {
		idb.Account1 = newDBList
	} else {
		idb.Account0 = newDBList
	}
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

	PRep            []*PRep
	PRepCandidates  map[common.Address]*PRepCandidate
	GV              []*GovernanceVariable

	preCommit       *preCommit
}

func (ctx *Context) getGV(blockHeight uint64) *GovernanceVariable {
	gvLen := len(ctx.GV)
	for i := gvLen - 1; i >= 0; i-- {
		if ctx.GV[i].BlockHeight < blockHeight {
			return ctx.GV[i]
		}
	}
	return nil
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

			// write to management DB
			value, _ := gv.Bytes()
			bucket.Set(gv.ID(), value)
		}
	}

	// delete old value
	gvLen := len(ctx.GV)
	deleteOld := false
	deleteIndex := -1
	for i := gvLen - 1; i >= 0 ; i-- {
		if ctx.GV[i].BlockHeight < ctx.db.info.BlockHeight {
			if deleteOld {
				// delete from management DB
				bucket.Delete(ctx.GV[i].ID())
			} else {
				deleteOld = true
				deleteIndex = i
			}
		}
	}
	// delete old value from memory
	if deleteOld && deleteIndex != -1 {
		ctx.GV = ctx.GV[deleteIndex:]
	}
}

// Update Main/Sub P-Rep list
func (ctx *Context) UpdatePRep(prepList []*PRep) {
	bucket, _ := ctx.db.management.GetBucket(db.PrefixPRep)

	// Update GV
	for _, prep := range prepList {
		// write to memory
		ctx.PRep = append(ctx.PRep, prep)

		// write to management DB
		value, _ := prep.Bytes()
		bucket.Set(prep.ID(), value)
	}

	// delete old value
	prepLen := len(ctx.PRep)
	deleteOld := false
	deleteIndex := -1
	for i := prepLen - 1; i >= 0 ; i-- {
		if ctx.PRep[i].BlockHeight < ctx.db.info.BlockHeight {
			if deleteOld {
				// delete from management DB
				bucket.Delete(ctx.PRep[i].ID())
			} else {
				deleteOld = true
				deleteIndex = i
			}
		}
	}
	// delete old value from memory
	if deleteOld && deleteIndex != -1 {
		ctx.PRep = ctx.PRep[deleteIndex:]
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
	log.Printf("Print context values\n")
	log.Printf("Database Info.: %s\n", ctx.db.info.String())
	log.Printf("Governance Variable: %d\n", len(ctx.GV))
	for i, v := range ctx.GV {
		log.Printf("\t%d: %s\n", i, v.String())
	}
	log.Printf("P-Rep list: %d\n", len(ctx.PRep))
	for i, v := range ctx.PRep {
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
		dbNameTemp := fmt.Sprintf(CalculateDBNameFormat, i + 1, isDB.info.DBCount, 0)
		isDB.Account0[i] = db.Open(isDB.info.DBRoot, isDB.info.DBType, dbNameTemp)
	}
	isDB.Account1 = make([]db.Database, isDB.info.DBCount)
	for i := 0; i < isDB.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf(CalculateDBNameFormat, i + 1, isDB.info.DBCount, 1)
		isDB.Account1[i] = db.Open(isDB.info.DBRoot, isDB.info.DBType, dbNameTemp)
	}

	// Open claim DB
	isDB.claim = db.Open(isDB.info.DBRoot, isDB.info.DBType, "claim")

	// Init preCommit
	ctx.preCommit = new(preCommit)

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
