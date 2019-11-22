package core

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/stretchr/testify/assert"
)

var testHash = []byte{
	0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
	0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
	0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
	0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef,
}

var zeroHash = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

const (
	claimAddress               = "hxa"
	claimIScore uint64         = 10
	claimBlockHeight uint64    = 1
	claimTXIndex uint64        = 0
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

func TestDBClaim_BackupID(t *testing.T) {
	claim := makeClaim()

	blockHeight := uint64(123)
	id := claim.BackupID(blockHeight)

	assert.Equal(t, ClaimBackupIDSize, len(id))
	assert.Equal(t, blockHeight, common.BytesToUint64(id[:BlockHeightSize]))
	assert.Equal(t, claim.ID(), id[BlockHeightSize:])
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

	assert.NoError(t, err)

	assert.Equal(t, 0, claim.Data.IScore.Cmp(&newClaim.Data.IScore.Int))
	assert.Equal(t, claim.Data.BlockHeight, newClaim.Data.BlockHeight)
	assert.Equal(t, claim.Bytes(), newClaim.Bytes())
}

func makePreCommit() *PreCommit {
	claim := makeClaim()
	preCommit := newPreCommit(claimBlockHeight, testHash, claimTXIndex, zeroHash, claim.Address)
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
	assert.NoError(t, err)

	pcNew.SetBytes(bs)
	bsNew, err := pcNew.Bytes()
	assert.NoError(t, err)

	assert.True(t, pc.Data.equal(&pcNew.Data))
	assert.Equal(t, pc.Confirmed, pcNew.Confirmed)
	assert.Equal(t, bs, bsNew)
}

func TestDBPreCommit_newPreCommit(t *testing.T) {
	hx11 := common.NewAddressFromString("hx11")
	preCommit := newPreCommit(claimBlockHeight, testHash, claimTXIndex, testHash, *hx11)

	assert.Equal(t, claimBlockHeight, preCommit.BlockHeight)
	assert.Equal(t, testHash, preCommit.BlockHash)
	assert.Equal(t, claimTXIndex, preCommit.TXIndex)
	assert.Equal(t, testHash, preCommit.TXHash)
	assert.Equal(t, hx11.String(), preCommit.Address.String())
}

func TestDBPreCommit_query(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	pc := makePreCommit()
	pc.Confirmed = true
	pcDB := ctx.DB.getPreCommitDB()
	bucket, _ := pcDB.GetBucket(db.PrefixClaim)
	bs, _ := pc.Bytes()

	assert.False(t, pc.query(pcDB))

	// write to preCommit DB
	bucket.Set(pc.ID(), bs)

	newPC := makePreCommit()
	assert.False(t, newPC.Confirmed)
	assert.True(t, newPC.query(pcDB))
	assert.True(t, newPC.Confirmed)
	newPC.BlockHeight = 321
	assert.False(t, newPC.query(pcDB))
}

func TestDBPreCommit_write_delete(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	pc := makePreCommit()
	pcDB := ctx.DB.getPreCommitDB()

	assert.False(t, pc.query(pcDB))

	// write to preCommit DB
	assert.NoError(t, pc.write(pcDB, nil))
	assert.True(t, pc.query(pcDB))

	iScore := common.NewHexIntFromUint64(uint64(100))

	assert.NoError(t, pc.write(pcDB, iScore))
	assert.True(t, pc.query(pcDB))

	// delete from preCommit DB
	assert.NoError(t, pc.delete(pcDB))
	assert.False(t, pc.query(pcDB))
}

func TestDBPreCommit_commit_revert(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	pc := makePreCommit()
	pcDB := ctx.DB.getPreCommitDB()

	// write to preCommit DB
	assert.NoError(t, pc.write(pcDB, nil))
	assert.True(t, pc.query(pcDB))

	// revert unconfirmed entry from preCommit DB
	assert.NoError(t, pc.revert(pcDB))
	assert.False(t, pc.query(pcDB))

	// write to preCommit DB
	assert.NoError(t, pc.write(pcDB, nil))
	assert.True(t, pc.query(pcDB))

	// commit entry
	assert.False(t, pc.Confirmed)
	assert.NoError(t, pc.commit(pcDB))
	assert.True(t, pc.query(pcDB))
	assert.True(t, pc.Confirmed)

	// revert confirmed entry from preCommit DB
	assert.NoError(t, pc.revert(pcDB))
	assert.True(t, pc.query(pcDB))
}

func TestDBPreCommit_manage(t *testing.T) {
	tests := [] struct {
		blockHeight uint64
		txIndex     uint64
		hash        []byte
		address     *common.Address
	}{
		{blockHeight: 1, txIndex: 0, hash: []byte{0x01, 0x01}, address: common.NewAddressFromString("hx11")},
		{blockHeight: 1, txIndex: 0, hash: []byte{0x01, 0x02}, address: common.NewAddressFromString("hx12")},
	}
	iScore := uint64(100)

	ctx := initTest(1)
	defer finalizeTest(ctx)

	pcDB := ctx.DB.getPreCommitDB()

	for i, tt := range tests {
		pc := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
		// query no ent
		assert.False(t, pc.query(pcDB))

		// write
		assert.NoError(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
		assert.Equal(t, iScore, pc.Data.IScore.Uint64())
		assert.False(t, pc.Confirmed)

		// delete
		assert.NoError(t, pc.delete(pcDB))
		assert.False(t, pc.query(pcDB))

		// rewrite to test commit
		assert.NoError(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
		assert.Equal(t, iScore, pc.Data.IScore.Uint64())

		// commit and query confirmed preCommit
		if i != len(tests) - 1 {
			assert.NoError(t, pc.commit(pcDB))
			assert.True(t, pc.Confirmed)
			pc2 := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
			assert.True(t, pc2.query(pcDB))
			assert.Equal(t, pc.Data.IScore.Uint64(), pc2.Data.IScore.Uint64())

			// revert - confirmed precommit
			assert.NoError(t, pc.revert(pcDB))
			// can query
			assert.True(t, pc.query(pcDB))
		} else {
			// do not commit last one

			// commit - invalid blockHeight
			pc = newPreCommit(tt.blockHeight + 1, tt.hash, tt.txIndex, tt.hash, *tt.address)
			assert.Error(t, pc.commit(pcDB))

			// commit - invalid blockHash
			pc = newPreCommit(tt.blockHeight, nil, tt.txIndex, tt.hash, *tt.address)
			assert.Error(t, pc.commit(pcDB))

			// commit - invalid txIndex
			pc = newPreCommit(tt.blockHeight, tt.hash, tt.txIndex + 1, tt.hash, *tt.address)
			pcConfirmed := pc.Confirmed
			assert.NoError(t, pc.commit(pcDB))
			// no change confirmed flag
			assert.Equal(t, pcConfirmed, pc.Confirmed)

			// revert - invalid blockHeight
			pc = newPreCommit(tt.blockHeight + 1, tt.hash, tt.txIndex, tt.hash, *tt.address)
			assert.Error(t, pc.revert(pcDB))

			// revert - invalid blockHash
			pc = newPreCommit(tt.blockHeight, nil, tt.txIndex, tt.hash, *tt.address)
			assert.Error(t, pc.revert(pcDB))

			// revert - invalid txIndex
			pc = newPreCommit(tt.blockHeight, tt.hash, tt.txIndex + 1, tt.hash, *tt.address)
			pcConfirmed = pc.Confirmed
			assert.NoError(t, pc.revert(pcDB))
			// no change confirmed flag
			assert.Equal(t, pcConfirmed, pc.Confirmed)
			// can query
			assert.True(t, pc.query(pcDB))

			// revert
			pc = newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
			assert.NoError(t, pc.revert(pcDB))
			// can't query
			assert.False(t, pc.query(pcDB))

			// rewrite to writePreCommitToClaimDB()
			assert.NoError(t, pc.write(pcDB, common.NewHexIntFromUint64(iScore)))
			assert.Equal(t, iScore, pc.Data.IScore.Uint64())
			// can query
			assert.True(t, pc.query(pcDB))
		}
	}

	// write to claim DB with commit
	cDB := ctx.DB.getClaimDB()
	assert.NoError(t, writePreCommitToClaimDB(pcDB, cDB, ctx.DB.getClaimBackupDB(),
		tests[0].blockHeight, tests[0].hash))

	// can't query commited preCommit data
	pc := newPreCommit(tests[0].blockHeight, tests[0].hash, tests[0].txIndex, tests[0].hash, *tests[0].address)
	assert.False(t, pc.query(pcDB))

	// can't query not commited preCommit data
	pc = newPreCommit(tests[1].blockHeight, tests[1].hash, tests[1].txIndex, tests[1].hash, *tests[1].address)
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

func TestDBPreCommit_deletePreCommits(t *testing.T) {
	tests := []struct {
		blockHeight uint64
		txIndex     uint64
		hash        []byte
		address     *common.Address
	}{
		{blockHeight: 1, txIndex:0, hash: []byte{0x01, 0x01}, address: common.NewAddressFromString("hx11")},
		{blockHeight: 1, txIndex:1, hash: []byte{0x01, 0x02}, address: common.NewAddressFromString("hx12")},
		{blockHeight: 2, txIndex:0, hash: []byte{0x02, 0x01}, address: common.NewAddressFromString("hx21")},
		{blockHeight: 2, txIndex:1, hash: []byte{0x02, 0x02}, address: common.NewAddressFromString("hx22")},
		{blockHeight: 3, txIndex:0, hash: []byte{0x03, 0x01}, address: common.NewAddressFromString("hx31")},
		{blockHeight: 3, txIndex:1, hash: []byte{0x03, 0x02}, address: common.NewAddressFromString("hx32")},
		{blockHeight: 4, txIndex:0, hash: []byte{0x04, 0x01}, address: common.NewAddressFromString("hx41")},
		{blockHeight: 4, txIndex:1, hash: []byte{0x04, 0x02}, address: common.NewAddressFromString("hx42")},
	}

	ctx := initTest(1)
	defer finalizeTest(ctx)

	pcDB := ctx.DB.getPreCommitDB()

	for _, tt := range tests {
		pc := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
		assert.NoError(t, pc.write(pcDB, nil), "Write() failed with preCommit(%s)", pc.String())
	}

	// check initPreCommit
	initData := tests[1]
	assert.NoError(t, initPreCommit(pcDB, initData.blockHeight))
	for _, tt := range tests {
		pc := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
		if pc.BlockHeight > initData.blockHeight {
			assert.False(t, pc.query(pcDB), "initPreCommit() failed with preCommit(%s)", pc.String())
		} else {
			assert.True(t, pc.query(pcDB), "initPreCommit() failed with preCommit(%s)", pc.String())
		}
	}

	// check flushPreCommit with blockHash
	flushData := tests[1]
	assert.NoError(t, flushPreCommit(pcDB, flushData.blockHeight, flushData.hash))
	for _, tt := range tests {
		if tt.blockHeight == flushData.blockHeight {
			pc := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
			if bytes.Compare(tt.hash, flushData.hash) == 0 {
				assert.False(t, pc.query(pcDB), "flushPreCommit() failed with preCommit(%s)", pc.String())
			} else {
				assert.True(t, pc.query(pcDB), "flushPreCommit() failed with preCommit(%s)", pc.String())
			}
		}
	}

	// check flushPreCommit without blockHash
	flushData = tests[0]
	assert.NoError(t, flushPreCommit(pcDB, flushData.blockHeight, nil))
	for _, tt := range tests {
		if tt.blockHeight == flushData.blockHeight {
			pc := newPreCommit(tt.blockHeight, tt.hash, tt.txIndex, tt.hash, *tt.address)
			assert.False(t, pc.query(pcDB), "flushPreCommit(noHash) failed with preCommit(%s)", pc.String())
		}
	}
}

type backupClaimData struct {
	blockHeight uint64
	claim       Claim
}

func makeBackupClaimData(bucket db.Bucket) []backupClaimData {
	// make claim backup data
	backupClaim := make([]backupClaimData, 5)
	backupClaim = []backupClaimData {
		{
			blockHeight: 10,
			claim: Claim {
				Address: *common.NewAddressFromString("hxa"),
				Data: ClaimData {
					BlockHeight: 9,
					IScore: *common.NewHexIntFromUint64(uint64(1000)),
				},
			},
		},
		{
			blockHeight: 11,
			claim: Claim {
				Address: *common.NewAddressFromString("hxa"),
				Data: ClaimData {
					BlockHeight: 10,
					IScore: *common.NewHexIntFromUint64(uint64(500)),
				},
			},
		},
		{
			blockHeight: 11,
			claim: Claim {
				Address: *common.NewAddressFromString("hxb"),
				Data: ClaimData {
					BlockHeight: 10,
					IScore: *common.NewHexIntFromUint64(uint64(2000)),
				},
			},
		},
		{
			blockHeight: 1,
			claim: Claim {
				Address: *common.NewAddressFromString("hxa"),
				Data: ClaimData {
					BlockHeight: 0,
					IScore: *common.NewHexIntFromUint64(uint64(0)),
				},
			},
		},
		{
			blockHeight: 5,
			claim: Claim {
				Address: *common.NewAddressFromString("hxb"),
				Data: ClaimData {
					BlockHeight: 0,
					IScore: *common.NewHexIntFromUint64(uint64(0)),
				},
			},
		},
	}

	for _, data := range backupClaim {
		key := data.claim.BackupID(data.blockHeight)
		value := data.claim.Bytes()
		bucket.Set(key, value)

		// check write
		bs, err := bucket.Get(data.claim.BackupID(data.blockHeight))
		if err != nil || bs == nil {
			return nil
		}
	}

	return backupClaim
}

func Test_writeClaimBackupInfo(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	const (
		blockHeight uint64 = 100
	)

	cbDB := ctx.DB.getClaimBackupDB()

	err := writeClaimBackupInfo(cbDB, blockHeight)
	assert.NoError(t, err)

	var cbInfo ClaimBackupInfo
	cbBucket, _ := cbDB.GetBucket(db.PrefixManagement)
	bs, err := cbBucket.Get(cbInfo.ID())
	assert.NotNil(t, bs)
	assert.NoError(t, err)
	err = cbInfo.SetBytes(bs)
	assert.NoError(t, err)
	assert.Equal(t, blockHeight, cbInfo.FirstBlockHeight)
	assert.Equal(t, blockHeight, cbInfo.LastBlockHeight)

	// write invalid blockHeight
	err = writeClaimBackupInfo(cbDB, blockHeight - 10)
	assert.NoError(t, err)
	bs, err = cbBucket.Get(cbInfo.ID())
	assert.NotNil(t, bs)
	assert.NoError(t, err)
	err = cbInfo.SetBytes(bs)
	assert.NoError(t, err)
	assert.Equal(t, blockHeight, cbInfo.FirstBlockHeight)
	assert.Equal(t, blockHeight, cbInfo.LastBlockHeight)

	// write valid blockHeight
	err = writeClaimBackupInfo(cbDB, blockHeight + 1)
	assert.NoError(t, err)
	bs, err = cbBucket.Get(cbInfo.ID())
	assert.NotNil(t, bs)
	assert.NoError(t, err)
	err = cbInfo.SetBytes(bs)
	assert.NoError(t, err)
	assert.Equal(t, blockHeight, cbInfo.FirstBlockHeight)
	assert.Equal(t, blockHeight + 1, cbInfo.LastBlockHeight)

	// write valid blockHeight
	err = writeClaimBackupInfo(cbDB, blockHeight + ClaimBackupPeriod + 1)
	assert.NoError(t, err)
	bs, err = cbBucket.Get(cbInfo.ID())
	assert.NotNil(t, bs)
	assert.NoError(t, err)
	err = cbInfo.SetBytes(bs)
	assert.NoError(t, err)
	assert.Equal(t, blockHeight + 1, cbInfo.FirstBlockHeight)
	assert.Equal(t, blockHeight + ClaimBackupPeriod + 1, cbInfo.LastBlockHeight)
}

func Test_garbageCollectClaimBackupDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	cbDB := ctx.DB.getClaimBackupDB()
	bucket, _ := cbDB.GetBucket(db.PrefixClaim)

	backupClaim := makeBackupClaimData(bucket)
	assert.NotNil(t, backupClaim)

	// do garbage collection
	garbageCollectClaimBackupDB(cbDB, 0, backupClaim[0].blockHeight)

	// check result
	bs, err := bucket.Get(backupClaim[0].claim.BackupID(backupClaim[0].blockHeight))
	assert.NoError(t, err)
	assert.Nil(t, bs)
	bs, err = bucket.Get(backupClaim[1].claim.BackupID(backupClaim[1].blockHeight))
	assert.NoError(t, err)
	assert.NotNil(t, bs)
	bs, err = bucket.Get(backupClaim[2].claim.BackupID(backupClaim[2].blockHeight))
	assert.NoError(t, err)
	assert.NotNil(t, bs)

	// do garbage collection
	garbageCollectClaimBackupDB(cbDB, backupClaim[0].blockHeight, backupClaim[1].blockHeight)

	// check result
	bs, err = bucket.Get(backupClaim[1].claim.BackupID(backupClaim[1].blockHeight))
	assert.NoError(t, err)
	assert.Nil(t, bs)
	bs, err = bucket.Get(backupClaim[2].claim.BackupID(backupClaim[2].blockHeight))
	assert.NoError(t, err)
	assert.Nil(t, bs)
}

func TestDBClaim_checkClaimDBRollback(t *testing.T) {
	cbInfo := ClaimBackupInfo {
		FirstBlockHeight: 100,
		LastBlockHeight: 200,
	}

	tests := []struct {
		name string
		rollback uint64
		ok bool
		error bool
	}{
		{
			name: "Too low",
			rollback: cbInfo.FirstBlockHeight - 1,
			ok: false,
			error: true,
		},
		{
			name:     "Too high",
			rollback: cbInfo.LastBlockHeight + 1,
			ok: false,
			error:    false,
		},
		{
			name:     "good",
			rollback: cbInfo.LastBlockHeight - 1,
			ok: true,
			error:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ok, err := checkClaimDBRollback(&cbInfo, tt.rollback); err != nil {
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

func TestDBClaim_rollbackClaimDB(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)

	cDB := ctx.DB.getClaimDB()
	cBucket, _ := cDB.GetBucket(db.PrefixClaim)
	cbDB := ctx.DB.getClaimBackupDB()
	cbBucket, _ := cbDB.GetBucket(db.PrefixClaim)

	backupClaim := makeBackupClaimData(cbBucket)
	assert.NotNil(t, backupClaim)

	// Rollback
	for i := uint64(11) ; i > 0; i-- {
		err := _rollbackClaimDB(cbDB, cBucket, i)
		assert.NoError(t, err)
		checkRollbackResult(t, cbBucket, cBucket, backupClaim, i)
	}
}

func checkRollbackResult(t *testing.T,
	cbBucket db.Bucket, cBucket db.Bucket, backupClaim []backupClaimData, blockHeight uint64) {
	backupMap := make(map[common.Address]backupClaimData)
	for _, data := range backupClaim {
		if data.blockHeight >= blockHeight {
			// make backupMap for claim DB checking
			backup, ok := backupMap[data.claim.Address]
			if ok == false || backup.blockHeight > data.blockHeight {
				backupMap[data.claim.Address] = data
			}

			// check claim backup DB
			if data.blockHeight == blockHeight {
				bs, _ := cbBucket.Get(data.claim.BackupID(blockHeight))
				assert.Nil(t, bs)
			}
		}
	}


	// check claim DB value
	for _, v := range backupMap {
		bs, _ := cBucket.Get(v.claim.ID())
		if v.claim.Data.BlockHeight == 0 && v.claim.Data.IScore.Sign() == 0 {
			assert.Nil(t, bs)
		} else {
			assert.NotNil(t, bs)
			assert.Equal(t, v.claim.Bytes(), bs)
		}
	}
}
