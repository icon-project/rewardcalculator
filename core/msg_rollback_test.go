package core

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestMsgRollback_checkRollback(t *testing.T) {
	ctx := initTest(2)
	defer finalizeTest(ctx)

	const calcBlockHeight1 uint64 = 100
	const calcBlockHeight2 uint64 = 200
	ctx.DB.setBlockInfo(calcBlockHeight1, []byte(string(calcBlockHeight1)))
	ctx.DB.setBlockInfo(calcBlockHeight2, []byte(string(calcBlockHeight2)))

	tests := []struct {
		name string
		rollback uint64
		ok bool
		error bool
	} {
		{
			name: "too low1",
			rollback: calcBlockHeight1 - 1,
			ok: false,
			error: true,
		},
		{
			name: "too low2",
			rollback: calcBlockHeight1,
			ok: false,
			error: true,
		},
		{
			name: "good",
			rollback: calcBlockHeight1 + 1,
			ok: true,
			error: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ok, err := checkRollback(ctx, tt.rollback); err != nil {
				if !tt.error {
					t.Error(err)
				}
				return
			} else {
				if tt.error {
					t.Errorf("It expects error but it doesn't. rollback:%d", tt.rollback)
					return
				}
				if ok != tt.ok {
					t.Errorf("It expects %s but it returns %s", strconv.FormatBool(tt.ok), strconv.FormatBool(ok))
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
	ctx.DB.setCalculateBlockHeight(calcBlockHeight)
	assert.True(t, checkAccountDBRollback(ctx, calcBlockHeight - 1))
	assert.False(t, checkAccountDBRollback(ctx, calcBlockHeight))
	assert.False(t, checkAccountDBRollback(ctx, calcBlockHeight + 1))

	tests := []struct {
		name string
		rollback uint64
		ok bool
	} {
		{
			name: "too low1",
			rollback: calcBlockHeight - 1,
			ok: true,
		},
		{
			name: "too low2",
			rollback: calcBlockHeight,
			ok: false,
		},
		{
			name: "good",
			rollback: calcBlockHeight + 1,
			ok: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := checkAccountDBRollback(ctx, tt.rollback)
			if ok != tt.ok {
				t.Errorf("It expects %s but it returns %s", strconv.FormatBool(tt.ok), strconv.FormatBool(ok))
				return
			}
		})
	}
}

func TestRollback_newChannel(t *testing.T) {
	var rb Rollback

	assert.Nil(t, rb.channel)
	rb.newChannel()
	assert.NotNil(t, rb.channel)
}

func TestRollback_getChannel(t *testing.T) {
	var rb Rollback
	rb.newChannel()

	assert.Equal(t, rb.channel, rb.GetChannel())
}

func TestRollback_notifyRollback(t *testing.T) {
	var rb Rollback
	rb.newChannel()

	oldChannel := rb.GetChannel()

	rb.notifyRollback()
	assert.NotEqual(t, oldChannel, rb.GetChannel())
	assert.NotNil(t, rb.GetChannel())
}
