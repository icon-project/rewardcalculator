package core

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
)

func makeDBInfo() *DBInfo {
	dbInfo := new(DBInfo)

	dbInfo.DBRoot = testDBDir
	dbInfo.DBType = string(db.GoLevelDBBackend)
	dbInfo.DBCount = 1
	dbInfo.QueryDBIsZero = false
	dbInfo.CalcDone = iaBlockHeight
	dbInfo.Calculating = iaBlockHeight

	return dbInfo
}

func TestDBMNGDBInfo_ID(t *testing.T) {
	dbInfo := makeDBInfo()

	assert.Equal(t, []byte(""), dbInfo.ID())
}

func TestDBMNGDBInfo_BytesAndSetBytes(t *testing.T) {
	dbInfo := makeDBInfo()

	var dbInfoNew DBInfo

	bs, _ := dbInfo.Bytes()
	dbInfoNew.SetBytes(bs)

	assert.Equal(t, dbInfo.DBCount, dbInfoNew.DBCount)
	assert.Equal(t, dbInfo.QueryDBIsZero, dbInfoNew.QueryDBIsZero)
	assert.Equal(t, dbInfo.CalcDone, dbInfoNew.CalcDone)
	assert.Equal(t, dbInfo.Calculating, dbInfoNew.Calculating)
	bsNew, _ := dbInfoNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func TestDBMNGDBInfo_NewDBInfo(t *testing.T) {
	mngDB := db.Open(testDBDir, string(db.GoLevelDBBackend), testDB)
	defer mngDB.Close()
	defer os.RemoveAll(testDBDir)

	// make new DB
	dbInfo, err := NewDBInfo(mngDB, testDBDir, string(db.GoLevelDBBackend), testDB, 1)

	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(testDBDir, testDB), dbInfo.DBRoot)
	assert.Equal(t, string(db.GoLevelDBBackend), dbInfo.DBType)
	assert.Equal(t, 1, dbInfo.DBCount)
	assert.False(t, dbInfo.QueryDBIsZero)
	assert.True(t, dbInfo.Current.checkValue(uint64(0), zeroBlockHash))
	assert.Equal(t, uint64(0), dbInfo.CalcDone)
	assert.Equal(t, uint64(0), dbInfo.PrevCalcDone)
	assert.Equal(t, uint64(0), dbInfo.Calculating)

	bucket, _ := mngDB.GetBucket(db.PrefixManagement)
	bsNew, err := bucket.Get(dbInfo.ID())
	assert.NotNil(t, bsNew)
	assert.Nil(t, err)
	bs, _ := dbInfo.Bytes()
	assert.Equal(t, bs, bsNew)

	// update DB Info
	dbInfo.DBCount = 2
	dbInfo.QueryDBIsZero = true
	dbInfo.CalcDone = 200
	dbInfo.PrevCalcDone = 100
	dbInfo.Calculating = dbInfo.CalcDone

	bs, _ = dbInfo.Bytes()
	bucket.Set(dbInfo.ID(), bs)

	// read from DB
	dbInfo1, err := NewDBInfo(mngDB, testDBDir, string(db.GoLevelDBBackend), testDB, 10)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(testDBDir, testDB), dbInfo1.DBRoot)
	assert.Equal(t, string(db.GoLevelDBBackend), dbInfo1.DBType)
	assert.Equal(t, dbInfo.DBCount, dbInfo1.DBCount)
	assert.Equal(t, dbInfo.QueryDBIsZero, dbInfo1.QueryDBIsZero)
	assert.True(t, dbInfo.Current.equal(&dbInfo1.Current))
	assert.Equal(t, dbInfo.CalcDone, dbInfo1.CalcDone)
	assert.Equal(t, dbInfo.PrevCalcDone, dbInfo1.PrevCalcDone)
	assert.Equal(t, dbInfo.Calculating, dbInfo1.Calculating)
}


func makeGV(blockHeight uint64) *GovernanceVariable {
	gv := new(GovernanceVariable)

	gv.BlockHeight = blockHeight
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(2)
	gv.setReward()

	return gv
}

func TestDBMNGGV_ID(t *testing.T) {
	gv := makeGV(iaBlockHeight)

	id := gv.ID()

	assert.Equal(t, iaBlockHeight, common.BytesToUint64(id))
	assert.Equal(t, 8, len(id))
}

