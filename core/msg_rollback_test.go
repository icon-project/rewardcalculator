package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMsgRollback_checkRollback(t *testing.T) {
	ctx := initTest(2)
	defer finalizeTest(ctx)

	const calcBlockHeight1 uint64 = 100
	const calcBlockHeight2 uint64 = 200
	ctx.DB.setCalcDoneBH(calcBlockHeight1)
	ctx.DB.setCalcDoneBH(calcBlockHeight2)

	tests := []struct {
		name     string
		rollback uint64
		error    bool
	}{
		{
			name:     "too low1",
			rollback: calcBlockHeight1 - 1,
			error:    true,
		},
		{
			name:     "too low2",
			rollback: calcBlockHeight1,
			error:    true,
		},
		{
			name:     "good",
			rollback: calcBlockHeight1 + 1,
			error:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkRollback(ctx, tt.rollback); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't. rollback:%d", tt.rollback)
					return
				}
			}
		})
	}
}

func TestMsgRollback_checkAccountDBRollback(t *testing.T) {
	ctx := initTest(2)
	defer finalizeTest(ctx)

	const calcBlockHeight uint64 = 100
	ctx.DB.setCalcDoneBH(calcBlockHeight)
	assert.True(t, checkAccountDBRollback(ctx, calcBlockHeight-1))
	assert.True(t, checkAccountDBRollback(ctx, calcBlockHeight))
	assert.False(t, checkAccountDBRollback(ctx, calcBlockHeight+1))
}

func TestRollback_newChannel(t *testing.T) {
	var c CancelCalculation

	assert.Nil(t, c.channel)
	c.newChannel()
	assert.NotNil(t, c.channel)
}

func TestRollback_getChannel(t *testing.T) {
	var c CancelCalculation
	c.newChannel()

	assert.Equal(t, c.channel, c.GetChannel())
}

func TestRollback_notifyRollback(t *testing.T) {
	var c CancelCalculation
	c.newChannel()

	oldChannel := c.GetChannel()

	c.notifyRollback()
	assert.NotEqual(t, oldChannel, c.GetChannel())
	assert.NotNil(t, c.GetChannel())
}
