package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
)

var testDir string

func initTest(dbCount int) *Context{
	var err error
	testDir, err = ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}

	ctx, _ := NewContext(testDir, string(db.GoLevelDBBackend), "test", dbCount)

	return ctx
}

func finalizeTest(ctx *Context) {
	CloseIScoreDB(ctx.DB)
	os.RemoveAll(testDir)
}

func TestContext_NewContext(t *testing.T) {
	const dbCount int = 16
	ctx := initTest(dbCount)
	defer finalizeTest(ctx)
	assert.NotNil(t, ctx)

	assert.NotNil(t, ctx.DB)
	assert.NotNil(t, ctx.DB.info)
	assert.Equal(t, dbCount, ctx.DB.info.DBCount)
	assert.True(t, ctx.DB.getCurrentBlockInfo().checkValue(uint64(0), zeroHash))
	assert.Equal(t, uint64(0), ctx.DB.getCalcDoneBH())
	assert.Equal(t, uint64(0), ctx.DB.getPrevCalcDoneBH())
	assert.Equal(t, uint64(0), ctx.DB.getCalculatingBH())
	assert.False(t, ctx.DB.info.QueryDBIsZero)
	assert.Equal(t, dbCount, len(ctx.DB.Account0))
	assert.Equal(t, dbCount, len(ctx.DB.Account1))
	assert.NotNil(t, ctx.DB.calcResult)
	assert.NotNil(t, ctx.DB.preCommit)
	assert.NotNil(t, ctx.DB.claim)

	assert.NotNil(t, ctx.GV)
	assert.NotNil(t, ctx.PRep)
	assert.NotNil(t, ctx.PRepCandidates)
}