func TestDBMNGGV_BytesAndSetBytes(t *testing.T) {
	gv := makeGV(iaBlockHeight)

	var gvNew GovernanceVariable

	bs, _ := gv.Bytes()
	gvNew.SetBytes(bs)

	assert.Equal(t, 0, gv.MainPRepCount.Cmp(&gvNew.MainPRepCount.Int))
	assert.Equal(t, 0, gv.SubPRepCount.Cmp(&gvNew.SubPRepCount.Int))
	assert.Equal(t, 0, gv.BlockProduceReward.Cmp(&gvNew.BlockProduceReward.Int))
	assert.Equal(t, 0, gv.PRepReward.Cmp(&gvNew.PRepReward.Int))
	assert.Equal(t, 0, gv.CalculatedIncentiveRep.Cmp(&gvNew.CalculatedIncentiveRep.Int))
	assert.Equal(t, 0, gv.RewardRep.Cmp(&gvNew.RewardRep.Int))
	bsNew, _ := gvNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func TestDBMNGGV_LoadGovernanceVariable(t *testing.T) {
	mngDB := db.Open(testDBDir, string(db.GoLevelDBBackend), testDB)
	defer mngDB.Close()
	defer os.RemoveAll(testDBDir)
	bucket, _ := mngDB.GetBucket(db.PrefixGovernanceVariable)

	// write governance variable to DB
	gvList := make([]*GovernanceVariable, 0)
	gv := makeGV(iaBlockHeight)
	gvList = append(gvList, gv)
	gv = makeGV(iaBlockHeight + 100)
	gvList = append(gvList, gv)
	gv = makeGV(iaBlockHeight + 200)
	gvList = append(gvList, gv)

	for _, gv = range gvList {
		bs, _ := gv.Bytes()
		bucket.Set(gv.ID(), bs)
	}

	gvListNew, err := LoadGovernanceVariable(mngDB, iaBlockHeight + 101)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(gvListNew))
	assert.Equal(t, gvList[1].BlockHeight, gvListNew[0].BlockHeight)
	assert.Equal(t, gvList[2].BlockHeight, gvListNew[1].BlockHeight)
}

func TestDBMNGGV_NewGVFromIISS(t *testing.T) {
	iissGV := makeIISSGV()

	gv := NewGVFromIISS(iissGV)

	assert.Equal(t, iissGV.BlockHeight, gv.BlockHeight)
	assert.Equal(t, iissGV.MainPRepCount, gv.MainPRepCount.Uint64())
	assert.Equal(t, iissGV.SubPRepCount, gv.SubPRepCount.Uint64())
	assert.Equal(t, iissGV.IncentiveRep, gv.CalculatedIncentiveRep.Uint64())
	assert.Equal(t, iissGV.RewardRep, gv.RewardRep.Uint64())
}

type GVDataV1 struct {
	CalculatedIncentiveRep common.HexInt
	RewardRep              common.HexInt
}

func (gv *GVDataV1) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&gv); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func TestDBMNGGV_BackwardCompatibility(t *testing.T) {
	var gvV1 GVDataV1
	gvV1.CalculatedIncentiveRep.SetUint64(uint64(123))
	gvV1.RewardRep.SetUint64(uint64(456))

	bs, err := gvV1.Bytes()
	assert.Nil(t, err)

	var gv GovernanceVariable
	err = gv.SetBytes(bs)
	assert.Nil(t, err)
	assert.Equal(t, gvV1.CalculatedIncentiveRep.Uint64(), gv.CalculatedIncentiveRep.Uint64())
	assert.Equal(t, gvV1.RewardRep.Uint64(), gv.RewardRep.Uint64())
	assert.Equal(t, uint64(0), gv.MainPRepCount.Uint64())
	assert.Equal(t, uint64(0), gv.SubPRepCount.Uint64())
}

func makePRep() *PRep{
	pRep := new(PRep)

	pRep.BlockHeight = iaBlockHeight
	pRep.TotalDelegation.SetUint64(100)
	pRep.List = make([]PRepDelegationInfo, 0)

	pRepDelInfo := new(PRepDelegationInfo)
	pRepDelInfo.Address = *common.NewAddressFromString("hx1")
	pRepDelInfo.DelegatedAmount.SetUint64(40)
	pRep.List = append(pRep.List, *pRepDelInfo)

	pRepDelInfo = new(PRepDelegationInfo)
	pRepDelInfo.Address = *common.NewAddressFromString("hx2")
	pRepDelInfo.DelegatedAmount.SetUint64(60)
	pRep.List = append(pRep.List, *pRepDelInfo)

	return pRep
}

func TestDBMNGPRep_ID(t *testing.T) {
	pRep := makePRep()

	id := pRep.ID()

	assert.Equal(t, 8, len(id))
	assert.Equal(t, iaBlockHeight, common.BytesToUint64(id))
}

