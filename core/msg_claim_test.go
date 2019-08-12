package core

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
)

func TestMsgClaim_PreCommit(t *testing.T) {
	tests := [] struct {
		blockHeight uint64
		blockHash   []byte
		address     *common.Address
	}{
		{blockHeight: 1, blockHash: []byte("1-1"), address: common.NewAddressFromString("hx11")},
		{blockHeight: 1, blockHash: []byte("1-2"), address: common.NewAddressFromString("hx12")},
	}

	ctx := initTest(1)
	defer finalizeTest()

	var claim *Claim

	// Query and add
	for _, tt := range tests {
		claim = ctx.preCommit.queryAndAdd(tt.blockHeight, tt.blockHash, *tt.address)
		assert.Nil(t, claim)
		claim = ctx.preCommit.queryAndAdd(tt.blockHeight, tt.blockHash, *tt.address)
		assert.NotNil(t, claim)
		assert.True(t, tt.address.Equal(&claim.Address))
	}

	// update - success
	ia := new(IScoreAccount)
	ia.Address = *tests[0].address
	ia.BlockHeight = tests[0].blockHeight
	ia.IScore.SetUint64(100)
	err := ctx.preCommit.update(tests[0].blockHeight, tests[0].blockHash, ia)
	assert.NoError(t, err)
	claim = ctx.preCommit.queryAndAdd(tests[0].blockHeight, tests[0].blockHash, ia.Address)
	assert.NotNil(t, claim)
	assert.Equal(t, ia.IScore.Uint64(), claim.Data.IScore.Uint64())
	assert.Equal(t, ia.BlockHeight, claim.Data.BlockHeight)
	assert.False(t, claim.Confirmed)

	// update - invalid block height
	err = ctx.preCommit.update(0, tests[0].blockHash, ia)
	assert.Error(t, err)

	// update - invalid address
	ia.Address = *common.NewAddressFromString("hx22")
	err = ctx.preCommit.update(tests[0].blockHeight, tests[0].blockHash, ia)
	assert.Error(t, err)
	ia.Address = *tests[0].address

	// revert
	err = ctx.preCommit.revert(tests[0].blockHeight, tests[0].blockHash, ia.Address)
	assert.NoError(t, err)
	claim = ctx.preCommit.queryAndAdd(tests[0].blockHeight, tests[0].blockHash, ia.Address)
	assert.NotNil(t, claim)
	assert.Equal(t, uint64(0), claim.Data.IScore.Uint64())
	assert.Equal(t, uint64(0), claim.Data.BlockHeight)
	assert.True(t, claim.Confirmed)

	// revert - invalid block height
	err = ctx.preCommit.revert(0, tests[0].blockHash, ia.Address)
	assert.Error(t, err)

	// revert - invalid address
	err = ctx.preCommit.revert(tests[0].blockHeight, tests[0].blockHash, *common.NewAddressFromString("hx22"))
	assert.Error(t, err)

	// delete
	ret := ctx.preCommit.delete(tests[0].blockHeight, tests[0].blockHash, *tests[0].address)
	assert.True(t, ret)

	// delete - invalid claim
	ret = ctx.preCommit.delete(tests[0].blockHeight, tests[0].blockHash, *tests[0].address)
	assert.False(t, ret)

	// flush precommitData
	pcLen := len(ctx.preCommit.dataList)
	ctx.preCommit.flush(tests[1].blockHeight, tests[1].blockHash)
	assert.Equal(t, pcLen - 1, len(ctx.preCommit.dataList))

	// make data for write test
	ia.Address = *tests[0].address
	for _, tt := range tests {
		ctx.preCommit.queryAndAdd(tt.blockHeight, tt.blockHash, *tt.address)
		err = ctx.preCommit.update(tt.blockHeight, tt.blockHash, ia)
	}

	// write to claim DB without commit - there is no data to write
	ctx.preCommit.writeClaimToDB(ctx, tests[0].blockHeight, tests[0].blockHash)
	claimDB := ctx.DB.getClaimDB()
	bucket, _ := claimDB.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(tests[0].address.Bytes())
	assert.NoError(t, err)
	assert.Nil(t, bs)

	// make data for commit & write test
	ia.Address = *tests[0].address
	for _, tt := range tests {
		ctx.preCommit.queryAndAdd(tt.blockHeight, tt.blockHash, *tt.address)
		err = ctx.preCommit.update(tt.blockHeight, tt.blockHash, ia)
	}

	// commit
	err = ctx.preCommit.commit(tests[0].blockHeight, tests[0].blockHash, ia.Address)
	assert.NoError(t, err)
	claim = ctx.preCommit.queryAndAdd(tests[0].blockHeight, tests[0].blockHash, ia.Address)
	assert.NotNil(t, claim)
	assert.Equal(t, uint64(0), claim.PrevData.IScore.Uint64())
	assert.Equal(t, uint64(0), claim.PrevData.BlockHeight)
	assert.True(t, claim.Confirmed)

	// commit - block height
	err = ctx.preCommit.commit(0, tests[0].blockHash, ia.Address)
	assert.Error(t, err)

	// commit - invalid address
	err = ctx.preCommit.commit(tests[0].blockHeight, tests[0].blockHash, *common.NewAddressFromString("hx22"))
	assert.Error(t, err)

	// write to claim DB
	ctx.preCommit.writeClaimToDB(ctx, tests[0].blockHeight, tests[0].blockHash)
	claimDB = ctx.DB.getClaimDB()
	bucket, _ = claimDB.GetBucket(db.PrefixIScore)
	bs, err = bucket.Get(tests[0].address.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, bs)

	claim.SetBytes(bs)
	assert.Equal(t, tests[0].blockHeight, claim.Data.BlockHeight)
	assert.Equal(t, 0, claim.Data.IScore.Int.Cmp(&ia.IScore.Int))
	assert.Equal(t, 0, len(ctx.preCommit.dataList))
}

