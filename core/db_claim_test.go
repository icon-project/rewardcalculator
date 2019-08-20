package core

import (
	"testing"

	"github.com/icon-project/rewardcalculator/common"
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

	assert.Equal(t, 0, claim.Data.IScore.Cmp(&claimNew.Data.IScore.Int))
	assert.Equal(t, claim.Data.BlockHeight, claimNew.Data.BlockHeight)
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
