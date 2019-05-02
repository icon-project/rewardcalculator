package rewardcalculator

import (
	"bytes"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

var testDir string

func initTest(dbCount int) *Context{
	var err error
	testDir, err = ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDir)

	ctx, _ := NewContext(testDir, string(db.GoLevelDBBackend), "test", dbCount)

	return ctx
}

func finalizeTest() {
	defer os.RemoveAll(testDir)
}

func TestContext_NewContext(t *testing.T) {
	const dbCount int = 16
	ctx := initTest(dbCount)
	defer finalizeTest()
	assert.NotNil(t, ctx)

	assert.NotNil(t, ctx.DB)
	assert.NotNil(t, ctx.DB.info)
	assert.Equal(t, dbCount, ctx.DB.info.DBCount)
	assert.Equal(t, uint64(0), ctx.DB.info.BlockHeight)
	assert.False(t, ctx.DB.info.QueryDBIsZero)
	assert.Equal(t, dbCount, len(ctx.DB.Account0))
	assert.Equal(t, dbCount, len(ctx.DB.Account1))
	assert.NotNil(t, ctx.DB.claim)

	assert.NotNil(t, ctx.GV)
	assert.NotNil(t, ctx.PRep)
	assert.NotNil(t, ctx.PRepCandidates)
	assert.NotNil(t, ctx.preCommit)
}

func TestContext_UpdateGovernanceVariable(t *testing.T) {
	const (
		ctxBlockHeight uint64 = 100
	)
	ctx := initTest(1)
	defer finalizeTest()

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixGovernanceVariable)

	// Set Block height
	ctx.DB.info.BlockHeight = ctxBlockHeight

	// Insert initial GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = ctxBlockHeight - 20
	gv.RewardRep = *common.NewHexIntFromUint64(ctxBlockHeight - 20)
	ctx.GV = append(ctx.GV, gv)
	value, _ := gv.Bytes()
	bucket.Set(gv.ID(), value)

	gv = new(GovernanceVariable)
	gv.BlockHeight = ctxBlockHeight - 10
	gv.RewardRep = *common.NewHexIntFromUint64(ctxBlockHeight - 10)
	ctx.GV = append(ctx.GV, gv)
	value, _ = gv.Bytes()
	bucket.Set(gv.ID(), value)

	// make IISSGovernanceVariable list
	gvList := make([]*IISSGovernanceVariable, 0)
	iissGV := new(IISSGovernanceVariable)
	iissGV.BlockHeight = ctxBlockHeight + 10
	iissGV.RewardRep = ctxBlockHeight + 10
	gvList = append(gvList, iissGV)
	iissGV = new(IISSGovernanceVariable)
	iissGV.BlockHeight = ctxBlockHeight + 20
	iissGV.RewardRep = ctxBlockHeight + 20
	gvList = append(gvList, iissGV)

	// update GV
	ctx.UpdateGovernanceVariable(gvList)

	// check - len
	assert.Equal(t, len(gvList) + 1, len(ctx.GV))

	// check - values

	// In memory
	assert.Equal(t, ctxBlockHeight - 10, ctx.GV[0].BlockHeight)
	// In DB
	bs, _ := bucket.Get(ctx.GV[0].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[0].RewardRep.Cmp(&gv.RewardRep.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight + 10, ctx.GV[1].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.GV[1].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[1].RewardRep.Cmp(&gv.RewardRep.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight + 20, ctx.GV[2].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.GV[2].ID())
	assert.NotNil(t, bs)
	gv.SetBytes(bs)
	assert.Equal(t, 0, ctx.GV[2].RewardRep.Cmp(&gv.RewardRep.Int))
}

func TestContext_UpdatePRep(t *testing.T) {
	const (
		ctxBlockHeight uint64 = 100
	)
	ctx := initTest(1)
	defer finalizeTest()

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRep)

	// Set Block height
	ctx.DB.info.BlockHeight = ctxBlockHeight

	// Insert initial PRep
	pRep := new(PRep)
	pRep.BlockHeight = ctxBlockHeight - 20
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight - 20)
	ctx.PRep = append(ctx.PRep, pRep)
	value, _ := pRep.Bytes()
	bucket.Set(pRep.ID(), value)

	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight - 10
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight - 10)
	ctx.PRep = append(ctx.PRep, pRep)
	value, _ = pRep.Bytes()
	bucket.Set(pRep.ID(), value)

	// make IISS list
	prepList := make([]*PRep, 0)
	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight + 10
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight + 10)
	prepList = append(prepList, pRep)
	pRep = new(PRep)
	pRep.BlockHeight = ctxBlockHeight + 20
	pRep.TotalDelegation = *common.NewHexIntFromUint64(ctxBlockHeight + 20)
	prepList = append(prepList, pRep)

	// update PRep
	ctx.UpdatePRep(prepList)

	// check - len
	assert.Equal(t, len(prepList) + 1, len(ctx.PRep))

	// check - values

	// In memory
	assert.Equal(t, ctxBlockHeight - 10, ctx.PRep[0].BlockHeight)
	// In DB
	bs, _ := bucket.Get(ctx.PRep[0].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[0].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight + 10, ctx.PRep[1].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.PRep[1].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[1].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))

	// In memory
	assert.Equal(t, ctxBlockHeight + 20, ctx.PRep[2].BlockHeight)
	// In DB
	bs, _ = bucket.Get(ctx.PRep[2].ID())
	assert.NotNil(t, bs)
	pRep.SetBytes(bs)
	assert.Equal(t, 0, ctx.PRep[2].TotalDelegation.Cmp(&pRep.TotalDelegation.Int))
}

