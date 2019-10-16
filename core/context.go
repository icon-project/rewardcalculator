package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	NumDelegate         = 10
	AccountDBNameFormat = "calculate_%d_%d_%d"
	BackupDBNamePrefix  = "backup_"
	BackupDBNameFormat  = BackupDBNamePrefix + "%d_%d"	// CalcBH_accountDBIndex
)

type IScoreDB struct {
	// Info. for service
	info *DBInfo

	// DB instance
	management    db.Database
	calcResult    db.Database
	preCommit     db.Database
	claim         db.Database
	claimBackup   db.Database

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

func (idb *IScoreDB) GetCalcDBList() []db.Database {
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

func (idb *IScoreDB) OpenAccountDB() {
	idb.Account0 = make([]db.Database, idb.info.DBCount)
	for i := 0; i < idb.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf(AccountDBNameFormat, i+1, idb.info.DBCount, 0)
		idb.Account0[i] = db.Open(idb.info.DBRoot, idb.info.DBType, dbNameTemp)
	}
	idb.Account1 = make([]db.Database, idb.info.DBCount)
	for i := 0; i < idb.info.DBCount; i++ {
		dbNameTemp := fmt.Sprintf(AccountDBNameFormat, i+1, idb.info.DBCount, 1)
		idb.Account1[i] = db.Open(idb.info.DBRoot, idb.info.DBType, dbNameTemp)
	}
}

func (idb *IScoreDB) CloseAccountDB() {
	for _, aDB := range idb.Account0 {
		aDB.Close()
	}
	idb.Account0 = nil
	for _, aDB := range idb.Account1 {
		aDB.Close()
	}
	idb.Account1 = nil
}

func (idb *IScoreDB) getCalculateDB(address common.Address) db.Database {
	cDB := idb.GetCalcDBList()
	return cDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) getQueryDB(address common.Address) db.Database {
	qDB := idb.getQueryDBList()
	return qDB[idb.getAccountDBIndex(address)]
}

func (idb *IScoreDB) getPreCommitDB() db.Database {
	return idb.preCommit
}

func (idb *IScoreDB) getClaimDB() db.Database {
	return idb.claim
}

func (idb *IScoreDB) getClaimBackupDB() db.Database {
	return idb.claimBackup
}

func (idb *IScoreDB) getCalculateResultDB() db.Database {
	return idb.calcResult
}


func (idb *IScoreDB) resetAccountDB(blockHeight uint64) error {
	idb.accountLock.Lock()
	defer idb.accountLock.Unlock()

	// account DB was toggled, so calculate DB points old query DB
	oldQueryDBs := idb._getCalcDBList()
	var oldQueryDBPostFix = 0
	if idb.info.QueryDBIsZero {
		oldQueryDBPostFix = 1
	}

	// delete old backup account DB
	oldBackups, err := filepath.Glob(filepath.Join(idb.info.DBRoot, BackupDBNamePrefix + "*"))
	if err != nil {
		log.Printf("Failed to get old backup account DB")
		return err
	}
	for _, f := range oldBackups {
		err = os.RemoveAll(f)
		if err != nil {
			log.Printf("Failed to delete old backup account DB. %s", f)
			return err
		} else {
			log.Printf("delete old backup account DB. %s", f)
		}
	}

	newCalcDBs := make([]db.Database, len(oldQueryDBs))
	for i, oldQueryDB := range oldQueryDBs {
		oldQueryDB.Close()
		dbName := fmt.Sprintf(AccountDBNameFormat, i+1, idb.info.DBCount, oldQueryDBPostFix)
		backupName := fmt.Sprintf(BackupDBNameFormat, blockHeight, i+1)

		// backup old query DB
		err = os.Rename(filepath.Join(idb.info.DBRoot, dbName), filepath.Join(idb.info.DBRoot, backupName))
		if err != nil {
			log.Printf("Failed to backup old query DB. %s -> %s, %+v", dbName, backupName, err)
			return err
		} else {
			log.Printf("backup old query DB. %s -> %s", dbName, backupName)
		}

		// open new calculate DB
		newCalcDBs[i] = db.Open(idb.info.DBRoot, idb.info.DBType, dbName)
	}

	// set new calculate DB
	if idb.info.QueryDBIsZero {
		idb.Account1 = newCalcDBs
	} else {
		idb.Account0 = newCalcDBs
	}

	return nil
}

func (idb *IScoreDB) setCalculateBlockHeight(blockHeight uint64) {
	idb.info.CalcBlockHeight = blockHeight

	idb.writeToDB()
}

func (idb *IScoreDB) resetCalculateBlockHeight() {
	idb.info.CalcBlockHeight = idb.info.BlockHeight

	idb.writeToDB()
}

func (idb *IScoreDB) isCalculating() bool {
	return idb.info.CalcBlockHeight > idb.info.BlockHeight
}

func (idb *IScoreDB) setBlockInfo(blockHeight uint64, blockHash []byte) {
	tempBlockHash := make([]byte, BlockHashSize)
	copy(tempBlockHash, blockHash)

	// backup to prev Info.
	if idb.info.PrevBlockHeight < blockHeight {
		idb.info.PrevBlockHeight = idb.info.BlockHeight
		copy(idb.info.PrevBlockHash, idb.info.BlockHash)
	}

	// set current Info.
	idb.info.BlockHeight = blockHeight
	copy(idb.info.BlockHash, tempBlockHash)

	idb.writeToDB()
}

func (idb *IScoreDB) rollbackBlockInfo() {
	idb.info.BlockHeight = idb.info.PrevBlockHeight
	copy(idb.info.BlockHash, idb.info.PrevBlockHash)
	idb.setCalculateBlockHeight(idb.info.BlockHeight)

	idb.writeToDB()
}

func (idb *IScoreDB) writeToDB() {
	bucket, _ := idb.management.GetBucket(db.PrefixManagement)
	value, _ := idb.info.Bytes()
	bucket.Set(idb.info.ID(), value)
}

func (idb *IScoreDB) rollbackAccountDB(blockHeight uint64) error {
	log.Printf("Start Rollback account DB to %d", blockHeight)
	var calcDBPostFix = 0
	if idb.info.QueryDBIsZero {
		calcDBPostFix = 1
	}

	backups, err := filepath.Glob(filepath.Join(idb.info.DBRoot, BackupDBNamePrefix + "*"))
	if err != nil {
		log.Printf("Failed to get backup account DB")
		return err
	}

	if len(backups) != idb.info.DBCount {
		return fmt.Errorf("there is no backup account DB. %d", len(backups))
	} else {
		_, name := filepath.Split(backups[0])
		nameSlice := strings.Split(name, "_")
		backupBH, err := strconv.ParseUint(nameSlice[1], 10, 64)
		if err != nil {
			return err
		}
		if blockHeight >= backupBH {
			// no need to Rollback account DB
			log.Printf("no need to Rollback account DB to %d. backup: %d", blockHeight, backupBH)
			return nil
		}
	}

	// rollback account DB
	idb.CloseAccountDB()
	for i, f := range backups {
		calcDBName := fmt.Sprintf(AccountDBNameFormat, i+1, idb.info.DBCount, calcDBPostFix)

		// remove calculate DB
		err = os.RemoveAll(filepath.Join(idb.info.DBRoot, calcDBName))
		if err != nil {
			log.Printf("Failed to remove old calculate DB")
			return err
		} else {
			log.Printf("remove old calculate DB. %s", calcDBName)
		}

		// rename backup DB to calculate DB
		err = os.Rename(f, filepath.Join(idb.info.DBRoot, calcDBName))
		if err != nil {
			log.Printf("Failed to rename backup DB to query DB. %s -> %s", f, calcDBName)
			return err
		} else {
			log.Printf("rename backup DB to query DB. %s -> %s", f, calcDBName)
		}
	}
	idb.OpenAccountDB()

	// toggle query DB switch
	idb.toggleAccountDB()

	// Rollback block height and block hash
	idb.rollbackBlockInfo()

	// delete calculation result
	DeleteCalculationResult(idb.getCalculateResultDB(), idb.info.CalcBlockHeight)

	log.Printf("End rollblack account DB to %d", blockHeight)
	return nil
}

type Context struct {
	DB              *IScoreDB

	Revision        uint64
	PRep            []*PRep
	PRepCandidates  map[common.Address]*PRepCandidate
	GV              []*GovernanceVariable

	stats    *Statistics
	Rollback *Rollback
}

func (ctx *Context) getGVByBlockHeight(blockHeight uint64) *GovernanceVariable {
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
	bucket, _ := ctx.DB.management.GetBucket(db.PrefixGovernanceVariable)

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
		if ctx.GV[i].BlockHeight < ctx.DB.info.BlockHeight {
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
	if deleteOld && deleteIndex > 0 {
		ctx.GV = ctx.GV[deleteIndex:]
	}
}

// Update Main/Sub P-Rep list
func (ctx *Context) UpdatePRep(prepList []*PRep) {
	bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRep)

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
		if ctx.PRep[i].BlockHeight < ctx.DB.info.BlockHeight {
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
	if deleteOld && deleteIndex > 0 {
		ctx.PRep = ctx.PRep[deleteIndex:]
	}
}

// Update P-Rep candidate with IISS TX(P-Rep register/unregister)
func (ctx *Context) UpdatePRepCandidate(iissDB db.Database) {
	var tx IISSTX

	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	entries := 0
	for entries = 0; iter.Next(); entries++ {
		err := tx.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to load IISS TX data")
			continue
		}
		tx.Index = common.BytesToUint64(iter.Key()[len(db.PrefixIISSTX):])
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
				bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRepCandidate)
				data, _ := p.Bytes()
				bucket.Set(p.ID(), data)
				log.Printf("P-Rep : register '%s'", tx.Address.String())
			} else {
				log.Printf("P-Rep : '%s' was registered already\n", tx.Address.String())
				continue
			}
		case TXDataTypePrepUnReg:
			pRep, ok := ctx.PRepCandidates[tx.Address]
			if ok == true {
				if pRep.End != 0 {
					log.Printf("P-Rep : %s was unregistered already\n", tx.Address.String())
					continue
				}

				// write to memory
				pRep.End = tx.BlockHeight

				// write to global DB
				bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRepCandidate)
				data, _ := pRep.Bytes()
				bucket.Set(pRep.ID(), data)
				log.Printf("P-Rep : unregister '%s'", tx.Address.String())
			} else {
				log.Printf("P-Rep :  %s was not registered\n", tx.Address.String())
				continue
			}
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while IISS TX iteration for P-Rep update. %+v", err)
	}
}