func TestContext_GovernanceVariable(t *testing.T) {
	const (
		ctxBlockHeight1 uint64 = 50
		ctxBlockHeight2 uint64 = 100
	)
	ctx := initTest(1)
	defer finalizeTest(ctx)

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixGovernanceVariable)

	// Set Block height of CalcDone and PrevCalcDone
	ctx.DB.setCalcDoneBH(ctxBlockHeight1)

	// Insert initial GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = ctxBlockHeight1 - 20
	gv.RewardRep = *common.NewHexIntFromUint64(ctxBlockHeight1 - 20)
	ctx.GV = append(ctx.GV, gv)
	value, _ := gv.Bytes()
	bucket.Set(gv.ID(), value)

	gv = new(GovernanceVariable)
	gv.BlockHeight = ctxBlockHeight1
	gv.RewardRep = *common.NewHexIntFromUint64(ctxBlockHeight1)
	ctx.GV = append(ctx.GV, gv)
	value, _ = gv.Bytes()
	bucket.Set(gv.ID(), value)

	// Set Block height of CalcDone and PrevCalcDone
	ctx.DB.setCalcDoneBH(ctxBlockHeight2)

	// make IISSGovernanceVariable list
	gvList := make([]*IISSGovernanceVariable, 0)
	iissGV := new(IISSGovernanceVariable)
	iissGV.BlockHeight = ctxBlockHeight2 - 20
	iissGV.RewardRep = ctxBlockHeight2 - 20
	gvList = append(gvList, iissGV)
	iissGV = new(IISSGovernanceVariable)
	iissGV.BlockHeight = ctxBlockHeight2
	iissGV.RewardRep = ctxBlockHeight2
	gvList = append(gvList, iissGV)

	// update GV
	ctx.UpdateGovernanceVariable(gvList)

	// check - len
	assert.Equal(t, len(gvList) + 1, len(ctx.GV))

	// In memory
	assert.Equal(t, ctxBlockHeight1, ctx.GV[0].BlockHeight)
	// In DB
	bs, _ := bucket.Get(ctx.GV[0].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[0].RewardRep.Cmp(&gv.RewardRep.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight2 - 20, ctx.GV[1].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.GV[1].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[1].RewardRep.Cmp(&gv.RewardRep.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight2, ctx.GV[2].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.GV[2].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[2].RewardRep.Cmp(&gv.RewardRep.Int))

	// test rollback
	ctx.RollbackManagementDB(ctxBlockHeight1)

	// check - len
	assert.Equal(t, 1, len(ctx.GV))

	// In memory
	assert.Equal(t, ctxBlockHeight1, ctx.GV[0].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.GV[0].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[0].RewardRep.Cmp(&gv.RewardRep.Int))
}

func TestContext_PRep(t *testing.T) {
	const (
		ctxBlockHeight1 uint64 = 50
		ctxBlockHeight2 uint64 = 100
	)
	ctx := initTest(1)
	defer finalizeTest(ctx)

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRep)

	// Set Block height
	ctx.DB.setCalcDoneBH(ctxBlockHeight1)

	// Insert initial PRep
	pRep := new(PRep)
	pRep.BlockHeight = ctxBlockHeight1 - 20
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight1 - 20)
	ctx.PRep = append(ctx.PRep, pRep)
	value, _ := pRep.Bytes()
	bucket.Set(pRep.ID(), value)

	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight1
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight1)
	ctx.PRep = append(ctx.PRep, pRep)
	value, _ = pRep.Bytes()
	bucket.Set(pRep.ID(), value)

	// Set Block height
	ctx.DB.setCalcDoneBH(ctxBlockHeight1)

	// make IISS list
	prepList := make([]*PRep, 0)
	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight2 - 20
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight1 - 20)
	prepList = append(prepList, pRep)
	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight2
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight2)
	prepList = append(prepList, pRep)

	// update PRep
	ctx.UpdatePRep(prepList)

	// check - len
	assert.Equal(t, len(prepList) + 1, len(ctx.PRep))

	// check - values

	// In memory
	assert.Equal(t, ctxBlockHeight1, ctx.PRep[0].BlockHeight)
	// In DB
	bs, _ := bucket.Get(ctx.PRep[0].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[0].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight2 - 20, ctx.PRep[1].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.PRep[1].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[1].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight2, ctx.PRep[2].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.PRep[2].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[2].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

	// rollback
	ctx.RollbackManagementDB(ctxBlockHeight1)

	// check - len
	assert.Equal(t, 1, len(ctx.PRep))

	// In memory
	assert.Equal(t, ctxBlockHeight1, ctx.PRep[0].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.PRep[0].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[0].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

}

func TestContext_UpdatePRepCandidate(t *testing.T) {
	const (
		regPRepABH uint64 = 1
		unRegPRepABH uint64 = 100
		NotUnRegBH uint64 = 0
	)
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// write IISS TX
	iissDBDir := testDBDir + "/iiss"
	iissDB := db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	txList := make([]*IISSTX, 0)
	pRepA := *common.NewAddressFromString("hxa")

	// 1. register PRep candidate A
	tx := makeIISSTX(TXDataTypePrepReg, pRepA.String(), nil)
	tx.BlockHeight = regPRepABH
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(iissDB)
	iissDB.Close()
	os.RemoveAll(iissDBDir)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, NotUnRegBH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bucketPRep, _ := ctx.DB.management.GetBucket(db.PrefixPRepCandidate)
	bs, _ := bucketPRep.Get(pRepA.Bytes())
	var pRep PRepCandidate
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, NotUnRegBH, pRep.End)

	// 2. register PRep candidate A with different block height again
	iissDB = db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	txList = make([]*IISSTX, 0)
	tx = makeIISSTX(TXDataTypePrepReg, pRepA.String(), nil)
	tx.BlockHeight = regPRepABH + 1
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(iissDB)
	iissDB.Close()
	os.RemoveAll(iissDBDir)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, NotUnRegBH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ = bucketPRep.Get(pRepA.Bytes())
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, NotUnRegBH, pRep.End)

	// 3. unregister PRep candidate A
	iissDB = db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	txList = make([]*IISSTX, 0)
	tx = makeIISSTX(TXDataTypePrepUnReg, pRepA.String(), nil)
	tx.BlockHeight = unRegPRepABH
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(iissDB)
	iissDB.Close()
	os.RemoveAll(iissDBDir)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[pRepA].Start)
	assert.Equal(t, unRegPRepABH, ctx.PRepCandidates[pRepA].End)

	// check - in DB
	bs, _ = bucketPRep.Get(pRepA.Bytes())
	pRep.SetBytes(bs)
	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, unRegPRepABH, pRep.End)

	// 4. unregister PRep candidate A with different block height again
	iissDB = db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	txList = make([]*IISSTX, 0)
	tx = makeIISSTX(TXDataTypePrepUnReg, pRepA.String(), nil)
	tx.BlockHeight = unRegPRepABH + 1
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(iissDB)
	iissDB.Close()
	os.RemoveAll(iissDBDir)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, unRegPRepABH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ = bucketPRep.Get(pRepA.Bytes())
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, unRegPRepABH, pRep.End)

	// 5. unregister PRep candidate B
	iissDB = db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	txList = make([]*IISSTX, 0)
	tx = makeIISSTX(TXDataTypePrepUnReg, pRepA.String(), nil)
	tx.DataType = TXDataTypePrepUnReg
	tx.BlockHeight = unRegPRepABH
	tx.Address = *common.NewAddressFromString("hxb")
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(iissDB)
	iissDB.Close()
	os.RemoveAll(iissDBDir)

	// check - in memory
	_, ok := ctx.PRepCandidates[tx.Address]
	assert.False(t, ok)

	// check - in DB
	bs, _ = bucketPRep.Get(tx.Address.Bytes())
	assert.Nil(t, bs)
}

