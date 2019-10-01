package core

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
)

func TestMsg_DoQuery(t *testing.T) {
	address := common.NewAddressFromString("hx11")
	dbContent0 := IScoreAccount { Address: *address }
	dbContent0.BlockHeight = 100
	dbContent0.IScore.SetUint64(claimMinIScore + 100)

	claim := ClaimMessage{BlockHeight: 101, BlockHash: []byte("1-1"), Address: *address}
	commit := CommitClaim{Success:true, BlockHeight:claim.BlockHeight, BlockHash:claim.BlockHash, Address:claim.Address}

	ctx := initTest(1)
	defer finalizeTest()

	// write content to Query DB
	queryDB := ctx.DB.getQueryDB(dbContent0.Address)
	bucket, _ := queryDB.GetBucket(db.PrefixIScore)
	bucket.Set(dbContent0.ID(), dbContent0.Bytes())

	// Query
	resp := DoQuery(ctx, *address)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// claim I-Score
	DoClaim(ctx, &claim)

	// Query to claimed Account before commit
	resp = DoQuery(ctx, *address)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// commit claim
	DoCommitClaim(ctx, &commit)

	// Query to claimed Account before commit to claim DB
	resp = DoQuery(ctx, *address)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// commit to claim DB
	writePreCommitToClaimDB(ctx.DB.getPreCommitDB(), ctx.DB.getClaimDB(), claim.BlockHeight, claim.BlockHash)

	// Query to claimed Account after commit
	resp = DoQuery(ctx, *address)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, resp.IScore.Cmp(&common.NewHexIntFromUint64(100).Int))
}
