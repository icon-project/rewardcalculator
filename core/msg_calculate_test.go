package core

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/sha3"
)

func TestMsgCalc_CalculateIISSTX(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
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

	// write IISS TX
	iissDBDir := testDBDir + "/iiss"
	iissDB := db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	defer iissDB.Close()
	defer os.RemoveAll(iissDBDir)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSTX)
	txList := make([]*IISSTX, 0)
	iconist := *common.NewAddressFromString("hx11")

	// TX 0: Add new delegation at block height 10
	// iconist delegates MinDelegation to prepA and delegates 2 * MinDelegation to prepB
	dgDataSlice := []DelegateData {
		{prepA.Address, *common.NewHexIntFromUint64(MinDelegation)},
		{prepB.Address, *common.NewHexIntFromUint64(MinDelegation * 2)},
	}
	tx := makeIISSTX(TXDataTypeDelegate, iconist.String(), dgDataSlice)
	tx.Index = 0
	tx.BlockHeight = 10
	txList = append(txList, tx)

	// TX 1: Modify delegation at block height 20
	// iconist delegates MinDelegation to prepA and delegates MinDelegation to iconist
	dgDataSlice = []DelegateData {
		{prepA.Address, *common.NewHexIntFromUint64(MinDelegation)},
		{iconist, *common.NewHexIntFromUint64(MinDelegation)},
	}
	tx = makeIISSTX(TXDataTypeDelegate, iconist.String(), dgDataSlice)
	tx.Index = 1
	tx.BlockHeight = 20
	txList = append(txList, tx)

	// TX 2: iconist Delete delegation at block height 30
	tx = makeIISSTX(TXDataTypeDelegate, iconist.String(), nil)
	tx.Index = 2
	tx.BlockHeight = 30
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// calculate IISS TX
	account, stats, hash := calculateIISSTX(ctx, iissDB, 100, false)
	assert.Equal(t, uint64(1), account)

	// check Calculate DB
	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ = calcDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(iconist.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)
	ia.Address = iconist
	iaHash, _ := NewIScoreAccountFromBytes(bs)
	iaHash.Address = iconist
	iaHash.BlockHeight = 100

	stateHash := make([]byte, 64)
	h := sha3.NewShake256()
	iaHash.IScore.SetUint64(3 * MinDelegation * (100 - 10) * minRewardRep / rewardDivider)
	h.Write(iaHash.BytesForHash())
	iaHash.IScore.SetUint64(3 * MinDelegation * (20 - 10) * minRewardRep / rewardDivider +
		MinDelegation * (100 - 20) * minRewardRep / rewardDivider)
	h.Write(iaHash.BytesForHash())
	iaHash.IScore.SetUint64(3 * MinDelegation * (20 - 10) * minRewardRep / rewardDivider +
		MinDelegation * (30 - 20) * minRewardRep / rewardDivider)
	h.Write(iaHash.BytesForHash())
	h.Read(stateHash)

	reward := 3 * MinDelegation * (20 - 10) * minRewardRep / rewardDivider +
		MinDelegation * (30 - 20) * minRewardRep / rewardDivider

	assert.Equal(t, uint64(reward), ia.IScore.Uint64())
	assert.Equal(t, uint64(reward), stats.Uint64())
	assert.Equal(t, stateHash, hash)
}

