package core

import (
	"encoding/json"
	"github.com/icon-project/rewardcalculator/common/db"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
)
type CRData struct {
	Success bool
	StateHash []byte
	IScore common.HexInt
	Beta1 common.HexInt
	Beta2 common.HexInt
	Beta3 common.HexInt
}

type CalculationResult struct {
	BlockHeight uint64
    CRData
}

func (cr *CalculationResult) ID() []byte {
	return common.Uint64ToBytes(cr.BlockHeight)
}

func (cr *CalculationResult) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&cr.CRData); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (cr *CalculationResult) String() string {
	b, err := json.Marshal(cr)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (cr *CalculationResult) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &cr.CRData)
	if err != nil {
		return err
	}
	return nil
}

func NewCalculationResultFromBytes(bs []byte) (*CalculationResult, error) {
	cr := new(CalculationResult)
	if err:= cr.SetBytes(bs); err != nil {
		return nil, err
	} else {
		return cr, nil
	}
}

func WriteCalculationResult(crDB db.Database, blockHeight uint64, stats *Statistics, stateHash []byte) {
	cr := new(CalculationResult)

	cr.Success = true
	cr.BlockHeight = blockHeight
	cr.StateHash = stateHash
	cr.IScore.Set(&stats.TotalReward.Int)
	cr.Beta1.Set(&stats.Beta1.Int)
	cr.Beta2.Set(&stats.Beta2.Int)
	cr.Beta3.Set(&stats.Beta3.Int)

	bucket, _ := crDB.GetBucket(db.PrefixCalcResult)
	bs, _ := cr.Bytes()
	bucket.Set(cr.ID(), bs)
}
