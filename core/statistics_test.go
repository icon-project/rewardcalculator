package core

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStatistics_Set(t *testing.T) {
	var stats Statistics
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
	err = stats.Set("Beta3", *common.NewHexInt(1))
	assert.Equal(t, uint64(1), stats.Beta3.Uint64())
	assert.Nil(t, err)
}

func TestStatistics_Increase(t *testing.T) {
	const (
		initAccounts uint64 = 1
		initIScore int64 = 100
	)

	stats := Statistics {
		Accounts: initAccounts,
		Beta3:    *common.NewHexInt(initIScore),
	}
	var err error

	// Accounts
	err = stats.Increase("Accounts", initAccounts)
	assert.Equal(t, 2*initAccounts, stats.Accounts)
	assert.Nil(t, err)

	// IScore
	err = stats.Increase("Beta3", *common.NewHexInt(initIScore))
	assert.Equal(t, 2*initIScore, stats.Beta3.Int64())
	assert.Nil(t, err)

	// IScore
	err = stats.Increase("Beta3", *common.NewHexInt(-initIScore))
	assert.Equal(t, initIScore, stats.Beta3.Int64())
}

func TestStatistics_Decrease(t *testing.T) {
	const (
		initAccounts uint64 = 1
		initIScore int64 = 100
	)

	stats := Statistics {
		Accounts: initAccounts,
		Beta3:    *common.NewHexInt(initIScore),
	}
	var err error

	// Accounts
	err = stats.Decrease("Accounts", initAccounts)
	assert.Equal(t, uint64(0), stats.Accounts)
	assert.Nil(t, err)

	// Beta3
	err = stats.Decrease("Beta3", *common.NewHexInt(initIScore))
	assert.Equal(t, int64(0), stats.Beta3.Int64())
	assert.Nil(t, err)
	err = stats.Decrease("Beta3", *common.NewHexInt(initIScore))
	assert.Equal(t, 0 - initIScore, stats.Beta3.Int64())
	assert.Nil(t, err)
}