func TestMsgCalc_CalculateIISSTX_small_delegation(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
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

	// write IISS TX
	iissDBDir := testDBDir + "/iiss"
	iissDB := db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	defer iissDB.Close()
	defer os.RemoveAll(iissDBDir)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSBPInfo)
	txList := make([]*IISSTX, 0)
	iconist := *common.NewAddressFromString("hx11")

	// TX 0: Add new delegation at block height 10
	// iconist delegates MinDelegation - 1 to prepA
	dgDataSlice := []DelegateData {
		{prepA.Address, *common.NewHexIntFromUint64(MinDelegation - 1)},
	}
	tx := makeIISSTX(TXDataTypeDelegate, iconist.String(), dgDataSlice)
	tx.Index = 0
	tx.BlockHeight = 10
	txList = append(txList, tx)

	// write to IISS data DB
	writeTX(iissDB, txList)

	// calculate IISS TX
	account, stats, hash := calculateIISSTX(ctx, iissDB, 100, false)
	assert.Equal(t, uint64(1), account)

	// check Calculate DB
	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ = calcDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(iconist.Bytes())
	ia, err := NewIScoreAccountFromBytes(bs)
	if err != nil {
		log.Printf("NewIScoreAccountFromBytes error : %+v", err)
	}
	ia.Address = iconist
	stateHash := make([]byte, 64)
	sha3.ShakeSum256(stateHash, ia.BytesForHash())

	assert.Equal(t, uint64(0), ia.IScore.Uint64())
	assert.Equal(t, uint64(0), stats.Uint64())
	assert.Equal(t, stateHash, hash)
}

func TestMsgCalc_CalculateIISSBlockProduce(t *testing.T) {
	const (
		bp0BlockHeight = 5
		bp1BlockHeight = 11
		bp2BlockHeight = 12
		gv0BlockHeight = 0
		gv1BlockHeight = 10
	)

	ctx := initTest(1)
	defer finalizeTest(ctx)

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = gv0BlockHeight
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	gv = new(GovernanceVariable)
	gv.BlockHeight = gv1BlockHeight
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
	gv.CalculatedIncentiveRep.SetUint64(10)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	// set P-Rep
	prepA := *common.NewAddressFromString("hxaa")
	prepB := *common.NewAddressFromString("hxbb")
	prepC := *common.NewAddressFromString("hxcc")

	// write IISS block produce Info.
	iconist := *common.NewAddressFromString("hx11")
	iissDBDir := testDBDir + "/iiss"
	iissDB := db.Open(iissDBDir, string(db.GoLevelDBBackend), testDB)
	defer iissDB.Close()
	defer os.RemoveAll(iissDBDir)
	bucket, _ := iissDB.GetBucket(db.PrefixIISSBPInfo)

	// BP 0:
	// Generator : prepA, Validator : prepB, prepC
	bp := new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp0BlockHeight
	bp.Generator = prepA
	bp.Validator = append(bp.Validator, prepB)
	bp.Validator = append(bp.Validator, prepC)
	bs, _ := bp.Bytes()
	bucket.Set(bp.ID(), bs)

	// BP 1:
	// Generator : prepA, Validator : prepC
	bp = new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp1BlockHeight
	bp.Generator = prepA
	bp.Validator = append(bp.Validator, prepC)
	bs, _ = bp.Bytes()
	bucket.Set(bp.ID(), bs)

	// BP 2:
	// Generator : prepB, Validator : prepA
	bp = new(IISSBlockProduceInfo)
	bp.Validator = make([]common.Address, 0)
	bp.BlockHeight = bp2BlockHeight
	bp.Generator = prepB
	bp.Validator = append(bp.Validator, prepA)
	bs, _ = bp.Bytes()
	bucket.Set(bp.ID(), bs)

	// calculate BP
	account, stats, hash := calculateIISSBlockProduce(ctx, iissDB, 100, false)
	assert.Equal(t, uint64(3), account)

	calcDB := ctx.DB.getCalculateDB(iconist)
	bucket, _ = calcDB.GetBucket(db.PrefixIScore)

	var reward, reward0, reward1, reward2, totalReward common.HexInt
	stateHash := make([]byte, 64)
	iaSlice := make([]*IScoreAccount, 0)

	// check prepA
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Set(&gv.BlockProduceReward.Int)
	gv = ctx.getGVByBlockHeight(bp1BlockHeight)
	reward1.Set(&gv.BlockProduceReward.Int)
	gv = ctx.getGVByBlockHeight(bp2BlockHeight)
	reward2.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward1.Int)
	reward.Add(&reward.Int, &reward2.Int)

	bs, _ = bucket.Get(prepA.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)
	ia.Address = prepA
	iaSlice = append(iaSlice, ia)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	totalReward.Add(&totalReward.Int, &reward.Int)

	// check prepB
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Div(&gv.BlockProduceReward.Int, &common.NewHexIntFromUint64(2).Int)
	gv = ctx.getGVByBlockHeight(bp2BlockHeight)
	reward2.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward2.Int)

	bs, _ = bucket.Get(prepB.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	ia.Address = prepB
	iaSlice = append(iaSlice, ia)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	totalReward.Add(&totalReward.Int, &reward.Int)

	// check prepC
	gv = ctx.getGVByBlockHeight(bp0BlockHeight)
	reward0.Div(&gv.BlockProduceReward.Int, &common.NewHexIntFromUint64(2).Int)
	gv = ctx.getGVByBlockHeight(bp1BlockHeight)
	reward1.Set(&gv.BlockProduceReward.Int)

	reward.Add(&reward0.Int, &reward1.Int)

	bs, _ = bucket.Get(prepC.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	ia.Address = prepC
	iaSlice = append(iaSlice, ia)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))

	totalReward.Add(&totalReward.Int, &reward.Int)

	// check stats
	assert.Equal(t, 0, totalReward.Cmp(&stats.Int))

	// check state hash
	// sort data and make state root hash
	sort.Slice(iaSlice, func(i, j int) bool {
		return iaSlice[i].Compare(iaSlice[j]) < 0
	})
	h := sha3.NewShake256()
	for _, ia := range iaSlice {
		h.Write(ia.BytesForHash())
	}
	h.Read(stateHash)
	assert.Equal(t, stateHash, hash)

}