func TestContext_GetQueryDBList(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	ctx.DB.info.QueryDBIsZero = true
	assert.Equal(t, ctx.DB.Account0, ctx.DB.getQueryDBList())
	ctx.DB.info.QueryDBIsZero = false
	assert.Equal(t, ctx.DB.Account1, ctx.DB.getQueryDBList())
}

func TestContext_GetCalcDBList(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	ctx.DB.info.QueryDBIsZero = true
	assert.Equal(t, ctx.DB.Account1, ctx.DB.GetCalcDBList())
	ctx.DB.info.QueryDBIsZero = false
	assert.Equal(t, ctx.DB.Account0, ctx.DB.GetCalcDBList())
}

func TestContext_GetPreCommitDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	assert.Equal(t, ctx.DB.preCommit, ctx.DB.getPreCommitDB())
}

func TestContext_GetClaimDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	assert.Equal(t, ctx.DB.claim, ctx.DB.getClaimDB())
}

func TestContext_GetCalculateResultDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	assert.Equal(t, ctx.DB.calcResult, ctx.DB.getCalculateResultDB())
}

func TestContext_ToggleAccountDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	original := ctx.DB.info.QueryDBIsZero

	ctx.DB.toggleAccountDB(10)
	assert.Equal(t, !original, ctx.DB.info.QueryDBIsZero)
	assert.Equal(t, uint64(10), ctx.DB.info.ToggleBH)
	ctx.DB.toggleAccountDB(20)
	assert.Equal(t, original, ctx.DB.info.QueryDBIsZero)
	assert.Equal(t, uint64(20), ctx.DB.info.ToggleBH)
}

