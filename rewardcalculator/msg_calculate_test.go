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
	gv.RewardRep.SetUint64(minRewardRep)
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
	// hx11 delegates MinDelegation to prepA and delegates 2 * MinDelegation to prepB
	tx := new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = 10
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation := make([]interface{}, 0)

	dgData := make([]interface{}, 0)
	dgData = append(dgData, &prepA.Address)
	dgData = append(dgData, MinDelegation)
	delegation = append(delegation, dgData)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &prepB.Address)
	dgData = append(dgData, 2 * MinDelegation)
	delegation = append(delegation, dgData)

	var err error
	tx.Data, err = common.EncodeAny(delegation)
	if err != nil {
		fmt.Printf("Can't encode delegation. err=%+v\n", err)
		return
	}
	tests = append(tests, tx)

	// TX 1: Modify delegation at block height 20
	// hx11 delegates MinDelegation to prepA and delegates MinDelegation to iconist
	tx = new(IISSTX)
	tx.Index = 1
	tx.BlockHeight = 20
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation = make([]interface{}, 0)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &prepA.Address)
	dgData = append(dgData, MinDelegation)
	delegation = append(delegation, dgData)

	dgData = make([]interface{}, 0)
	dgData = append(dgData, &iconist)
	dgData = append(dgData, MinDelegation)
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

	reward := 3 * MinDelegation * (20 - 10) * minRewardRep / rewardDivider +
		MinDelegation * (30 - 20) * minRewardRep / rewardDivider

	//log.Printf("%d , %d", reward, ia.IScore.Uint64())
	assert.Equal(t, uint64(reward), ia.IScore.Uint64())
}

func TestMsgCalc_CalculateIISSTX_small_delegation(t *testing.T) {
	ctx := initTest()
	defer finalizeTest()

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(minRewardRep)
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
	// hx11 delegates MinDelegation to prepA and delegates 2 * MinDelegation to prepB
	tx := new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = 10
	tx.Address = iconist
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation := make([]interface{}, 0)

	// delegate small value
	dgData := make([]interface{}, 0)
	dgData = append(dgData, &prepA.Address)
	dgData = append(dgData, MinDelegation - 1)
	delegation = append(delegation, dgData)

	var err error
	tx.Data, err = common.EncodeAny(delegation)
	if err != nil {
		fmt.Printf("Can't encode delegation. err=%+v\n", err)
		return
	}
	tests = append(tests, tx)

	// calculate IISS TX
	calculateIISSTX(ctx, tests, 100)

	// check Calculate DB
	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ := calcDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(iconist.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)

	//log.Printf("%d , %d", reward, ia.IScore.Uint64())
	assert.Equal(t, uint64(0), ia.IScore.Uint64())
}

func TestMsgCalc_CalculateIISSBlockProduce(t *testing.T) {
	const (
		bp0BlockHeight = 5
		bp1BlockHeight = 11
		bp2BlockHeight = 12
		gv0BlockHeight = 0
		gv1BlockHeight = 10
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
		BlockHeight0 uint64 = 0
		BlockHeight1 uint64 = 10
		BlockHeight2 uint64 = 20

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
	assert.Equal(t, BlockHeight2, ia.BlockHeight)

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
	assert.Equal(t, BlockHeight2, ia.BlockHeight)
}

func TestMsgCalc_CalculateDB(t *testing.T) {
	const (
		rewardRep = 1

		calculateBlockHeight uint64 = 100

		addr1BlockHeight uint64 = 1
		addr1InitIScore = 100
		addr1DelegationToPRepA = 10

		addr2BlockHeight uint64 = 10
		addr2InitIScore = 0
		addr2DelegationToPRepA = 20
		addr2DelegationToPRepB = 30
	)
	ctx := initTest()
	defer finalizeTest()

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(rewardRep)
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

	addr1 := *common.NewAddressFromString("hx11")
	addr2 := *common.NewAddressFromString("hx22")

	// set Query DB for read
	queryDB := ctx.DB.getQueryDB(addr1)
	calcDB := ctx.DB.getCalculateDB(addr1)
	bucket, _ := queryDB.GetBucket(db.PrefixIScore)

	/// addr1
	ia := new(IScoreAccount)
	ia.Address = addr1
	ia.BlockHeight = addr1BlockHeight
	ia.IScore.SetUint64(addr1InitIScore)
	ia.Delegations = make([]*DelegateData, 0)
	delegation := new(DelegateData)
	delegation.Address = prepA.Address
	delegation.Delegate.SetUint64(addr1DelegationToPRepA)
	ia.Delegations = append(ia.Delegations, delegation)
	delegation = new(DelegateData)
	delegation.Address = addr2
	delegation.Delegate.SetUint64(1000)
	ia.Delegations = append(ia.Delegations, delegation)
	bucket.Set(ia.ID(), ia.Bytes())

	/// addr2
	ia = new(IScoreAccount)
	ia.Address = addr2
	ia.BlockHeight = addr2BlockHeight
	ia.IScore.SetUint64(addr2InitIScore)
	ia.Delegations = make([]*DelegateData, 0)
	delegation = new(DelegateData)
	delegation.Address = prepA.Address
	delegation.Delegate.SetUint64(addr2DelegationToPRepA)
	ia.Delegations = append(ia.Delegations, delegation)
	delegation = new(DelegateData)
	delegation.Address = prepB.Address
	delegation.Delegate.SetUint64(addr2DelegationToPRepB)
	ia.Delegations = append(ia.Delegations, delegation)
	bucket.Set(ia.ID(), ia.Bytes())

	// calculate
	calculateDB(queryDB, calcDB, ctx.GV, ctx.PRepCandidates, calculateBlockHeight, writeBatchCount)

	var reward uint64

	// check - addr1
	period := calculateBlockHeight - addr1BlockHeight
	gv = ctx.getGVByBlockHeight(addr1BlockHeight)
	if gv == nil {
		assert.True(t, false)
		return
	}
	// calculate delegation reward for P-Rep only
	reward = gv.RewardRep.Uint64() * period * addr1DelegationToPRepA / rewardDivider + addr1InitIScore

	bucket, _ = calcDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(addr1.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	//log.Printf("%d : %d", reward, ia.IScore.Uint64())
	assert.Equal(t, reward, ia.IScore.Uint64())
	assert.Equal(t, calculateBlockHeight, ia.BlockHeight)

	// check - addr2
	period = calculateBlockHeight - addr2BlockHeight
	gv = ctx.getGVByBlockHeight(addr2BlockHeight)
	if gv == nil {
		assert.True(t, false)
		return
	}
	reward = gv.RewardRep.Uint64() * period * (addr2DelegationToPRepA + addr2DelegationToPRepB) / rewardDivider

	bs, _ = bucket.Get(addr2.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	//log.Printf("%d : %d", reward, ia.IScore.Uint64())
	assert.Equal(t, reward, ia.IScore.Uint64())
	assert.Equal(t, calculateBlockHeight, ia.BlockHeight)
}