func setRevision(ctx *Context, revision uint64) {
	ctx.Revision = revision
}

func TestMsgCalc_CalculatePRepReward(t *testing.T) {
	for revision := RevisionMin - 1; revision <= RevisionMax; revision++ {
		t.Run(fmt.Sprintf("Revision:%d", revision), func(t *testing.T) {
			testCalculatePRepReward(t, revision)
		})
	}
}

func testCalculatePRepReward(t *testing.T, revision uint64) {
	const (
		BlockHeight0 uint64 = 0
		BlockHeight1 uint64 = 10
		BlockHeight2 uint64 = 20

		DelegationA0     = 4
		DelegationB0     = 6
		DelegationC0     = 10
		TotalDelegation0 = DelegationA0 + DelegationB0 + DelegationC0

		DelegationA1     = 14
		DelegationB1     = 6
		TotalDelegation1 = DelegationA1 + DelegationB1
	)

	ctx := initTest(1)
	defer finalizeTest(ctx)
	setRevision(ctx, revision)

	prepA := *common.NewAddressFromString("hxaa")
	prepB := *common.NewAddressFromString("hxbb")
	prepC := *common.NewAddressFromString("hxcc")

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = BlockHeight0
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gv.setReward()
	ctx.GV = append(ctx.GV, gv)

	gv = new(GovernanceVariable)
	gv.BlockHeight = BlockHeight1
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
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

	dInfo = new(PRepDelegationInfo)
	dInfo.Address = prepC
	dInfo.DelegatedAmount.SetUint64(DelegationC0)
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
	account, stats, hash := calculatePRepReward(ctx, BlockHeight2)
	assert.Equal(t, uint64(3), account)

	calcDB := ctx.DB.getCalculateDB(prepA)
	bucket, _ := calcDB.GetBucket(db.PrefixIScore)

	var reward, reward0, reward1, totalReward common.HexInt

	// check prepA
	period := common.NewHexIntFromUint64(BlockHeight1 - BlockHeight0)
	gv = ctx.getGVByBlockHeight(BlockHeight1)
	reward0.Mul(&gv.PRepReward.Int, &period.Int)
	reward0.Mul(&reward0.Int, &common.NewHexIntFromUint64(DelegationA0).Int)
	reward0.Div(&reward0.Int, &common.NewHexIntFromUint64(TotalDelegation0).Int)

	var iaPRepA0 *IScoreAccount
	if ctx.Revision < Revision8 {
		iaPRepA0 = newIScoreAccount(prepA, BlockHeight1, reward0)
	} else {
		iaPRepA0 = newIScoreAccount(prepA, BlockHeight2, reward0)
	}

	period = common.NewHexIntFromUint64(BlockHeight2 - BlockHeight1)
	gv = ctx.getGVByBlockHeight(BlockHeight2)
	reward1.Mul(&gv.PRepReward.Int, &period.Int)
	reward1.Mul(&reward1.Int, &common.NewHexIntFromUint64(DelegationA1).Int)
	reward1.Div(&reward1.Int, &common.NewHexIntFromUint64(TotalDelegation1).Int)

	reward.Add(&reward0.Int, &reward1.Int)

	iaPRepA1 := newIScoreAccount(prepA, BlockHeight2, reward)

	bs, _ := bucket.Get(prepA.Bytes())
	ia, _ := NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))
	assert.Equal(t, BlockHeight2, ia.BlockHeight)

	totalReward.Add(&totalReward.Int, &reward.Int)

	// check prepB
	period = common.NewHexIntFromUint64(BlockHeight1 - BlockHeight0)
	gv = ctx.getGVByBlockHeight(BlockHeight1)
	reward0.Mul(&gv.PRepReward.Int, &period.Int)
	reward0.Mul(&reward0.Int, &common.NewHexIntFromUint64(DelegationB0).Int)
	reward0.Div(&reward0.Int, &common.NewHexIntFromUint64(TotalDelegation0).Int)

	var iaPRepB0 *IScoreAccount
	if ctx.Revision < Revision8 {
		iaPRepB0 = newIScoreAccount(prepB, BlockHeight1, reward0)
	} else {
		iaPRepB0 = newIScoreAccount(prepB, BlockHeight2, reward0)
	}

	period = common.NewHexIntFromUint64(BlockHeight2 - BlockHeight1)
	gv = ctx.getGVByBlockHeight(BlockHeight2)
	reward1.Mul(&gv.PRepReward.Int, &period.Int)
	reward1.Mul(&reward1.Int, &common.NewHexIntFromUint64(DelegationB1).Int)
	reward1.Div(&reward1.Int, &common.NewHexIntFromUint64(TotalDelegation1).Int)

	reward.Add(&reward0.Int, &reward1.Int)

	iaPRepB1 := newIScoreAccount(prepB, BlockHeight2, reward)

	bs, _ = bucket.Get(prepB.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward.Cmp(&ia.IScore.Int))
	assert.Equal(t, BlockHeight2, ia.BlockHeight)

	totalReward.Add(&totalReward.Int, &reward.Int)

	// check prepC
	period = common.NewHexIntFromUint64(BlockHeight1 - BlockHeight0)
	gv = ctx.getGVByBlockHeight(BlockHeight1)
	reward0.Mul(&gv.PRepReward.Int, &period.Int)
	reward0.Mul(&reward0.Int, &common.NewHexIntFromUint64(DelegationC0).Int)
	reward0.Div(&reward0.Int, &common.NewHexIntFromUint64(TotalDelegation0).Int)

	var iaPRepC0 *IScoreAccount
	if ctx.Revision < Revision8 {
		iaPRepC0 = newIScoreAccount(prepC, BlockHeight1, reward0)
	} else {
		iaPRepC0 = newIScoreAccount(prepC, BlockHeight2, reward0)
	}

	bs, _ = bucket.Get(prepC.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	assert.Equal(t, 0, reward0.Cmp(&ia.IScore.Int))
	if ctx.Revision < Revision8 {
		assert.Equal(t, BlockHeight1, ia.BlockHeight)
	} else {
		assert.Equal(t, BlockHeight2, ia.BlockHeight)
	}


	totalReward.Add(&totalReward.Int, &reward0.Int)

	// check stats
	assert.Equal(t, 0, totalReward.Cmp(&stats.Int))

	// check state root hash
	stateHash := make([]byte, 64)
	stateHash1 := make([]byte, 64)
	stateHash2 := make([]byte, 64)

	h1 := sha3.NewShake256()
	h1.Write(iaPRepA0.BytesForHash())
	h1.Write(iaPRepB0.BytesForHash())
	h1.Write(iaPRepC0.BytesForHash())
	h1.Read(stateHash1)

	h2 := sha3.NewShake256()
	h2.Write(iaPRepA1.BytesForHash())
	h2.Write(iaPRepB1.BytesForHash())
	h2.Read(stateHash2)

	h := sha3.NewShake256()
	h.Write(stateHash1)
	h.Write(stateHash2)
	h.Read(stateHash)

	assert.Equal(t, stateHash, hash)
}