func TestContext_ResetAccountDB(t *testing.T) {
	dbCount := 4
	ctx := initTest(dbCount)
	defer finalizeTest(ctx)

	qDBList := ctx.DB.getQueryDBList()
	cDBList := ctx.DB.GetCalcDBList()

	// toggled now. write to old query DB to check backup DB
	ia := makeIA()
	qDB := ctx.DB.getCalculateDB(ia.Address)
	bucket, _ := qDB.GetBucket(db.PrefixIScore)
	err := bucket.Set(ia.ID(), ia.Bytes())
	assert.NoError(t, err)

	blockHeight := uint64(1000)
	oldBlockHeight := ctx.DB.getCalcDoneBH()
	err = ctx.DB.resetAccountDB(blockHeight, oldBlockHeight)
	assert.NoError(t, err)

	// same query DB
	assert.Equal(t, qDBList, ctx.DB.getQueryDBList())

	// new calculate DB
	assert.NotEqual(t, cDBList, ctx.DB.GetCalcDBList())
	assert.Equal(t, dbCount, len(ctx.DB.GetCalcDBList()))

	// old and new backup DB
	for i := 0; i < ctx.DB.info.DBCount; i++ {
		// old backup DB were deleted
		backupName := fmt.Sprintf(BackupDBNameFormat, oldBlockHeight, i+1)
		stat, err := os.Stat(filepath.Join(ctx.DB.info.DBRoot, backupName))
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		// new backup DB wew created
		backupName = fmt.Sprintf(BackupDBNameFormat, blockHeight, i+1)
		stat, err = os.Stat(filepath.Join(ctx.DB.info.DBRoot, backupName))
		assert.NoError(t, err)
		assert.True(t, stat.IsDir())
	}

	// check value in backup DB
	backupName := fmt.Sprintf(BackupDBNameFormat, blockHeight, ctx.DB.getAccountDBIndex(ia.Address) + 1)
	backupDB := db.Open(ctx.DB.info.DBRoot, ctx.DB.info.DBType, backupName)
	bucket, _ = backupDB.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(ia.ID())
	assert.NoError(t, err)
	assert.NotNil(t, bs)
	assert.Equal(t, ia.Bytes(), bs)
	backupDB.Close()
}

func TestContext_CurrentBlockInfo(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	const (
		blockHeight1 uint64 = 100
		blockHeight2 uint64 = 200
	)

	blockHash1 := make([]byte, BlockHashSize)
	copy(blockHash1,  []byte(string(blockHeight1)))
	blockHash2 := make([]byte, BlockHashSize)
	copy(blockHash2,  []byte(string(blockHeight2)))

	ctx.DB.setCurrentBlockInfo(blockHeight2, blockHash2)
	assert.True(t, ctx.DB.getCurrentBlockInfo().checkValue(blockHeight2, blockHash2))

	ctx.DB.rollbackCurrentBlockInfo(blockHeight1, blockHash1)
	assert.True(t, ctx.DB.getCurrentBlockInfo().checkValue(blockHeight1, blockHash1))
}

func TestContext_AccountDBBlockHeight(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	const (
		blockHeight1 uint64 = 100
		blockHeight2 uint64 = 200
	)

	ctx.DB.setCalcDoneBH(blockHeight1)
	ctx.DB.setCalcDoneBH(blockHeight2)
	assert.Equal(t, blockHeight2, ctx.DB.getCalcDoneBH())
	assert.Equal(t, blockHeight1, ctx.DB.getPrevCalcDoneBH())

	ctx.DB.rollbackAccountDBBlockInfo()

	assert.Equal(t, blockHeight1, ctx.DB.getCalcDoneBH())
	assert.Equal(t, blockHeight1, ctx.DB.getPrevCalcDoneBH())
	assert.Equal(t, blockHeight1, ctx.DB.getCalculatingBH())
}

