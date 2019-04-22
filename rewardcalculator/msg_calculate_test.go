package rewardcalculator

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestMsgCalc_CalculateIISSTX(t *testing.T) {
	ctx := initTest()
	defer finalizeTest()

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	// set P-Rep candidate
	prepA := new(PRepCandidate)
	prepA.Address = *common.NewAddressFromString("hxaa")
	prepA.Start = 0
	ctx.PRepCandidates[prepA.Address] = prepA
	prepB := new(PRepCandidate)
	prepB.Address = *common.NewAddressFromString("hxbb")
	prepB.Start = 0
	ctx.PRepCandidates[prepB.Address] = prepB

	// set IISS TX
	tests := make([]*IISSTX, 0)
	iconist := *common.NewAddressFromString("hx11")

	// TX 0: Add new delegation at block height 10
	// hx11 delegates 10 to prepA and delegates 20 to prepB
	// get 10 + 20 I-Score per block
	tx := new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = 10
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation := make([]interface{}, 0)

	dgData := make([]interface{}, 0)
	dgData = append(dgData, &prepA.Address)
	dgData = append(dgData, 10)
	delegation = append(delegation, dgData)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &prepB.Address)
	dgData = append(dgData, 20)
	delegation = append(delegation, dgData)

	var err error
	tx.Data, err = common.EncodeAny(delegation)
	if err != nil {
		fmt.Printf("Can't encode delegation. err=%+v\n", err)
		return
	}
	tests = append(tests, tx)

	// TX 1: Modify delegation at block height 20
	// hx11 delegates 100 to prepA and delegates 200 to iconist
	// get 100 I-Score per block
	tx = new(IISSTX)
	tx.Index = 1
	tx.BlockHeight = 20
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation = make([]interface{}, 0)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &prepA.Address)
	dgData = append(dgData, 100)
	delegation = append(delegation, dgData)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &iconist)
	dgData = append(dgData, 200)
	delegation = append(delegation, dgData)

	tx.Data, err = common.EncodeAny(delegation)
	if err != nil {
		fmt.Printf("Can't encode delegation. err=%+v\n", err)
		return
	}
	tests = append(tests, tx)

	// TX 2: Delete delegation at block height 30
	tx = new(IISSTX)

	tx.Index = 2
	tx.BlockHeight = 30
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)
	tx.Data.Type = codec.TypeNil
	tx.Data.Object = []byte("")

	tests = append(tests, tx)

	// calculate IISS TX
	calculateIISSTX(ctx, tests, 100)

	// check Calculate DB
	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ := calcDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(iconist.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)

	reward := common.NewHexInt(30 * (20 - 10) + 100 * (30 - 20))
	log.Printf("%s , %s", reward.String(), ia.String())
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))
}

func TestMsgCalc_CalculateIISSBlockProduce(t *testing.T) {
	const (
		bp0BlockHeight = 5
		bp1BlockHeight = 11
		bp2BlockHeight uint64 = 12
		gv0BlockHeight uint64 = 0
		gv1BlockHeight uint64 = 10
	)

	ctx := initTest()
	defer finalizeTest()

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = gv0BlockHeight
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	gv = new(GovernanceVariable)
	gv.BlockHeight = gv1BlockHeight
	gv.CalculatedIncentiveRep.SetUint64(10)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	// set P-Rep
	prepA := *common.NewAddressFromString("hxaa")
	prepB := *common.NewAddressFromString("hxbb")
	prepC := *common.NewAddressFromString("hxcc")

	// set IISS Block produce Info.
	tests := make([]*IISSBlockProduceInfo, 0)
	iconist := *common.NewAddressFromString("hx11")

	// BP 0:
	// Generator : prepA, Validator : prepB, prepC
	bp := new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp0BlockHeight
	bp.Generator = prepA
	bp.Validator = append(bp.Validator, prepB)
	bp.Validator = append(bp.Validator, prepC)
	tests = append(tests, bp)

	// BP 1:
	// Generator : prepA, Validator : prepC
	bp = new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp1BlockHeight
	bp.Generator = prepA
	bp.Validator = append(bp.Validator, prepC)
	tests = append(tests, bp)

	// BP 2:
	// Generator : prepB, Validator : prepA
	bp = new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp2BlockHeight
	bp.Generator = prepB
	bp.Validator = append(bp.Validator, prepA)
	tests = append(tests, bp)

	// calculate BP
	calculateIISSBlockProduce(ctx, tests, 100)

	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ := calcDB.GetBucket(db.PrefixIScore)

	var reward, reward0, reward1, reward2 common.HexInt

	// check prepA
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Set(&gv.BlockProduceReward.Int)
	gv = ctx.getGVByBlockHeight(bp1BlockHeight)
	reward1.Set(&gv.BlockProduceReward.Int)
	gv = ctx.getGVByBlockHeight(bp2BlockHeight)
	reward2.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward1.Int)
	reward.Add(&reward.Int, &reward2.Int)

	bs, _ := bucket.Get(prepA.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	// check prepB
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Div(&gv.BlockProduceReward.Int, &common.NewHexIntFromUint64(2).Int)
	gv = ctx.getGVByBlockHeight(bp2BlockHeight)
	reward2.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward2.Int)

	bs, _ = bucket.Get(prepB.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	// check prepC
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Div(&gv.BlockProduceReward.Int, &common.NewHexIntFromUint64(2).Int)
	gv = ctx.getGVByBlockHeight(bp1BlockHeight)
	reward1.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward1.Int)

	bs, _ = bucket.Get(prepC.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))
}