func TestMsgCalc_CalculateDB(t *testing.T) {
	const (
		rewardRep = minRewardRep

		calculateBlockHeight uint64 = 100

		addr1BlockHeight uint64 = 1
		addr1InitIScore = 100
		addr1DelegationToPRepA = 10 + MinDelegation

		addr2BlockHeight uint64 = 10
		addr2InitIScore = 0
		addr2DelegationToPRepA = 20 + MinDelegation
		addr2DelegationToPRepB = 30 + MinDelegation
	)
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// set GV
	gv := new(GovernanceVariable)
	gv.BlockHeight = 0
	gv.MainPRepCount.SetUint64(NumMainPRep)
	gv.SubPRepCount.SetUint64(NumSubPRep)
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
	count, stats, hash := calculateDB(ctx.Rollback.GetChannel(), 0, queryDB, calcDB, ctx,
		calculateBlockHeight, writeBatchCount)

	var reward, totalReward uint64
	stateHash := make([]byte, 64)
	h := sha3.NewShake256()

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
	assert.Equal(t, reward, ia.IScore.Uint64())
	assert.Equal(t, calculateBlockHeight, ia.BlockHeight)

	totalReward += reward
	totalReward -= addr1InitIScore

	ia.Address = addr1
	h.Write(ia.BytesForHash())

	// check - addr2
	period = calculateBlockHeight - addr2BlockHeight
	gv = ctx.getGVByBlockHeight(addr2BlockHeight)
	if gv == nil {
		assert.True(t, false)
		return
	}
	reward = gv.RewardRep.Uint64() * period * (addr2DelegationToPRepA + addr2DelegationToPRepB) / rewardDivider + addr2InitIScore

	bs, _ = bucket.Get(addr2.Bytes())
	ia, _ = NewIScoreAccountFromBytes(bs)
	assert.Equal(t, reward, ia.IScore.Uint64())
	assert.Equal(t, calculateBlockHeight, ia.BlockHeight)

	totalReward += reward
	totalReward -= addr2InitIScore

	ia.Address = addr2
	h.Write(ia.BytesForHash())

	// check stats
	assert.Equal(t, count, stats.Accounts)
	assert.Equal(t, totalReward, stats.Beta3.Uint64())

	// check state root hash
	h.Read(stateHash)
	assert.Equal(t, stateHash, hash)
}

