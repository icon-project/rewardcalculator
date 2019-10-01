package core

import (
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
)

const (
	claimAddress               = "hxa"
	claimIScore uint64         = 10
	claimBlockHeight uint64    = 1
)

func makeClaim() *Claim {
	claim := new(Claim)

	claim.Address = *common.NewAddressFromString(claimAddress)
	claim.Data.IScore.SetUint64(claimIScore)
	claim.Data.BlockHeight = claimBlockHeight

	return claim
}

func TestDBClaim_ID(t *testing.T) {
	claim := makeClaim()

	assert.Equal(t, claim.Address.Bytes(), claim.ID())
}

func TestDBClaim_BytesAndSetBytes(t *testing.T) {
	claim := makeClaim()

	var claimNew Claim

	claimNew.SetBytes(claim.Bytes())

	assert.True(t, claim.Data.equal(&claimNew.Data))
	assert.Equal(t, claim.Bytes(), claimNew.Bytes())
}

func TestDBClaim_NewClaimFromBytes(t *testing.T) {
	claim := makeClaim()

	newClaim, err := NewClaimFromBytes(claim.Bytes())

	assert.Nil(t, err)

	assert.Equal(t, 0, claim.Data.IScore.Cmp(&newClaim.Data.IScore.Int))
	assert.Equal(t, claim.Data.BlockHeight, newClaim.Data.BlockHeight)
	assert.Equal(t, claim.Bytes(), newClaim.Bytes())
}

var testBlockHash = []byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34}

func makePreCommit() *PreCommit {
	claim := makeClaim()
	preCommit := newPreCommit(claimBlockHeight, testBlockHash, claim.Address)
	preCommit.Confirmed = false
	preCommit.Data = claim.Data

	return preCommit
}

func TestDBPreCommit_ID(t *testing.T) {
	pc := makePreCommit()

	id := pc.ID()

	assert.Equal(t, 8 + 32 + 21, len(id))
	assert.Equal(t, pc.BlockHeight, common.BytesToUint64(id[:8]))
	assert.Equal(t, pc.BlockHash, id[8:8+32])
	assert.Equal(t, pc.Address.Bytes(), id[8+32:])
}

func TestDBPreCommit_BytesAndSetBytes(t *testing.T) {
	pc := makePreCommit()

	var pcNew PreCommit

	bs, err := pc.Bytes()
	assert.Nil(t, err)

	pcNew.SetBytes(bs)
	bsNew, err := pcNew.Bytes()
	assert.Nil(t, err)

	assert.True(t, pc.Data.equal(&pcNew.Data))
	assert.Equal(t, pc.Confirmed, pcNew.Confirmed)
	assert.Equal(t, bs, bsNew)
}

func TestDBPreCommit_manage(t *testing.T) {
	tests := [] struct {
		blockHeight uint64
		blockHash   []byte
		address     *common.Address
	}{
		{blockHeight: 1, blockHash: []byte{0x01, 0x01}, address: common.NewAddressFromString("hx11")},
		{blockHeight: 1, blockHash: []byte{0x01, 0x02}, address: common.NewAddressFromString("hx12")},
	}
	iScore := uint64(100)

	ctx := initTest(1)
	defer finalizeTest()

	pcDB := ctx.DB.getPreCommitDB()

	for i, tt := range tests {
		pc := newPreCommit(tt.blockHeight, tt.blockHash, *tt.address)
		// query no ent
		assert.False(t, pc.query(pcDB))

		// write
		assert.Nil(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
		assert.Equal(t, iScore, pc.Data.IScore.Uint64())
		assert.False(t, pc.Confirmed)

		// delete
		assert.Nil(t, pc.delete(pcDB))
		assert.False(t, pc.query(pcDB))

		// rewrite to test commit
		assert.Nil(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
		assert.Equal(t, iScore, pc.Data.IScore.Uint64())

		// commit and query confirmed preCommit
		if i != len(tests) - 1 {
			assert.Nil(t, pc.commit(pcDB))
			assert.True(t, pc.Confirmed)
			pc2 := newPreCommit(tt.blockHeight, tt.blockHash, *tt.address)
			assert.True(t, pc2.query(pcDB))
			assert.Equal(t, pc.Data.IScore.Uint64(), pc2.Data.IScore.Uint64())

			// revert - confirmed precommit
			assert.Nil(t, pc.revert(pcDB))
			// can query
			assert.True(t, pc.query(pcDB))
		} else {
			// do not commit last one

			// commit - invalid blockHeight
			pc = newPreCommit(tt.blockHeight + 1, tt.blockHash, *tt.address)
			assert.Error(t, pc.commit(pcDB))

			// commit - invalid blockHash
			pc = newPreCommit(tt.blockHeight, nil, *tt.address)
			assert.Error(t, pc.commit(pcDB))

			// revert - invalid blockHeight
			pc = newPreCommit(tt.blockHeight + 1, tt.blockHash, *tt.address)
			assert.Error(t, pc.revert(pcDB))

			// revert - invalid blockHash
			pc = newPreCommit(tt.blockHeight, nil, *tt.address)
			assert.Error(t, pc.revert(pcDB))

			// revert
			pc = newPreCommit(tt.blockHeight, tt.blockHash, *tt.address)
			assert.Nil(t, pc.revert(pcDB))
			// can't query
			assert.False(t, pc.query(pcDB))

			// rewrite to writePreCommitToClaimDB()
			assert.Nil(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
			assert.Equal(t, iScore, pc.Data.IScore.Uint64())
			// can query
			assert.True(t, pc.query(pcDB))
		}
	}

	// write to claim DB with commit
	cDB := ctx.DB.getClaimDB()
	assert.Nil(t, writePreCommitToClaimDB(pcDB, cDB, tests[0].blockHeight, tests[0].blockHash))

	// can't query commited precommit data
	pc := newPreCommit(tests[0].blockHeight, tests[0].blockHash, *tests[0].address)
	assert.False(t, pc.query(pcDB))

	// can't query not commited precommit data
	pc = newPreCommit(tests[1].blockHeight, tests[1].blockHash, *tests[1].address)
	assert.False(t, pc.query(pcDB))

	// read claim data from claimDB
	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(tests[0].address.Bytes())
	assert.NoError(t, err)
	assert.NotNil(t, bs)

	var claim Claim
	claim.SetBytes(bs)

	assert.Equal(t, tests[0].blockHeight, claim.Data.BlockHeight)
	assert.Equal(t, iScore, claim.Data.IScore.Uint64())
}
