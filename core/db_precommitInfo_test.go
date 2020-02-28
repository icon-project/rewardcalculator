package core

import (
	"encoding/binary"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makePreCommitHierarchy() *PreCommitHierarchy {
	preCommitHierarchy := new(PreCommitHierarchy)
	var blockHash [BlockHashSize]byte
	var childrenHash1 [BlockHashSize]byte
	var childrenHash2 [BlockHashSize]byte
	binary.BigEndian.PutUint64(blockHash[:], 33)
	binary.BigEndian.PutUint64(childrenHash1[:], 34)
	binary.BigEndian.PutUint64(childrenHash2[:], 35)
	preCommitHierarchy.blockHash = blockHash
	preCommitHierarchy.childrenBlockHashes = append(preCommitHierarchy.childrenBlockHashes, childrenHash1)
	preCommitHierarchy.childrenBlockHashes = append(preCommitHierarchy.childrenBlockHashes, childrenHash2)
	return preCommitHierarchy
}

func TestDBPreCommitHierarchy_ID(t *testing.T) {
	preCommitHierarchy := makePreCommitHierarchy()

	assert.Equal(t, preCommitHierarchy.blockHash, preCommitHierarchy.ID())
}

func TestDBPreCommitHierarchy_BytesAndSetBytes(t *testing.T) {
	preCommitHierarchy := makePreCommitHierarchy()

	var preCommitHierarchyNew PreCommitHierarchy

	bs, err := preCommitHierarchy.Bytes()
	assert.NoError(t, err)
	err = preCommitHierarchyNew.SetBytes(bs)
	assert.NoError(t, err)
	bsNew, err := preCommitHierarchyNew.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, preCommitHierarchy.childrenBlockHashes, preCommitHierarchyNew.childrenBlockHashes)
	assert.Equal(t, bs, bsNew)
}

func TestDBPreCommitHierarchy_NewPreCommitHierarchy(t *testing.T) {
	preCommitHierarchy := makePreCommitHierarchy()

	bs, err := preCommitHierarchy.Bytes()
	assert.NoError(t, err)
	preCommitHierarchyNew, err := NewPreCommitHierarchyFromBytes(bs)
	assert.NoError(t, err)
	bsNew, err := preCommitHierarchyNew.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, preCommitHierarchy.childrenBlockHashes, preCommitHierarchyNew.childrenBlockHashes)
	assert.Equal(t, bs, bsNew)
}

func TestDBPreCommitInfo_PreCommitHierarchy(t *testing.T) {
	var preCommitHierarchy PreCommitHierarchy

	ctx := initTest(1)
	defer finalizeTest(ctx)
	pdb := ctx.DB.preCommitHierarchy

	var blockHash [BlockHashSize]byte
	var childrenHash1 [BlockHashSize]byte
	var childrenHash2 [BlockHashSize]byte
	binary.BigEndian.PutUint64(blockHash[:], 33)
	binary.BigEndian.PutUint64(childrenHash1[:], 34)
	binary.BigEndian.PutUint64(childrenHash2[:], 35)

	err := AppendPreCommitChildInDB(pdb, blockHash, childrenHash1)
	assert.NoError(t, err)
	err = AppendPreCommitChildInDB(pdb, blockHash, childrenHash2)
	assert.NoError(t, err)

	bucket, err := pdb.GetBucket(db.PrefixIScore)
	assert.NoError(t, err)
	bs, err := bucket.Get(blockHash[:])
	assert.NoError(t, err)
	err = preCommitHierarchy.SetBytes(bs)
	assert.NoError(t, err)

	childrenHashesBytes, _ := preCommitHierarchy.Bytes()
	assert.Equal(t, childrenHashesBytes, bs)

	DeletePreCommitHierarchy(pdb, blockHash[:])
	bs, _ = bucket.Get(common.Uint64ToBytes(calcBlockHeight))
	assert.Nil(t, bs)
}

func TestLoadPreCommitInfo(t *testing.T) {
	ctx := initTest(1)
	defer finalizeTest(ctx)
	pdb := ctx.DB.preCommitHierarchy

	var blockHash [BlockHashSize]byte
	var childrenHash1 [BlockHashSize]byte
	var childrenHash2 [BlockHashSize]byte
	binary.BigEndian.PutUint64(blockHash[:], 33)
	binary.BigEndian.PutUint64(childrenHash1[:], 34)
	binary.BigEndian.PutUint64(childrenHash2[:], 35)

	preCommitInfo := make(PreCommitInfo, 0)
	preCommitInfo[blockHash] = make(map[[BlockHashSize]byte]bool, 0)
	preCommitInfo[blockHash][childrenHash1] = true
	preCommitInfo[blockHash][childrenHash2] = true

	err := AppendPreCommitChildInDB(pdb, blockHash, childrenHash1)
	assert.NoError(t, err)
	err = AppendPreCommitChildInDB(pdb, blockHash, childrenHash2)
	assert.NoError(t, err)

	loadedPreCommitInfo := LoadPreCommitInfo(pdb)

	assert.Equal(t, preCommitInfo, loadedPreCommitInfo)
}