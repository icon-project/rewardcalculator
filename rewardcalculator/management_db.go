package rewardcalculator

import (
	"encoding/json"
	"log"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const MaxDBCount  = 256

type DBInfo struct {
	DBRoot        string
	DBType        string
	DBCount       int
	BlockHeight   uint64 // BlockHeight of finished calculate message
	QueryDBIsZero bool
}

func (dbi *DBInfo) ID() []byte {
	return []byte("")
}

func (dbi *DBInfo) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(dbi); err != nil {
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
	_, err := codec.UnmarshalFromBytes(bs, dbi)
	if err != nil {
		return err
	}
	return nil
}

func NewDBInfo(globalDB db.Database, dbPath string, dbType string, dbName string, dbCount int) (*DBInfo, error) {
	bucket, err := globalDB.GetBucket(db.PrefixManagement)
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
		// write Initial values. DB path, type and count
		dbInfo.DBRoot = dbPath + "/" + dbName
		dbInfo.DBType = dbType
		dbInfo.DBCount = dbCount
		value, _ := dbInfo.Bytes()
		bucket.Set(dbInfo.ID(), value)
	}
	return dbInfo, nil
}

type GVData struct {
	IcxPrice      common.HexInt
	IncentiveRep  common.HexInt
}

type GovernanceVariable struct {
	BlockHeight uint64
	GVData
	RewardRep	common.HexInt
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
	gv.setRewardRep()
	return nil
}

func (gv *GovernanceVariable) setRewardRep() {
	// TODO modify RewardRep
	gv.RewardRep = gv.IncentiveRep
}

func LoadGovernanceVariable(dbi db.Database, workingBH uint64) ([]*GovernanceVariable, error) {
	gvCount := 0
	gvList := make([]*GovernanceVariable, 1)

	iter, err := dbi.GetIterator()
	if err != nil {
		return nil, err
	}
	prefix := util.BytesPrefix([]byte(db.PrefixGovernanceVariable))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		gvCount++
		gv := new(GovernanceVariable)
		// read
		gvBlockHeight := common.BytesToUint64(iter.Key()[len(db.PrefixGovernanceVariable):])

		gv.SetBytes(iter.Value())
		gv.BlockHeight = gvBlockHeight
		if workingBH < gvBlockHeight {
			gvList = append(gvList, gv)
		} else {
			// overwrite
			gvList[0] = gv
		}
	}

	// finalize iterator
	iter.Release()
	err = iter.Error()
	if err != nil {
		return nil, err
	}

	if gvCount == 0 {
		return nil, nil
	} else {
		return gvList, nil
	}
}

func NewGVFromIISS(iiss *IISSGovernanceVariable) *GovernanceVariable {
	gv := new(GovernanceVariable)
	gv.BlockHeight = iiss.BlockHeight
	gv.IcxPrice.SetUint64(iiss.IcxPrice)
	gv.IncentiveRep.SetUint64(iiss.IncentiveRep)
	gv.setRewardRep()

	return gv
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
	prefix := util.BytesPrefix([]byte(db.PrefixPrepCandidate))
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		pRep := new(PRepCandidate)
		// read
		addr := common.NewAddress(iter.Key()[len(db.PrefixPrepCandidate):])
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
