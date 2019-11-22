package core

import (
	"encoding/binary"
	"github.com/icon-project/rewardcalculator/common/db"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
)

const (
	calcIScore      uint64 = 10
	calcBlockHeight uint64 = 1000
)

func makeCalcResult() *CalculationResult {
	calculationResult := new(CalculationResult)

	calculationResult.Success = true
	calculationResult.BlockHeight = calcBlockHeight
	calculationResult.IScore.SetUint64(calcIScore)
	calculationResult.StateHash = make([]byte, 64)
	binary.BigEndian.PutUint64(calculationResult.StateHash, calculationResult.BlockHeight)

	return calculationResult
}

func TestDBCalculate_ID(t *testing.T) {
	calculationResult := makeCalcResult()

	assert.Equal(t, common.Uint64ToBytes(calculationResult.BlockHeight), calculationResult.ID())
}

func TestDBCalculate_BytesAndSetBytes(t *testing.T) {
	calculationResult := makeCalcResult()

	var calcResultNew CalculationResult

	bs, err := calculationResult.Bytes()
	assert.NoError(t, err)
	err = calcResultNew.SetBytes(bs)
	assert.NoError(t, err)
	bsNew, err := calcResultNew.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, calculationResult.Success, calcResultNew.Success)
	assert.Equal(t, 0, calculationResult.IScore.Cmp(&calcResultNew.IScore.Int))
	assert.Equal(t, calculationResult.StateHash, calcResultNew.StateHash)
	assert.Equal(t, bs, bsNew)
}

func TestDBCalculate_NewClaimFromBytes(t *testing.T) {
	calculationResult := makeCalcResult()

	bs, err := calculationResult.Bytes()
	assert.NoError(t, err)
	calcResultNew, err := NewCalculationResultFromBytes(bs)
	assert.NoError(t, err)
	bsNew, err := calcResultNew.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, calculationResult.Success, calcResultNew.Success)
	assert.Equal(t, 0, calculationResult.IScore.Cmp(&calcResultNew.IScore.Int))
	assert.Equal(t, calculationResult.StateHash, calcResultNew.StateHash)
	assert.Equal(t, bs, bsNew)
}

func TestDBCalculate_CalculationResult(t *testing.T) {
	var calculationResult CalculationResult

	ctx := initTest(1)
	defer finalizeTest(ctx)
	crDB := ctx.DB.getCalculateResultDB()

	stats := new(Statistics)
	stats.TotalReward.SetUint64(calcIScore)
	stateHash := make([]byte, 64)
	binary.BigEndian.PutUint64(stateHash, calcBlockHeight)

	WriteCalculationResult(crDB, calcBlockHeight, stats, stateHash)

	bucket, err := crDB.GetBucket(db.PrefixCalcResult)
	assert.NoError(t, err)
	bs, err := bucket.Get(common.Uint64ToBytes(calcBlockHeight))
	assert.NoError(t, err)

	err = calculationResult.SetBytes(bs)
	assert.NoError(t, err)

	assert.True(t, calculationResult.Success)
	assert.Equal(t, 0, calculationResult.IScore.Cmp(&stats.TotalReward.Int))
	assert.Equal(t, stateHash, calculationResult.StateHash)

	DeleteCalculationResult(crDB, calcBlockHeight)

	bs, _ = bucket.Get(common.Uint64ToBytes(calcBlockHeight))
	assert.Nil(t, bs)
}