func TestMsgCalc_DoCalculate_Error(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	iissDBDir := testDBDir + "/iiss"
	req := CalculateRequest{Path: iissDBDir, BlockHeight:100, BlockHash: testHash}

	// get CALCULATE message while processing CALCULATE message
	ctx.DB.setCalculatingBH(uint64(50))
	err, blockHeight, _, _ := DoCalculate(ctx.Rollback.GetChannel(), ctx, &req, nil, 0)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "calculating now. drop calculate message"))
	assert.Equal(t, req.BlockHeight, blockHeight)
	ctx.DB.resetCalculatingBH()

	// get CALCULATE message with no IISS data
	err, blockHeight, _, _ = DoCalculate(ctx.Rollback.GetChannel(), ctx, &req, nil, 0)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "Failed to load IISS data"))
	assert.Equal(t, req.BlockHeight, blockHeight)

	// write IISS data DB
	_, iissDB := writeHeader(testDBDir, "iiss", req.BlockHeight)
	iissDB.Close()
	defer os.RemoveAll(iissDBDir)

	// get CALCULATE message with invalid block height
	ctx.DB.setCalcDoneBH(uint64(200))
	err, blockHeight, _, _ = DoCalculate(ctx.Rollback.GetChannel(), ctx, &req, nil, 0)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "too low blockHeight"))
	assert.Equal(t, req.BlockHeight, blockHeight)

	// get CALCULATE message with duplicated block height
	ctx.DB.setCalcDoneBH(uint64(100))
	ctx.DB.setCalculatingBH(uint64(100))
	err, blockHeight, _, _ = DoCalculate(ctx.Rollback.GetChannel(), ctx, &req, nil, 0)
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "duplicated block"))
	assert.Equal(t, req.BlockHeight, blockHeight)

	// Cancel with ROLLBACK
	ctx.DB.setCalcDoneBH(uint64(50))
	ctx.DB.setCalculatingBH(uint64(50))

	quitChannel := ctx.Rollback.GetChannel()
	ctx.Rollback.notifyRollback()
	err, blockHeight, _, _ = DoCalculate(quitChannel, ctx, &req, nil, 0)
	assert.Error(t, err)
	assert.True(t, strings.HasSuffix(err.Error(), "was canceled by ROLLBACK"))
}

