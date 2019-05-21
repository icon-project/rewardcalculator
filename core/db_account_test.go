package core

import (
	"fmt"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/stretchr/testify/assert"
)

const (
	iaAddress               = "hxa"
	iaIScore uint64         = 10
	iaBlockHeight uint64    = 1
	delegationAddress       = "hx1"
	delegationAmount uint64 = 2
)

func makeIA() *IScoreAccount {
	ia := new(IScoreAccount)

	ia.Address = *common.NewAddressFromString(iaAddress)
	ia.IScore.SetUint64(iaIScore)
	ia.BlockHeight = iaBlockHeight
	ia.Delegations = make([]*DelegateData, 0)
	delegation := new(DelegateData)
	delegation.Address = *common.NewAddressFromString(delegationAddress)
	delegation.Delegate.SetUint64(delegationAmount)
	ia.Delegations = append(ia.Delegations, delegation)

	return ia
}

func TestDBAccount_ID(t *testing.T) {
	ia := makeIA()

	assert.Equal(t, ia.Address.Bytes(), ia.ID())
}

func TestDBAccount_BytesAndSetBytes(t *testing.T) {
	ia := makeIA()

	var iaNew IScoreAccount

	iaNew.SetBytes(ia.Bytes())

	assert.Equal(t, 0, ia.IScore.Cmp(&iaNew.IScore.Int))
	assert.Equal(t, ia.BlockHeight, iaNew.BlockHeight)
	assert.Equal(t, len(ia.Delegations), len(iaNew.Delegations))
	assert.True(t, ia.Delegations[0].Address.Equal(&iaNew.Delegations[0].Address))
	assert.Equal(t, 0, ia.Delegations[0].Delegate.Cmp(&iaNew.Delegations[0].Delegate.Int))
	assert.Equal(t, ia.Bytes(), iaNew.Bytes())
}

func TestDBAccount_NewIScoreAccountFromBytes(t *testing.T) {
	ia := makeIA()

	iaNew, err := NewIScoreAccountFromBytes(ia.Bytes())

	assert.Nil(t, err)

	assert.Equal(t, 0, ia.IScore.Cmp(&iaNew.IScore.Int))
	assert.Equal(t, ia.BlockHeight, iaNew.BlockHeight)
	assert.Equal(t, len(ia.Delegations), len(iaNew.Delegations))
	assert.True(t, ia.Delegations[0].Address.Equal(&iaNew.Delegations[0].Address))
	assert.Equal(t, 0, ia.Delegations[0].Delegate.Cmp(&iaNew.Delegations[0].Delegate.Int))
	assert.Equal(t, ia.Bytes(), iaNew.Bytes())
}

func TestDBAccount_NewIScoreAccountFromIISS(t *testing.T) {
	// set delegation
	tx := new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = iaBlockHeight
	tx.Address = *common.NewAddressFromString(iaAddress)
	tx.DataType = TXDataTypeDelegate
	tx.Data = new(codec.TypedObj)

	delegation := make([]interface{}, 0)

	dgData := make([]interface{}, 0)
	dgData = append(dgData, common.NewAddressFromString(delegationAddress))
	dgData = append(dgData, MinDelegation)
	delegation = append(delegation, dgData)
	var err error
	tx.Data, err = common.EncodeAny(delegation)
	if err != nil {
		fmt.Printf("Can't encode delegation. err=%+v\n", err)
		return
	}

	ia := NewIScoreAccountFromIISS(tx)

	assert.True(t, ia.Address.Equal(common.NewAddressFromString(iaAddress)))
	assert.Equal(t, uint64(0), ia.IScore.Uint64())
	assert.Equal(t, iaBlockHeight, ia.BlockHeight)
	assert.Equal(t, 1, len(ia.Delegations))
	assert.True(t, ia.Delegations[0].Address.Equal(common.NewAddressFromString(delegationAddress)))
	assert.Equal(t, uint64(MinDelegation), ia.Delegations[0].Delegate.Uint64())

	// register P-Rep
	tx = new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = iaBlockHeight
	tx.Address = *common.NewAddressFromString(iaAddress)
	tx.DataType = TXDataTypePRepReg

	ia = NewIScoreAccountFromIISS(tx)

	assert.True(t, ia.Address.Equal(common.NewAddressFromString(iaAddress)))
	assert.Equal(t, uint64(0), ia.IScore.Uint64())
	assert.Equal(t, iaBlockHeight, ia.BlockHeight)
	assert.Nil(t, ia.Delegations)

	// unregister P-Rep
	tx = new(IISSTX)
	tx.Index = 0
	tx.BlockHeight = iaBlockHeight
	tx.Address = *common.NewAddressFromString(iaAddress)
	tx.DataType = TXDataTypePRepUnReg

	ia = NewIScoreAccountFromIISS(tx)

	assert.True(t, ia.Address.Equal(common.NewAddressFromString(iaAddress)))
	assert.Equal(t, uint64(0), ia.IScore.Uint64())
	assert.Equal(t, iaBlockHeight, ia.BlockHeight)
	assert.Nil(t, ia.Delegations)
}
