package core

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
)


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
	assert.Equal(t, claim.BlockHeight, blockHeight)
	assert.Nil(t, iScore)

	// write claim to DB
	//ctx.preCommit.writeClaimToDB(ctx, claim.BlockHeight, claim.BlockHash)
	writePreCommitToClaimDB(ctx.DB.getPreCommitDB(), ctx.DB.getClaimDB(), claim.BlockHeight, claim.BlockHash)

	// invalid address
	blockHeight, iScore = DoClaim(ctx, &invalidAddressClaim)
	assert.Equal(t, uint64(0), blockHeight)
	assert.Nil(t, iScore)

	// already claimed in current period
	blockHeight, iScore = DoClaim(ctx, &alreadyClaimedInCurrentPeriodClaim)
	assert.Equal(t, uint64(0), blockHeight)
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