func TestMsgCalc_CalculatePRepReward(t *testing.T) {
	const (
		BlockHeight0 = 0
		BlockHeight1 = 10
		BlockHeight2 = 20

		TotalDelegation0 = 10
		DelegationA0     = 4
		DelegationB0     = 6
		TotalDelegation1 = 20
		DelegationA1     = 14
		DelegationB1     = 6
	)

	ctx := initTest()
	defer finalizeTest()

	prepA := *common.NewAddressFromString("hxaa")
	prepB := *common.NewAddressFromString("hxbb")

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = BlockHeight0
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	gv = new(GovernanceVariable)
	gv.BlockHeight = BlockHeight1
	gv.CalculatedIncentiveRep.SetUint64(2)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	// P-Rep 0
	prep := new(PRep)
	prep.BlockHeight = BlockHeight0
	prep.TotalDelegation.SetUint64(TotalDelegation0)
	prep.List = make([]PRepDelegationInfo, 0)

	dInfo := new(PRepDelegationInfo)
	dInfo.Address = prepA
	dInfo.DelegatedAmount.SetUint64(DelegationA0)
	prep.List = append(prep.List, *dInfo)

	dInfo = new(PRepDelegationInfo)
	dInfo.Address = prepB
	dInfo.DelegatedAmount.SetUint64(DelegationB0)
	prep.List = append(prep.List, *dInfo)
	ctx.PRep = append(ctx.PRep, prep)

	// P-Rep 1
	prep = new(PRep)
	prep.BlockHeight = BlockHeight1
	prep.TotalDelegation.SetUint64(TotalDelegation1)
	prep.List = make([]PRepDelegationInfo, 0)

	dInfo = new(PRepDelegationInfo)
	dInfo.Address = prepA
	dInfo.DelegatedAmount.SetUint64(DelegationA1)
	prep.List = append(prep.List, *dInfo)

	dInfo = new(PRepDelegationInfo)
	dInfo.Address = prepB
	dInfo.DelegatedAmount.SetUint64(DelegationB1)
	prep.List = append(prep.List, *dInfo)
	ctx.PRep = append(ctx.PRep, prep)

	// calculate P-Rep reward
	calculatePRepReward(ctx, BlockHeight2)

	calcDB := ctx.DB.getCalculateDB(prepA)
	bucket, _ := calcDB.GetBucket(db.PrefixIScore)

	var reward, reward0, reward1 common.HexInt

	// check prepA
	period := common.NewHexIntFromUint64(BlockHeight1 - BlockHeight0)
	gv = ctx.getGVByBlockHeight(BlockHeight1)
	reward0.Mul(&gv.PRepReward.Int, &period.Int)
	reward0.Mul(&reward0.Int, &common.NewHexIntFromUint64(DelegationA0).Int)
	reward0.Div(&reward0.Int, &common.NewHexIntFromUint64(TotalDelegation0).Int)

	period = common.NewHexIntFromUint64(BlockHeight2 - BlockHeight1)
	gv = ctx.getGVByBlockHeight(BlockHeight2)
	reward1.Mul(&gv.PRepReward.Int, &period.Int)
	reward1.Mul(&reward1.Int, &common.NewHexIntFromUint64(DelegationA1).Int)
	reward1.Div(&reward1.Int, &common.NewHexIntFromUint64(TotalDelegation1).Int)

	reward.Add(&reward0.Int, &reward1.Int)

	bs, _ := bucket.Get(prepA.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)
	log.Printf("%s + %s = %s : %s", reward0.String(), reward1.String(), reward.String(), ia.String())
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	// check prepB
	period = common.NewHexIntFromUint64(BlockHeight1 - BlockHeight0)
	gv = ctx.getGVByBlockHeight(BlockHeight1)
	reward0.Mul(&gv.PRepReward.Int, &period.Int)
	reward0.Mul(&reward0.Int, &common.NewHexIntFromUint64(DelegationB0).Int)
	reward0.Div(&reward0.Int, &common.NewHexIntFromUint64(TotalDelegation0).Int)

	period = common.NewHexIntFromUint64(BlockHeight2 - BlockHeight1)
	gv = ctx.getGVByBlockHeight(BlockHeight2)
	reward1.Mul(&gv.PRepReward.Int, &period.Int)
	reward1.Mul(&reward1.Int, &common.NewHexIntFromUint64(DelegationB1).Int)
	reward1.Div(&reward1.Int, &common.NewHexIntFromUint64(TotalDelegation1).Int)

	reward.Add(&reward0.Int, &reward1.Int)

	bs, _ = bucket.Get(prepB.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	log.Printf("%s + %s = %s : %s", reward0.String(), reward1.String(), reward.String(), ia.String())
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))
}
