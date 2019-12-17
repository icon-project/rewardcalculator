package core

import (
	"bytes"
	"encoding/json"
	"log"
	"math/big"
	"path/filepath"
	"sort"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)


const (
	MaxDBCount  int    = 256

	NumMainPRep uint64 = 22
	NumSubPRep  uint64 = 78
)

type DBInfoDataV1 struct {
	DBCount       int
	BlockHeight   uint64 // finish to calculate to this block height
	QueryDBIsZero bool
}

type BlockInfo struct {
	BlockHeight uint64
	BlockHash [BlockHashSize]byte
}

func (bi *BlockInfo) set(blockHeight uint64, blockHash []byte) {
	bi.BlockHeight = blockHeight
	copy(bi.BlockHash[:], blockHash)
}

func (bi *BlockInfo) checkValue(blockHeight uint64, blockHash []byte) bool {
	return bi.BlockHeight == blockHeight && bytes.Compare(bi.BlockHash[:], blockHash) == 0
}

func (bi *BlockInfo) equal(dst *BlockInfo) bool {
	return bi.BlockHeight == dst.BlockHeight && bytes.Compare(bi.BlockHash[:], dst.BlockHash[:]) == 0
}

type DBInfoDataV2 struct {
	DBCount       int
	QueryDBIsZero bool
	Current       BlockInfo // Latest COMMIT_BLOCK block height and hash
	CalcDone      uint64    // Latest CALCULATE_DONE block height
	PrevCalcDone  uint64    // Previous CALCULATE_DONE block height
	Calculating   uint64    // Latest CALCULATE block height
	ToggleBH      uint64	// Latest account DB toggle block height
}

type DBInfoData DBInfoDataV2

type DBInfo struct {
	DBRoot        string
	DBType        string
	DBInfoData
}

func (dbi *DBInfo) ID() []byte {
	return []byte("")
}

func (dbi *DBInfo) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&dbi.DBInfoData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (dbi *DBInfo) String() string {
	b, err := json.Marshal(dbi)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (dbi *DBInfo) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &dbi.DBInfoData)
	if err != nil {
		// handle backward compatibility
		return dbi.backwardCompatibility(bs)
	}
	return nil
}

func (dbi *DBInfo) backwardCompatibility(bs []byte) error {
	var v1 DBInfoDataV1
	_, err := codec.UnmarshalFromBytes(bs, &v1)
	if err != nil {
		return err
	}

	dbi.DBCount = v1.DBCount
	dbi.QueryDBIsZero = v1.QueryDBIsZero
	dbi.CalcDone = v1.BlockHeight
	dbi.PrevCalcDone = v1.BlockHeight
	dbi.Calculating = v1.BlockHeight

	return nil
}

func NewDBInfo(mngDB db.Database, dbPath string, dbType string, dbName string, dbCount int) (*DBInfo, error) {
	writeToDB := false
	bucket, err := mngDB.GetBucket(db.PrefixManagement)
	if err != nil {
		log.Panicf("Failed to get DB Information bucket\n")
		return nil, err
	}
	dbInfo := new(DBInfo)
	data, err := bucket.Get(dbInfo.ID())
	if data != nil {
		err = dbInfo.SetBytes(data)
		if err != nil {
			log.Panicf("Failed to set DB Information structure\n")
			return nil, err
		}
	} else {
		// set DB count
		dbInfo.DBCount = dbCount

		writeToDB = true
	}

	dbInfo.DBRoot = filepath.Join(dbPath, dbName)
	dbInfo.DBType = dbType

	// Write to management DB
	if writeToDB {
		value, _ := dbInfo.Bytes()
		bucket.Set(dbInfo.ID(), value)
	}

	return dbInfo, nil
}

var BigIntTwo = big.NewInt(2)
var BigInt100 = big.NewInt(100)
var BigIntIScoreMultiplier = big.NewInt(iScoreMultiplier)

type GVData struct {
	CalculatedIncentiveRep common.HexInt
	RewardRep              common.HexInt
	MainPRepCount          common.HexInt
	SubPRepCount           common.HexInt
}

type GovernanceVariable struct {
	BlockHeight uint64
	GVData
	BlockProduceReward common.HexInt
	PRepReward         common.HexInt
}

func (gv *GovernanceVariable) ID() []byte {
	bs := make([]byte, 8)
	id := common.Uint64ToBytes(gv.BlockHeight)
	copy(bs[len(bs)-len(id):], id)
	return bs
}

func (gv *GovernanceVariable) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&gv.GVData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (gv *GovernanceVariable) String() string {
	b, err := json.Marshal(gv)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}


func (gv *GovernanceVariable) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &gv.GVData)
	if err != nil {
		return err
	}
	gv.setReward()
	return nil
}