func TestContext_UpdatePRepCandidate(t *testing.T) {
	const (
		regPRepABH uint64 = 1
		unRegPRepABH uint64 = 100
		NotUnRegBH uint64 = 0
	)
	ctx := initTest(1)
	defer finalizeTest()

	bucket, _ := ctx.DB.management.GetBucket(db.PrefixPRepCandidate)

	pRepA := common.NewAddressFromString("hxa")

	// 1. register PRep candidate A
	txList := make([]*IISSTX, 0)
	tx := new(IISSTX)
	tx.Index = 0
	tx.DataType = TXDataTypePrepReg
	tx.BlockHeight = regPRepABH
	tx.Address = *pRepA
	txList = append(txList, tx)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(txList)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, NotUnRegBH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ := bucket.Get(pRepA.Bytes())
	var pRep PRepCandidate
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, NotUnRegBH, pRep.End)

	// 2. register PRep candidate A with different block height again
	txList = make([]*IISSTX, 0)
	tx = new(IISSTX)
	tx.Index = 0
	tx.DataType = TXDataTypePrepReg
	tx.BlockHeight = regPRepABH + 1
	tx.Address = *pRepA
	txList = append(txList, tx)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(txList)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, NotUnRegBH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ = bucket.Get(pRepA.Bytes())
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, NotUnRegBH, pRep.End)

	// 3. unregister PRep candidate A
	txList = make([]*IISSTX, 0)
	tx = new(IISSTX)
	tx.Index = 0
	tx.DataType = TXDataTypePrepUnReg
	tx.BlockHeight = unRegPRepABH
	tx.Address = *pRepA
	txList = append(txList, tx)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(txList)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, unRegPRepABH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ = bucket.Get(pRepA.Bytes())
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, unRegPRepABH, pRep.End)

	// 4. unregister PRep candidate A with different block height again
	txList = make([]*IISSTX, 0)
	tx = new(IISSTX)
	tx.Index = 0
	tx.DataType = TXDataTypePrepUnReg
	tx.BlockHeight = unRegPRepABH + 1
	tx.Address = *pRepA
	txList = append(txList, tx)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(txList)

	// check - in memory
	assert.Equal(t, regPRepABH, ctx.PRepCandidates[tx.Address].Start)
	assert.Equal(t, unRegPRepABH, ctx.PRepCandidates[tx.Address].End)

	// check - in DB
	bs, _ = bucket.Get(pRepA.Bytes())
	pRep.SetBytes(bs)

	assert.Equal(t, regPRepABH, pRep.Start)
	assert.Equal(t, unRegPRepABH, pRep.End)

	// 5. unregister PRep candidate B
	txList = make([]*IISSTX, 0)
	tx = new(IISSTX)
	tx.Index = 0
	tx.DataType = TXDataTypePrepUnReg
	tx.BlockHeight = unRegPRepABH
	tx.Address = *common.NewAddressFromString("hxb")
	txList = append(txList, tx)

	// update P-Rep candidate
	ctx.UpdatePRepCandidate(txList)

	// check - in memory
	_, ok := ctx.PRepCandidates[tx.Address]
	assert.False(t, ok)

	// check - in DB
	bs, _ = bucket.Get(tx.Address.Bytes())
	assert.Nil(t, bs)
}

func TestContext_GetQueryDBList(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest()

	ctx.DB.info.QueryDBIsZero = true
	assert.Equal(t, ctx.DB.Account0, ctx.DB.getQueryDBList())
	ctx.DB.info.QueryDBIsZero = false
	assert.Equal(t, ctx.DB.Account1, ctx.DB.getQueryDBList())
}

func TestContext_GetCalcDBList(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest()

	ctx.DB.info.QueryDBIsZero = true
	assert.Equal(t, ctx.DB.Account1, ctx.DB.GetCalcDBList())
	ctx.DB.info.QueryDBIsZero = false
	assert.Equal(t, ctx.DB.Account0, ctx.DB.GetCalcDBList())
}

func TestContext_ToggleAccountDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest()

	original := ctx.DB.info.QueryDBIsZero

	ctx.DB.toggleAccountDB()
	assert.Equal(t, !original, ctx.DB.info.QueryDBIsZero)
	ctx.DB.toggleAccountDB()
	assert.Equal(t, original, ctx.DB.info.QueryDBIsZero)
}

func TestContext_ResetCalcDB(t *testing.T) {
	dbCount := 4
	ctx := initTest(dbCount)
	defer finalizeTest()

	qDBList := ctx.DB.getQueryDBList()
	cDBList := ctx.DB.GetCalcDBList()

	ctx.DB.resetCalcDB()

	assert.Equal(t, qDBList, ctx.DB.getQueryDBList())
	assert.NotEqual(t, cDBList, ctx.DB.GetCalcDBList())
	assert.Equal(t, dbCount, len(ctx.DB.GetCalcDBList()))
}

func TestContext_WriteToDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest()

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
	defer finalizeTest()

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