func TestContext_RollbackAccountDB(t *testing.T) {
	dbCount := 4
	ctx := initTest(dbCount)
	defer finalizeTest(ctx)

	crDB := ctx.DB.getCalculateResultDB()
	crBucket, err := crDB.GetBucket(db.PrefixCalcResult)

	ia := makeIA()


	// write to query DB
	qDB := ctx.DB.getQueryDB(ia.Address)
	bucket, _ := qDB.GetBucket(db.PrefixIScore)
	err = bucket.Set(ia.ID(), ia.Bytes())
	assert.NoError(t, err)

	// emulate calculation process
	prevBlockHeight := uint64(5)
	ctx.DB.setCalcDoneBH(prevBlockHeight)
	blockHeight := uint64(10)
	ctx.DB.setCalculatingBH(blockHeight)
	ctx.DB.writeToDB()
	ctx.DB.toggleAccountDB(blockHeight)

	// Rollback without backup account DB
	err = ctx.DB.rollbackAccountDB(0)
	assert.Error(t, err)

	// reset account DB to make backup account DB
	err = ctx.DB.resetAccountDB(blockHeight, ctx.DB.getCalcDoneBH())
	assert.NoError(t, err)
	WriteCalculationResult(crDB, blockHeight, nil, nil)
	ctx.DB.setCalcDoneBH(blockHeight)
	ctx.DB.writeToDB()
	assert.Equal(t, prevBlockHeight, ctx.DB.getPrevCalcDoneBH())
	assert.Equal(t, blockHeight, ctx.DB.getCalcDoneBH())

	// read from query DB
	qDB = ctx.DB.getQueryDB(ia.Address)
	bucket, _ = qDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(ia.ID())
	assert.Nil(t, bs)

	// no need to Rollback with blockHeight >= ctx.DB.Info.CalcBlockHeight
	err = ctx.DB.rollbackAccountDB(blockHeight)

	// backup account DB remains
	backupName := fmt.Sprintf(BackupDBNameFormat, blockHeight, 1)
	stat, err := os.Stat(filepath.Join(ctx.DB.info.DBRoot, backupName))
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())

	// check block height and block hash
	assert.Equal(t, blockHeight, ctx.DB.getCalcDoneBH())

	// valid Rollback
	err = ctx.DB.rollbackAccountDB(0)
	assert.NoError(t, err)

	// backup account DB was deleted
	backupName = fmt.Sprintf(BackupDBNameFormat, blockHeight, 1)
	_, err = os.Stat(filepath.Join(ctx.DB.info.DBRoot, backupName))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// check calculation result
	bs, _ = crBucket.Get(common.Uint64ToBytes(prevBlockHeight))
	assert.Nil(t, bs)

	// check Rollback block height and block hash
	assert.Equal(t, prevBlockHeight, ctx.DB.getCalcDoneBH())
	assert.Equal(t, prevBlockHeight, ctx.DB.getPrevCalcDoneBH())

	// read from query DB
	qDB = ctx.DB.getQueryDB(ia.Address)
	bucket, _ = qDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(ia.ID())
	assert.NotNil(t, bs)
	assert.Equal(t, ia.Bytes(), bs)

	// read from calculate DB
	cDB := ctx.DB.getCalculateDB(ia.Address)
	bucket, _ = cDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(ia.ID())
	assert.Nil(t, bs)
}

func TestContext_WriteToDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	original, _ := ctx.DB.info.Bytes()

	ctx.DB.writeToDB()

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixManagement)
	bs, _ := bucket.Get(ctx.DB.info.ID())
	assert.NotNil(t, bs)
	assert.Equal(t, 0, bytes.Compare(original, bs))
}

func TestContext_GetGVByBlockHeight(t *testing.T) {
	const (
		gvBH0 uint64 = 10
		gvBH1 uint64 = 20
	)
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// Insert initial GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = gvBH0
	gv.RewardRep = *common.NewHexIntFromUint64(gvBH0)
	ctx.GV = append(ctx.GV, gv)

	gv = new(GovernanceVariable)
	gv.BlockHeight = gvBH1
	gv.RewardRep = *common.NewHexIntFromUint64(gvBH1)
	ctx.GV = append(ctx.GV, gv)

	// check
	gv = ctx.getGVByBlockHeight(gvBH0 - 1)
	assert.Nil(t, gv)
	gv = ctx.getGVByBlockHeight(gvBH0)
	assert.Nil(t, gv)
	gv = ctx.getGVByBlockHeight(gvBH0 + 1)
	assert.Equal(t, gvBH0, gv.BlockHeight)
	gv = ctx.getGVByBlockHeight(gvBH1)
	assert.Equal(t, gvBH0, gv.BlockHeight)
	gv = ctx.getGVByBlockHeight(gvBH1 + 1)
	assert.Equal(t, gvBH1, gv.BlockHeight)
	gv = ctx.getGVByBlockHeight(gvBH1 + 100)
	assert.Equal(t, gvBH1, gv.BlockHeight)
}
