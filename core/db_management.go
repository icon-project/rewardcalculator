package core

import (
	"encoding/json"
	"log"
	"math/big"
	"path/filepath"

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

type DBInfoData struct {
	DBCount       int
	BlockHeight   uint64 // BlockHeight of finished calculate message
	QueryDBIsZero bool
}

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
		return err
	}
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
var BigIntIScoreMultiplier = big.NewInt(iScoreMultiplier)

type GVData struct {
	MainPRepCount          common.HexInt
	SubPRepCount           common.HexInt
	CalculatedIncentiveRep common.HexInt
	RewardRep              common.HexInt
}

type GovernanceVariable struct {
	BlockHeight uint64
	GVData
	BlockProduceReward common.HexInt
	PRepReward         common.HexInt
}

func (gv *GovernanceVariable) ID() []byte {
	return common.Uint64ToBytes(gv.BlockHeight)
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
	var numPRep common.HexInt
	numPRep.Add(&gv.MainPRepCount.Int, &gv.SubPRepCount.Int)
	gv.PRepReward.Mul(&gv.CalculatedIncentiveRep.Int, &numPRep.Int)
	gv.PRepReward.Mul(&gv.PRepReward.Int, BigIntIScoreMultiplier)
}

func LoadGovernanceVariable(dbi db.Database, workingBH uint64) ([]*GovernanceVariable, error) {
	gvList := make([]*GovernanceVariable, 0)

	iter, err := dbi.GetIterator()
	if err != nil {
		return gvList, err
	}

	oldGV := 0
	prefix := util.BytesPrefix([]byte(db.PrefixGovernanceVariable))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		gv := new(GovernanceVariable)
		// read
		gvBlockHeight := common.BytesToUint64(iter.Key()[len(db.PrefixGovernanceVariable):])

		gv.SetBytes(iter.Value())
		gv.BlockHeight = gvBlockHeight
		gvList = append(gvList, gv)
		if workingBH > gvBlockHeight {
			oldGV++
		}
	}

	// finalize iterator
	iter.Release()
	err = iter.Error()
	if err != nil {
		return gvList, err
	}

	// delete old GVs except last one
	if oldGV > 0 {
		// delete from management DB
		bucket, _ := dbi.GetBucket(db.PrefixGovernanceVariable)
		for i := 0; i < oldGV-1; i++ {
			bucket.Delete(gvList[i].ID())
		}
		// delete from memory
		gvList = gvList[oldGV-1:]
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
		return nil, err
	}

	return pRepMap, nil
}
