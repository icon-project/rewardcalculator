package core

import (
	"encoding/hex"
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
	txHash := make([]byte, TXHashSize)
	bs, _ := hex.DecodeString("abcd0123")
	copy(txHash, bs)

	query := &Query{Address: *address}
	queryWithTXHash := &Query{Address: *address, TXHash: txHash}
	claim := ClaimMessage{
		BlockHeight: 101,
		BlockHash: []byte("1-1"),
		Address: *address,
		TXIndex: 0,
		TXHash: txHash,
	}
	commit := CommitClaim{
		Success:true,
		BlockHeight:claim.BlockHeight,
		BlockHash:claim.BlockHash,
		Address:claim.Address,
		TXIndex: 0,
		TXHash: txHash,
	}

	ctx := initTest(1)
	defer finalizeTest(ctx)

	// write content to Query DB
	queryDB := ctx.DB.getQueryDB(dbContent0.Address)
	bucket, _ := queryDB.GetBucket(db.PrefixIScore)
	bucket.Set(dbContent0.ID(), dbContent0.Bytes())

	// Query
	resp := DoQuery(ctx, query)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// claim I-Score
	DoClaim(ctx, &claim)

	// Query to claimed Account before commit
	resp = DoQuery(ctx, query)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// Query with TX hash to claimed Account before commit
	resp = DoQuery(ctx, queryWithTXHash)
	assert.Equal(t, dbContent0.BlockHeight+1, resp.BlockHeight)
	assert.Equal(t, int64(0), resp.IScore.Int64())

	// commit claim
	DoCommitClaim(ctx, &commit)

	// Query to claimed Account before commit to claim DB
	resp = DoQuery(ctx, query)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, dbContent0.IScore.Cmp(&resp.IScore.Int))

	// commit to claim DB
	writePreCommitToClaimDB(ctx.DB.getPreCommitDB(), ctx.DB.getClaimDB(), ctx.DB.getClaimBackupDB(),
		claim.BlockHeight, claim.BlockHash)

	// Query to claimed Account after commit
	resp = DoQuery(ctx, query)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, int64(100), resp.IScore.Int64())
}

func TestMsg_DoInit(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	// invalid - too high blockHeight
	err := DoInit(ctx, 100)
	assert.Error(t, err)

	// valid blockHeight
	err = DoInit(ctx, 0)
	assert.NoError(t, err)
}
