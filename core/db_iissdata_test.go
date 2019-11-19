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

func makeHeader(blockHeight uint64) *IISSHeader {
	header := new(IISSHeader)

	header.Version = IISSDataVersion
	header.BlockHeight = blockHeight
	header.Revision = IISSDataRevisionDefault

	return header
}

func TestDBIISSHeader_ID(t *testing.T) {
	header := makeHeader(iaBlockHeight)

	assert.Equal(t, []byte(""), header.ID())
}

func TestDBIISSHeader_BytesAndSetBytes(t *testing.T) {
	header := makeHeader(iaBlockHeight)

	var headerNew IISSHeader

	bs, _ := header.Bytes()
	headerNew.SetBytes(bs)

	assert.Equal(t, IISSDataVersion, headerNew.Version)
	assert.Equal(t, header.BlockHeight, headerNew.BlockHeight)
	bsNew, _ := headerNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeHeader(dbDir string, dbName string, blockHeight uint64) (*IISSHeader, db.Database) {
	header := makeHeader(blockHeight)

	// write IISS header
	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSHeader)
	bs, _ := header.Bytes()
	bucket.Set(header.ID(), bs)

	return header, iissDB
}

func TestDBIISSHeader_loadIISSHeader(t *testing.T) {
	// write IISS header to DB
	header, iissDB := writeHeader(testDBDir, testDB, iaBlockHeight)
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

type IISSHeaderV1 struct {
	Version     uint64
	BlockHeight uint64
}

func (ih *IISSHeaderV1) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(ih); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func TestDBIISSHeader_BackwardCompatibility(t *testing.T) {
	var headerV1 IISSHeaderV1
	headerV1.Version = 1
	headerV1.BlockHeight = 2345

	bs, err := headerV1.Bytes()
	assert.Nil(t, err)

	var ih IISSHeader
	ih.SetBytes(bs)
	assert.Equal(t, headerV1.Version, ih.Version)
	assert.Equal(t, headerV1.BlockHeight, ih.BlockHeight)
	assert.Equal(t, IISSDataRevisionDefault, ih.Revision)
}

const (
	gvRewardRep uint64 = 100
	gvIncentiveRep uint64 = 1
)

func makeIISSGV() *IISSGovernanceVariable {
	gv := new(IISSGovernanceVariable)

	gv.BlockHeight = iaBlockHeight
	gv.MainPRepCount = NumMainPRep
	gv.SubPRepCount = NumSubPRep
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
	gvNew.SetBytes(bs, IISSDataVersion)

	assert.Equal(t, gv.MainPRepCount, gvNew.MainPRepCount)
	assert.Equal(t, gv.SubPRepCount, gvNew.SubPRepCount)
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
	gv.RewardRep = gvRewardRep + 100
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
	gvListNew, err := loadIISSGovernanceVariable(iissDB, IISSDataVersion)

	// check
	assert.Nil(t, err)
	assert.Equal(t, len(gvList), len(gvListNew))
	for i := range gvListNew {
		assert.Equal(t, gvList[i].BlockHeight, gvListNew[i].BlockHeight)
		assert.Equal(t, gvList[i].MainPRepCount, gvListNew[i].MainPRepCount)
		assert.Equal(t, gvList[i].SubPRepCount, gvListNew[i].SubPRepCount)
		assert.Equal(t, gvList[i].RewardRep, gvListNew[i].RewardRep)
		assert.Equal(t, gvList[i].IncentiveRep, gvListNew[i].IncentiveRep)
		bs, _ := gvList[i].Bytes()
		bsNew, _ := gvListNew[i].Bytes()
		assert.Equal(t, bs, bsNew)
	}
}

type IISSGVDataV1 struct {
	IncentiveRep uint64
	RewardRep    uint64
}

func (gv *IISSGVDataV1) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(gv); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func TestDBIISSGovernanceVariable_BackwardCompatibility(t *testing.T) {
	var gvV1 IISSGVDataV1
	gvV1.IncentiveRep = 123
	gvV1.RewardRep = 456

	bs, err := gvV1.Bytes()
	assert.Nil(t, err)

	var gv IISSGovernanceVariable
	gv.SetBytes(bs, 1)
	assert.Equal(t, gvV1.IncentiveRep, gv.IncentiveRep)
	assert.Equal(t, gvV1.RewardRep, gv.RewardRep)
	assert.Equal(t, uint64(0), gv.MainPRepCount)
	assert.Equal(t, uint64(0), gv.SubPRepCount)
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

const (
	txIndex uint64 = 0
)

func makeIISSTX(dataType uint64, addr string, dgDataSlice []DelegateData) *IISSTX {
	tx := new(IISSTX)

	tx.Index = txIndex
	tx.BlockHeight = iaBlockHeight
	tx.Address = *common.NewAddressFromString(addr)
	tx.DataType = dataType
	if dataType == TXDataTypeDelegate {
		tx.Data = new(codec.TypedObj)

		if dgDataSlice != nil {
			delegation := make([]interface{}, 0)

			for _, dg := range dgDataSlice {
				dgData := addDelegationData(dg.Address, dg.Delegate.Uint64())
				delegation = append(delegation, dgData)
			}

			tx.Data, _ = common.EncodeAny(delegation)
		} else {
			tx.Data.Type = codec.TypeNil
			tx.Data.Object = []byte("")
		}
	}
	return tx
}

func addDelegationData(address common.Address, amount uint64) []interface{} {
	dgData := make([]interface{}, 0)
	dgData = append(dgData, &address)
	dgData = append(dgData, amount)

	return dgData
}

func TestDBIISSTX_ID(t *testing.T) {
	dgDataSlice := []DelegateData {
		{*common.NewAddressFromString(delegationAddress), *common.NewHexIntFromUint64(MinDelegation)},
	}
	tx := makeIISSTX(TXDataTypeDelegate, genAddress, dgDataSlice)

	id := tx.ID()

	assert.Equal(t, txIndex, common.BytesToUint64(id))
	assert.Equal(t, 8, len(id))
}

func TestDBIISSTX_BytesAndSetBytes(t *testing.T) {
	dgDataSlice := []DelegateData {
		{*common.NewAddressFromString(delegationAddress), *common.NewHexIntFromUint64(MinDelegation)},
	}
	tx := makeIISSTX(TXDataTypeDelegate, genAddress, dgDataSlice)

	var txNew IISSTX

	bs, _ := tx.Bytes()

	txNew.SetBytes(bs)

	assert.Equal(t, tx.BlockHeight, txNew.BlockHeight)
	assert.True(t, tx.Address.Equal(&txNew.Address))
	assert.Equal(t, tx.DataType, txNew.DataType)
	bsNew, _ := txNew.Bytes()
	assert.Equal(t, bs, bsNew)
}

func writeTX(iissDB db.Database, txList []*IISSTX) {
	bucket, _ := iissDB.GetBucket(db.PrefixIISSTX)
	for _, tx := range txList {
		bs, _ := tx.Bytes()
		bucket.Set(tx.ID(), bs)
	}
}

func writeIISSTX(dbDir string, dbName string) ([]*IISSTX, db.Database) {
	txList := make([]*IISSTX, 0)

	dgDataSlice := []DelegateData {
		{*common.NewAddressFromString(delegationAddress), *common.NewHexIntFromUint64(MinDelegation)},
	}
	tx := makeIISSTX(TXDataTypeDelegate, genAddress, dgDataSlice)
	txList = append(txList, tx)

	tx = makeIISSTX(TXDataTypePrepReg, genAddress, nil)
	tx.Index += 1
	tx.BlockHeight = iaBlockHeight + 100
	txList = append(txList, tx)


	tx = makeIISSTX(TXDataTypePrepUnReg, genAddress, nil)
	tx.Index += 2
	tx.BlockHeight = iaBlockHeight + 200
	txList = append(txList, tx)

	// write IISS TX
	iissDB := db.Open(dbDir, string(db.GoLevelDBBackend), dbName)
	writeTX(iissDB, txList)

	return txList, iissDB
}

func TestDBIISS_LoadIISSData(t *testing.T) {
	header, iissDB := writeHeader(testDBDir, testDB, iaBlockHeight)
	iissDB.Close()
	gvList, iissDB := writeIISSGV(testDBDir, testDB)
	iissDB.Close()
	_, iissDB = writeIISSBPInfo(testDBDir, testDB)
	iissDB.Close()
	_, iissDB = writeIISSTX(testDBDir, testDB)
	defer iissDB.Close()
	defer os.RemoveAll(testDBDir)

	headerNew, gvListNew, pRepListNew := LoadIISSData(iissDB)

	assert.NotNil(t, headerNew)
	bs, _ := header.Bytes()
	bsNew, _ := headerNew.Bytes()
	assert.Equal(t, bs, bsNew)

	assert.NotNil(t, gvListNew)
	assert.Equal(t, len(gvList), len(gvListNew))

	assert.NotNil(t, pRepListNew)
	assert.Equal(t, 0, len(pRepListNew))
}

func TestDBIISS_manageIISSData(t *testing.T) {
	rootPath, _ := filepath.Abs("./iissdata_test")
	os.MkdirAll(rootPath, os.ModePerm)
	iissPath := filepath.Join(rootPath, "current")
	os.MkdirAll(iissPath, os.ModePerm)
	finishPath := filepath.Join(rootPath, "finish_iiss")
	os.MkdirAll(finishPath, os.ModePerm)

	cleanupIISSData(iissPath)

	_, err := os.Stat(iissPath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(finishPath)
	assert.True(t, os.IsNotExist(err))
	backupPath := filepath.Join(rootPath, "finish_current")
	f, err := os.Stat(backupPath)
	assert.True(t, f.IsDir())

	os.RemoveAll(rootPath)
}