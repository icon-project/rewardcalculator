package core

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStatistics_Set(t *testing.T) {
	var stats statistics
	var err error

	// invalid field name
	err = stats.Set("InvalidField", 2)
	assert.NotNil(t, err)

	// invalid type
	err = stats.Set("Accounts", 2)
	assert.NotNil(t, err)

	// Accounts
	err = stats.Set("Accounts", uint64(2))
	assert.Equal(t, uint64(2), stats.Accounts)
	assert.Nil(t, err)

	// IScore
	err = stats.Set("IScore", *common.NewHexInt(1))
	assert.Equal(t, uint64(1), stats.IScore.Uint64())
	assert.Nil(t, err)
}

func TestStatistics_Increase(t *testing.T) {
	const (
		initAccounts uint64 = 1
		initIScore uint64 = 1
	)

	stats := statistics{
		Accounts: initAccounts,
		IScore: *common.NewHexIntFromUint64(initIScore),
	}
	var err error

	// Accounts
	err = stats.Increase("Accounts", initAccounts)
	assert.Equal(t, 2*initAccounts, stats.Accounts)
	assert.Nil(t, err)

	// IScore
	err = stats.Increase("IScore", *common.NewHexIntFromUint64(initIScore))
	assert.Equal(t, 2*initIScore, stats.IScore.Uint64())
	assert.Nil(t, err)
}

func TestStatistics_Decrease(t *testing.T) {
	const (
		initAccounts uint64 = 1
		initIScore uint64 = 1
	)

	stats := statistics{
		Accounts: initAccounts,
		IScore: *common.NewHexIntFromUint64(initIScore),
	}
	var err error

	// Accounts
	err = stats.Decrease("Accounts", initAccounts)
	assert.Equal(t, uint64(0), stats.Accounts)
	assert.Nil(t, err)

	// IScore
	err = stats.Decrease("IScore", *common.NewHexIntFromUint64(initIScore))
	assert.Equal(t, uint64(0), stats.IScore.Uint64())
	assert.Nil(t, err)
	err = stats.Decrease("IScore", *common.NewHexIntFromUint64(initIScore))
	assert.Equal(t, int64(-1), stats.IScore.Int64())
	assert.Nil(t, err)
}