func (gv *GovernanceVariable) setReward() {
	// block produce reward
	gv.BlockProduceReward.Mul(&gv.CalculatedIncentiveRep.Int, &gv.MainPRepCount.Int)
	gv.BlockProduceReward.Mul(&gv.BlockProduceReward.Int, BigIntIScoreMultiplier)
	gv.BlockProduceReward.Div(&gv.BlockProduceReward.Int, BigIntTwo)

	// Main/Sub P-Rep reward
	gv.PRepReward.Mul(&gv.CalculatedIncentiveRep.Int, BigInt100)
	gv.PRepReward.Mul(&gv.PRepReward.Int, BigIntIScoreMultiplier)
}

func LoadGovernanceVariable(dbi db.Database) ([]*GovernanceVariable, error) {
	gvList := make([]*GovernanceVariable, 0)

	iter, err := dbi.GetIterator()
	if err != nil {
		return gvList, err
	}

	prefix := util.BytesPrefix([]byte(db.PrefixGovernanceVariable))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		gv := new(GovernanceVariable)
		// read
		gvBlockHeight := common.BytesToUint64(iter.Key()[len(db.PrefixGovernanceVariable):])

		gv.SetBytes(iter.Value())
		gv.BlockHeight = gvBlockHeight
		gvList = append(gvList, gv)
	}
	sort.Slice(gvList, func(i, j int) bool {
		return gvList[i].BlockHeight < gvList[j].BlockHeight
	})

	// finalize iterator
	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Printf("There is error while load IISS GV iteration. %+v", err)
		return gvList, err
	}

	return gvList, nil
}

func NewGVFromIISS(iiss *IISSGovernanceVariable) *GovernanceVariable {
	gv := new(GovernanceVariable)
	gv.BlockHeight = iiss.BlockHeight
	gv.MainPRepCount.SetUint64(iiss.MainPRepCount)
	gv.SubPRepCount.SetUint64(iiss.SubPRepCount)
	gv.CalculatedIncentiveRep.SetUint64(iiss.IncentiveRep)
	gv.RewardRep.SetUint64(iiss.RewardRep)
	gv.setReward()

	return gv
}

type PRepDelegationInfo struct {
	Address         common.Address
	DelegatedAmount common.HexInt
}

func (di *PRepDelegationInfo) String() string {
	b, err := json.Marshal(di)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

type PRepData struct {
	TotalDelegation common.HexInt
	List []PRepDelegationInfo
}

func (pd *PRepData) String() string {
	b, err := json.Marshal(pd)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

type PRep struct {
	BlockHeight uint64
	PRepData
}

func (bp *PRep) ID() []byte {
	bs := make([]byte, 8)
	id := common.Uint64ToBytes(bp.BlockHeight)
	copy(bs[len(bs)-len(id):], id)
	return bs
}

func (bp *PRep) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&bp.PRepData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (bp *PRep) String() string {
	b, err := json.Marshal(bp)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (bp *PRep) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &bp.PRepData)
	if err != nil {
		return err
	}
	return nil
}

func LoadPRep(dbi db.Database) ([]*PRep, error) {
	pRepList := make([]*PRep, 0)

	iter, err := dbi.GetIterator()
	if err != nil {
		return nil, err
	}
	prefix := util.BytesPrefix([]byte(db.PrefixPRep))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		pRep := new(PRep)
		// read
		blockHeight := common.BytesToUint64(iter.Key()[len(db.PrefixPRep):])
		pRep.SetBytes(iter.Value())
		pRep.BlockHeight = blockHeight
		pRepList = append(pRepList, pRep)
	}

	// finalize iterator
	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Printf("There is error while load P-Rep iteration. %+v", err)
		return nil, err
	}

	return pRepList, nil
}

type PRepCandidateData struct {
	Start uint64
	End   uint64	// 0 means that did not unregister
}

type PRepCandidate struct {
	Address common.Address
	PRepCandidateData
}

func (prep *PRepCandidate) ID() []byte {
	return prep.Address.Bytes()
}

func (prep *PRepCandidate) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&prep.PRepCandidateData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (prep *PRepCandidate) String() string {
	b, err := json.Marshal(prep)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (prep *PRepCandidate) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &prep.PRepCandidateData)
	if err != nil {
		return err
	}
	return nil
}

func LoadPRepCandidate(dbi db.Database) (map[common.Address]*PRepCandidate, error) {
	pRepMap := make(map[common.Address]*PRepCandidate)

	iter, err := dbi.GetIterator()
	if err != nil {
		return nil, err
	}
	prefix := util.BytesPrefix([]byte(db.PrefixPRepCandidate))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		pRep := new(PRepCandidate)
		// read
		addr := common.NewAddress(iter.Key()[len(db.PrefixPRepCandidate):])
		pRep.SetBytes(iter.Value())
		pRep.Address = *addr
		pRepMap[*addr] = pRep
	}

	// finalize iterator
	iter.Release()
	err = iter.Error()
	if err != nil {
		log.Printf("There is error while load P-Rep candidate iteration. %+v", err)
		return nil, err
	}

	return pRepMap, nil
}
