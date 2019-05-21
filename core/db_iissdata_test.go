package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
)

const (
	testDBDir = ".testDB"
	testDB = "test"
)

func makeHeader() *IISSHeader {
	header := new(IISSHeader)

	header.Version = Version
	header.BlockHeight = iaBlockHeight

	return header
}

func TestDBIISSHeader_ID(t *testing.T) {
	header := makeHeader()

	assert.Equal(t, []byte(""), header.ID())
}

func TestDBIISSHeader_BytesAndSetBytes(t *testing.T) {
	header := makeHeader()

	var headerNew IISSHeader

	bs, _ := header.Bytes()
	headerNew.SetBytes(bs)

	assert.Equal(t, Version, headerNew.Version)
	assert.Equal(t, header.BlockHeight, headerNew.BlockHeight)
	bsNew, _ := headerNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeHeader(dbDir string, dbName string) (*IISSHeader, db.Database) {
	header := makeHeader()

	// write IISS header
	iissDB := db.Open(testDBDir, string(db.GoLevelDBBackend), testDB)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSHeader)
	bs, _ := header.Bytes()
	bucket.Set(header.ID(), bs)

	return header, iissDB
}

func TestDBIISSHeader_loadIISSHeader(t *testing.T) {
	// write IISS header to DB
	header, iissDB := writeHeader(testDBDir, testDB)
	defer iissDB.Close()
	defer os.RemoveAll(testDBDir)

	// load IISS header
	headerNew, err := loadIISSHeader(iissDB)

	assert.Nil(t, err)
	assert.Equal(t, header.Version, headerNew.Version)
	assert.Equal(t, header.BlockHeight, headerNew.BlockHeight)
	bs, _ := header.Bytes()
	bsNew, _ := headerNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

const (
	gvRewardRep uint64 = 100
	gvIncentiveRep uint64 = 1
)

func makeIISSGV() *IISSGovernanceVariable {
	gv := new(IISSGovernanceVariable)

	gv.BlockHeight = iaBlockHeight
	gv.IncentiveRep = gvIncentiveRep
	gv.RewardRep = gvRewardRep

	return gv
}

func TestDBIISSGovernanceVariable_ID(t *testing.T) {
	gv := makeIISSGV()

	id := gv.ID()

	assert.Equal(t, iaBlockHeight, common.BytesToUint64(id))
	assert.Equal(t, 8, len(id))
}

func TestDBIISSGovernanceVariable_BytesAndSetBytes(t *testing.T) {
	gv := makeIISSGV()

	var gvNew IISSGovernanceVariable

	bs, _ := gv.Bytes()
	gvNew.SetBytes(bs)

	assert.Equal(t, gv.RewardRep, gvNew.RewardRep)
	assert.Equal(t, gv.IncentiveRep, gvNew.IncentiveRep)
	bsNew, _ := gvNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeIISSGV(dbDir string, dbName string) ([]*IISSGovernanceVariable, db.Database) {
	gvList := make([]*IISSGovernanceVariable, 0)
	gv := makeIISSGV()
	gvList = append(gvList, gv)
	gv = makeIISSGV()
	gv.BlockHeight = iaBlockHeight + 100
	gvList = append(gvList, gv)

	// write IISS governance variable
	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSGV)
	for _, gv = range gvList {
		bs, _ := gv.Bytes()
		bucket.Set(gv.ID(), bs)
	}

	return gvList, iissDB
}

func TestDBIISSGovernanceVariable_loadIISSGovernanceVariable(t *testing.T) {
	// write IISS governance variable to DB
	gvList, iissDB := writeIISSGV(testDBDir, testDB)
	defer iissDB.Close()
	defer os.RemoveAll(testDBDir)

	// load IISS governance variable
	gvListNew, err := loadIISSGovernanceVariable(iissDB)

	// check
	assert.Nil(t, err)
	assert.Equal(t, len(gvList), len(gvListNew))
	for i := range gvListNew {
		assert.Equal(t, gvList[i].BlockHeight, gvListNew[i].BlockHeight)
		assert.Equal(t, gvList[i].RewardRep, gvListNew[i].RewardRep)
		assert.Equal(t, gvList[i].IncentiveRep, gvListNew[i].IncentiveRep)
		bs, _ := gvList[i].Bytes()
		bsNew, _ := gvListNew[i].Bytes()
		assert.Equal(t, bs, bsNew)
	}
}

const (
	genAddress               = "hxa"
	validatorAddress         = "hx1"
)

func makeIISSBPInfo() *IISSBlockProduceInfo {
	bpInfo := new(IISSBlockProduceInfo)

	bpInfo.BlockHeight = iaBlockHeight
	bpInfo.Generator = *common.NewAddressFromString(genAddress)
	bpInfo.Validator = make([]common.Address, 0)
	bpInfo.Validator = append(bpInfo.Validator, *common.NewAddressFromString(validatorAddress))

	return bpInfo
}

func TestDBIISSBPInfo_ID(t *testing.T) {
	bpInfo := makeIISSBPInfo()

	id := bpInfo.ID()

	assert.Equal(t, iaBlockHeight, common.BytesToUint64(id))
	assert.Equal(t, 8, len(id))
}

func TestDBIISSBPInfo_BytesAndSetBytes(t *testing.T) {
	bpInfo := makeIISSBPInfo()

	var bpInfoNew IISSBlockProduceInfo

	bs, _ := bpInfo.Bytes()

	bpInfoNew.SetBytes(bs)

	assert.True(t, bpInfo.Generator.Equal(&bpInfoNew.Generator))
	assert.Equal(t, len(bpInfo.Validator), len(bpInfoNew.Validator))
	assert.True(t, bpInfo.Validator[0].Equal(&bpInfoNew.Validator[0]))
	bsNew, _ := bpInfoNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeIISSBPInfo(dbDir string, dbName string) ([]*IISSBlockProduceInfo, db.Database) {
	bpInfoList := make([]*IISSBlockProduceInfo, 0)
	bpInfo := makeIISSBPInfo()
	bpInfoList = append(bpInfoList, bpInfo)
	bpInfo = makeIISSBPInfo()
	bpInfo.BlockHeight = iaBlockHeight + 100
	bpInfoList = append(bpInfoList, bpInfo)

	// write IISS block produce Info.
	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSBPInfo)
	for _, bpInfo = range bpInfoList {
		bs, _ := bpInfo.Bytes()
		bucket.Set(bpInfo.ID(), bs)
	}

	return bpInfoList, iissDB
}

func TestDBIISSBPInfo_loadIISSBlockProduceInfo(t *testing.T) {
	// write IISS block produce Info to DB
	bpInfoList, iissDB := writeIISSBPInfo(testDBDir, testDB)
	defer iissDB.Close()
	defer os.RemoveAll(testDBDir)

	// load IISS block produce Info.
	bpInfoListNew, err := loadIISSBlockProduceInfo(iissDB)

	// check
	assert.Nil(t, err)
	assert.Equal(t, len(bpInfoList), len(bpInfoListNew))
	for i := range bpInfoListNew {
		assert.Equal(t, bpInfoList[i].BlockHeight, bpInfoListNew[i].BlockHeight)
		assert.True(t, bpInfoList[i].Generator.Equal(&bpInfoListNew[i].Generator))
		assert.Equal(t, len(bpInfoList[i].Validator), len(bpInfoListNew[i].Validator))
		assert.True(t, bpInfoList[i].Validator[0].Equal(&bpInfoListNew[i].Validator[0]))
		bs, _ := bpInfoList[i].Bytes()
		bsNew, _ := bpInfoListNew[i].Bytes()
		assert.Equal(t, bs, bsNew)
	}
}

const (
	txIndex uint64 = 0
)

func makeIISSTX(dataType uint64) *IISSTX {
	tx := new(IISSTX)

	tx.Index = txIndex
	tx.BlockHeight = iaBlockHeight
	tx.Address = *common.NewAddressFromString(genAddress)
	tx.DataType = dataType
	if dataType == TXDataTypeDelegate {
		tx.Data = new(codec.TypedObj)

		delegation := make([]interface{}, 0)

		dgData := make([]interface{}, 0)
		dgData = append(dgData, common.NewAddressFromString(delegationAddress))
		dgData = append(dgData, MinDelegation)
		delegation = append(delegation, dgData)

		tx.Data, _ = common.EncodeAny(delegation)
	}
	return tx
}

func TestDBIISSTX_ID(t *testing.T) {
	tx := makeIISSTX(TXDataTypeDelegate)

	id := tx.ID()

	assert.Equal(t, txIndex, common.BytesToUint64(id))
	assert.Equal(t, 8, len(id))
}

func TestDBIISSTX_BytesAndSetBytes(t *testing.T) {
	tx := makeIISSTX(TXDataTypeDelegate)

	var txNew IISSTX

	bs, _ := tx.Bytes()

	txNew.SetBytes(bs)

	assert.Equal(t, tx.BlockHeight, txNew.BlockHeight)
	assert.True(t, tx.Address.Equal(&txNew.Address))
	assert.Equal(t, tx.DataType, txNew.DataType)
	bsNew, _ := txNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeIISSTX(dbDir string, dbName string) ([]*IISSTX, db.Database) {
	txList := make([]*IISSTX, 0)
	tx := makeIISSTX(TXDataTypeDelegate)
	txList = append(txList, tx)
	tx = makeIISSTX(TXDataTypePRepReg)
	tx.Index += 1
	tx.BlockHeight = iaBlockHeight + 100
	txList = append(txList, tx)
	tx = makeIISSTX(TXDataTypePRepUnReg)
	tx.Index += 2
	tx.BlockHeight = iaBlockHeight + 200
	txList = append(txList, tx)

	// write IISS TX
	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSTX)
	for _, tx = range txList {
		bs, _ := tx.Bytes()
		bucket.Set(tx.ID(), bs)
	}

	return txList, iissDB
}

func TestDBIISSTX_loadIISSTX(t *testing.T) {
	// write IISS TX to DB
	txList, iissDB := writeIISSTX(testDBDir, testDB)
	defer iissDB.Close()
	defer os.RemoveAll(testDBDir)

	// load IISS TX
	txListNew, err := loadIISSTX(iissDB)

	// check
	assert.Nil(t, err)
	assert.Equal(t, len(txList), len(txListNew))
	for i := range txListNew {
		assert.Equal(t, txList[i].Index, txListNew[i].Index)
		assert.Equal(t, txList[i].BlockHeight, txListNew[i].BlockHeight)
		assert.True(t, txList[i].Address.Equal(&txListNew[i].Address))
		assert.Equal(t, txList[i].DataType, txListNew[i].DataType)
		bs, _ := txList[i].Bytes()
		bsNew, _ := txListNew[i].Bytes()
		assert.Equal(t, bs, bsNew)
	}
}

func TestDBIISS_LoadIISSData(t *testing.T) {
	header, iissDB := writeHeader(testDBDir, testDB)
	iissDB.Close()
	gvList, iissDB := writeIISSGV(testDBDir, testDB)
	iissDB.Close()
	bpInfoList, iissDB := writeIISSBPInfo(testDBDir, testDB)
	iissDB.Close()
	txList, iissDB := writeIISSTX(testDBDir, testDB)
	iissDB.Close()
	defer os.RemoveAll(testDBDir)

	headerNew, gvListNew, bpInfoListNew, pRepListNew, txListNew := LoadIISSData(filepath.Join(testDBDir, testDB), false)

	assert.NotNil(t, headerNew)
	bs, _ := header.Bytes()
	bsNew, _ := headerNew.Bytes()
	assert.Equal(t, bs, bsNew)

	assert.NotNil(t, gvListNew)
	assert.Equal(t, len(gvList), len(gvListNew))

	assert.NotNil(t, bpInfoListNew)
	assert.Equal(t, len(bpInfoList), len(bpInfoListNew))

	assert.NotNil(t, pRepListNew)
	assert.Equal(t, 0, len(pRepListNew))

	assert.NotNil(t, txListNew)
	assert.Equal(t, len(txList), len(txListNew))
}