func (ctx *Context) Print() {
	log.Printf("============================================================================")
	log.Printf("Print context values\n")
	log.Printf("Revision : %d\n", ctx.Revision)
	log.Printf("Database Info.: %s\n", ctx.DB.info.String())
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
	ctx.DB = isDB
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
	ctx.GV, err = LoadGovernanceVariable(mngDB, ctx.DB.info.BlockHeight)
	if err != nil {
		log.Printf("Failed to load GV structure\n")
		return nil, err
	}

	// read P-Rep
	ctx.PRep, err = LoadPRep(mngDB)
	if err != nil {
		log.Printf("Failed to load P-Rep structure\n")
		return nil, err
	}

	// read P-Rep candidate list
	ctx.PRepCandidates, err = LoadPRepCandidate(mngDB)
	if err != nil {
		log.Printf("Failed to load P-Rep candidate structure\n")
		return nil, err
	}

	// Open calculation result DB
	isDB.calcResult= db.Open(isDB.info.DBRoot, isDB.info.DBType, "calculation_result")

	// Open preCommit DB
	isDB.preCommit = db.Open(isDB.info.DBRoot, isDB.info.DBType, "preCommit")

	// Open claim DB
	isDB.claim = db.Open(isDB.info.DBRoot, isDB.info.DBType, "claim")

	// Open claim backup DB
	isDB.claimBackup = db.Open(isDB.info.DBRoot, isDB.info.DBType, "claim_backup")

	// Open account DB
	isDB.OpenAccountDB()

	// make new Rollback stuff
	ctx.Rollback = NewRollback()

	return ctx, nil
}

func CloseIScoreDB(isDB *IScoreDB) {
	log.Printf("Close 1 global DB and %d account DBs\n", len(isDB.Account0) + len(isDB.Account1))

	// close management DB
	isDB.management.Close()

	// close account DBs
	isDB.CloseAccountDB()

	// close calculation result DB
	isDB.calcResult.Close()

	// close preCommit DB
	isDB.preCommit.Close()

	// close claim DB
	isDB.claim.Close()

	// close claim backup DB
	isDB.claimBackup.Close()
}