func TestDBMNGPRep_BytesAndSetBytes(t *testing.T) {
	pRep := makePRep()

	var pRepNew PRep

	bs, _ := pRep.Bytes()
	pRepNew.SetBytes(bs)

	assert.Equal(t, 0, pRep.TotalDelegation.Cmp(&pRepNew.TotalDelegation.Int))
	assert.Equal(t, len(pRep.List), len(pRepNew.List))
	bsNew, _ := pRepNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func TestDBMNGPRep_LoadPRep(t *testing.T) {
	mngDB := db.Open(testDBDir, string(db.GoLevelDBBackend), testDB)
	defer mngDB.Close()
	defer os.RemoveAll(testDBDir)
	bucket, _ := mngDB.GetBucket(db.PrefixPRep)

	// write PRep to DB
	pRepList := make([]*PRep, 0)
	pRep := makePRep()
	pRepList = append(pRepList, pRep)
	pRep = makePRep()
	pRep.BlockHeight = iaBlockHeight + 1
	pRepList = append(pRepList, pRep)
	pRep = makePRep()
	pRep.BlockHeight = iaBlockHeight + 2
	pRepList = append(pRepList, pRep)

	for _, pRep = range pRepList {
		bs, _ := pRep.Bytes()
		bucket.Set(pRep.ID(), bs)
	}

	pRepListNew, err := LoadPRep(mngDB)

	assert.Nil(t, err)
	assert.Equal(t, len(pRepList), len(pRepListNew))
	for i, pRepNew := range pRepListNew {
		pRep = pRepList[i]
		assert.Equal(t, pRep.BlockHeight, pRepNew.BlockHeight)
		assert.Equal(t, 0, pRep.TotalDelegation.Cmp(&pRepNew.TotalDelegation.Int))
		assert.Equal(t, len(pRep.List), len(pRepNew.List))
		bs, _ := pRep.Bytes()
		bsNew, _ := pRepNew.Bytes()
		assert.Equal(t, bs, bsNew)
	}
}



func makePRepCandidate(addr string) *PRepCandidate {
	pc := new(PRepCandidate)

	pc.Address = *common.NewAddressFromString(addr)
	pc.Start = iaBlockHeight
	pc.End = 0

	return pc
}

func TestDBMNGPRepCandidate_ID(t *testing.T) {
	pc := makePRepCandidate("hx1")

	assert.Equal(t, pc.Address.Bytes(), pc.ID())
}

func TestDBMNGPRepCandidate_BytesAndSetBytes(t *testing.T) {
	pc := makePRepCandidate("hx1")

	var pcNew PRepCandidate

	bs, _ := pc.Bytes()
	pcNew.SetBytes(bs)

	assert.Equal(t, pc.Start, pcNew.Start)
	assert.Equal(t, pc.End, pcNew.End)
}

func TestDBMNGPRepCandidate_LoadPRepCandidate(t *testing.T) {
	mngDB := db.Open(testDBDir, string(db.GoLevelDBBackend), testDB)
	defer mngDB.Close()
	defer os.RemoveAll(testDBDir)
	bucket, _ := mngDB.GetBucket(db.PrefixPRepCandidate)

	// write PRep candidates to DB
	pcList := make([]*PRepCandidate, 0)
	pc := makePRepCandidate("hx1")
	pcList = append(pcList, pc)
	pc = makePRepCandidate("hx2")
	pc.Start = iaBlockHeight + 100
	pcList = append(pcList, pc)
	pc = makePRepCandidate("hx3")
	pc.End = iaBlockHeight + 200
	pcList = append(pcList, pc)

	for _, pc = range pcList {
		bs, _ := pc.Bytes()
		bucket.Set(pc.ID(), bs)
	}

	pcMap, err := LoadPRepCandidate(mngDB)

	assert.Nil(t, err)
	assert.Equal(t, len(pcList), len(pcMap))
	for _, pc := range pcList {
		pcNew, ok := pcMap[pc.Address]
		assert.True(t, ok)
		assert.True(t, pc.Address.Equal(&pcNew.Address))
		assert.Equal(t, pc.Start, pcNew.Start)
		assert.Equal(t, pc.End, pcNew.End)
	}
}

type dataV1 struct {
	DBCount int
	BlockHeight uint64
	QueryDBIsZero bool
}

func TestDBMNGDBInfo_backwardCompatibility(t *testing.T) {
	v1 := dataV1{16, 1000, true }
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&v1); err != nil {
		log.Printf("Failed to marshal v1. %v", err)
		return
	} else {
		bytes = bs
	}

	var v2 DBInfoDataV2

	_, err := codec.UnmarshalFromBytes(bytes, &v2)
	if err != nil {
		log.Printf("Failed to unmarshal v1 with v2. %v", err)
		return
	}
}
