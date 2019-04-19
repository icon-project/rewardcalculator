package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"io/ioutil"
	"os"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
)

func TestMsg_DoQuery(t *testing.T) {
	address := common.NewAddressFromString("hx11")
	dbContent0 := IScoreAccount { Address: *address }
	dbContent0.BlockHeight = 100
	dbContent0.IScore.SetUint64(100)

	claim :=
		ClaimMessage{BlockHeight: 101, BlockHash: []byte("1-1"), Address: *address}

	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	ctx, _ := NewContext(dir, string(db.GoLevelDBBackend), "test", 2)

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
	ctx.preCommit.writeClaimToDB(ctx, claim.BlockHeight, claim.BlockHash)

	// Query to claimed Account after commit
	resp = DoQuery(ctx, *address)
	assert.Equal(t, dbContent0.BlockHeight, resp.BlockHeight)
	assert.Equal(t, 0, resp.IScore.Sign())
}