func TestMsgClaim_DoClaim(t *testing.T) {
	const (
		db1IScore = claimMinIScore + 100
		db2IScore = claimMinIScore + 2000
	)
	address := common.NewAddressFromString("hx11")
	dbContent0 := IScoreAccount { Address: *address }
	dbContent0.BlockHeight = 10
	dbContent0.IScore.SetUint64(100)

	dbContent1 := IScoreAccount { Address: *address }
	dbContent1.BlockHeight = 100
	dbContent1.IScore.SetUint64(db1IScore)

	dbContent2 := IScoreAccount { Address: *address }
	dbContent2.BlockHeight = 200
	dbContent2.IScore.SetUint64(db2IScore)

	claim :=
		ClaimMessage {BlockHeight: 101, BlockHash: []byte("1-1"), Address: *address}

	invalidAddressClaim :=
		ClaimMessage {BlockHeight: 101, BlockHash: []byte("1-1"), Address: *common.NewAddressFromString("hx33")}

	alreadyClaimedInCurrentPeriodClaim :=
		ClaimMessage{BlockHeight: 102, BlockHash: []byte("1-1"), Address: *address}

	ctx := initTest(1)
	defer finalizeTest()

	// write content to Query DB
	queryDB := ctx.DB.getQueryDB(dbContent0.Address)
	bucket, _ := queryDB.GetBucket(db.PrefixIScore)
	bucket.Set(dbContent0.ID(), dbContent0.Bytes())

	// claim I-Score less than 1000
	blockHeight, iScore := DoClaim(ctx, &claim)
	assert.Equal(t, uint64(0), blockHeight)
	assert.Nil(t, iScore)

	// update Query DB
	bucket.Set(dbContent1.ID(), dbContent1.Bytes())

	// claim I-Score
	blockHeight, iScore = DoClaim(ctx, &claim)
	assert.Equal(t, dbContent1.BlockHeight, blockHeight)
	assert.Equal(t, uint64(db1IScore - (db1IScore % claimMinIScore)), iScore.Uint64())

	// commit claim - true
	commit :=
		CommitClaim{Success:true, Address: claim.Address, BlockHeight:claim.BlockHeight, BlockHash:claim.BlockHash}
	DoCommitClaim(ctx, &commit)

	// already claimed in current block
	blockHeight, iScore = DoClaim(ctx, &claim)
	assert.Equal(t, dbContent1.BlockHeight, blockHeight)
	assert.Nil(t, iScore)

	// write claim to DB
	ctx.preCommit.writeClaimToDB(ctx, claim.BlockHeight, claim.BlockHash)

	// invalid address
	blockHeight, iScore = DoClaim(ctx, &invalidAddressClaim)
	assert.Equal(t, uint64(0), blockHeight)
	assert.Nil(t, iScore)

	// already claimed in current period
	blockHeight, iScore = DoClaim(ctx, &alreadyClaimedInCurrentPeriodClaim)
	assert.Equal(t, dbContent1.BlockHeight, blockHeight)
	assert.Nil(t, iScore)

	// update Query DB
	bucket.Set(dbContent2.ID(), dbContent2.Bytes())

	// claim I-Score in next period
	claim.BlockHeight = 201
	blockHeight, iScore = DoClaim(ctx, &claim)
	assert.Equal(t, dbContent2.BlockHeight, blockHeight)
	var iScoreExpected common.HexInt
	iScoreExpected.Sub(&dbContent2.IScore.Int, &dbContent1.IScore.Int)
	// db2Iscore - claimed IScore
	assert.Equal(t, uint64(db2IScore - (db2IScore % claimMinIScore) - (db1IScore - (db1IScore % claimMinIScore))),
		iScore.Uint64())
}