func newIScoreAccount(addr common.Address, blockHeight uint64, reward common.HexInt) *IScoreAccount {
	ia := new(IScoreAccount)
	ia.Address = addr
	ia.BlockHeight = blockHeight
	ia.IScore.Set(&reward.Int)

	return ia
}

func TestMsgQueryCalc_DoQueryCalculateStatus(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)
	var resp QueryCalculateStatusResponse

	DoQueryCalculateStatus(ctx, &resp)
	assert.Equal(t, CalculationDone, resp.Status)
	assert.Equal(t, uint64(0), resp.BlockHeight)

	// start calculation
	calcBH := uint64(1000)
	ctx.DB.setCalculatingBH(calcBH)

	DoQueryCalculateStatus(ctx, &resp)
	assert.Equal(t, CalculationDoing, resp.Status)
	assert.Equal(t, calcBH, resp.BlockHeight)

	// end calculation
	ctx.DB.setCalcDoneBH(calcBH)

	DoQueryCalculateStatus(ctx, &resp)
	assert.Equal(t, CalculationDone, resp.Status)
	assert.Equal(t, calcBH, resp.BlockHeight)
}

func TestMsgQueryCalc_DoQueryCalculateResult(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)
	var resp QueryCalculateResultResponse
	var blockHeight uint64 = 1000
	var iScore uint64 = 10

	DoQueryCalculateResult(ctx, blockHeight, &resp)
	assert.Equal(t, InvalidBH, resp.Status)
	assert.Equal(t, blockHeight, resp.BlockHeight)

	// start calculation
	ctx.DB.setCalculatingBH(blockHeight)

	DoQueryCalculateResult(ctx, blockHeight, &resp)
	assert.Equal(t, calcDoing, resp.Status)
	assert.Equal(t, blockHeight, resp.BlockHeight)

	// end calculation
	ctx.DB.setCalcDoneBH(blockHeight)

	crDB := ctx.DB.getCalculateResultDB()
	stats := new(Statistics)
	stats.TotalReward.SetUint64(iScore)
	stateHash := make([]byte, 64)
	binary.BigEndian.PutUint64(stateHash, blockHeight)

	WriteCalculationResult(crDB, blockHeight, stats, stateHash)

	DoQueryCalculateResult(ctx, blockHeight, &resp)
	assert.Equal(t, calcSucceeded, resp.Status)
	assert.Equal(t, blockHeight, resp.BlockHeight)
	assert.Equal(t, 0, resp.IScore.Cmp(&stats.TotalReward.Int))
	assert.Equal(t, stateHash, resp.StateHash)
}

func Test_isCalcCancelByRollback(t *testing.T) {
	assert.True(t, isCalcCancelByRollback(&CalcCancelByRollbackError{}))
	assert.False(t, isCalcCancelByRollback(&os.PathError{}))
}
