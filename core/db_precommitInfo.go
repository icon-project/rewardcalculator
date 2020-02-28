package core

import (
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"log"
)

type PreCommitHierarchy struct {
	blockHash           [BlockHashSize]byte
	childrenBlockHashes [][BlockHashSize]byte
}

func (ph *PreCommitHierarchy) ID() [BlockHashSize]byte {
	return ph.blockHash
}

func (ph *PreCommitHierarchy) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&ph.childrenBlockHashes); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (ph *PreCommitHierarchy) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &ph.childrenBlockHashes)
	if err != nil {
		return err
	}
	return nil
}

func NewPreCommitHierarchyFromBytes(bs []byte) (*PreCommitHierarchy, error) {
	ph := new(PreCommitHierarchy)
	if err := ph.SetBytes(bs); err != nil {
		return nil, err
	}
	return ph, nil
}

func AppendPreCommitChildInDB(pdb db.Database, prevBlockHash [BlockHashSize]byte, blockHash [BlockHashSize]byte) error {
	bucket, _ := pdb.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(prevBlockHash[:])
	if err != nil {
		log.Printf("Error while getting PrecommitHieriachy")
		return err
	}
	preCommitHierarchy := new(PreCommitHierarchy)
	preCommitHierarchy.SetBytes(bs)
	preCommitHierarchy.childrenBlockHashes = append(preCommitHierarchy.childrenBlockHashes, blockHash)
	data, _ := preCommitHierarchy.Bytes()
	bucket.Set(prevBlockHash[:], data)
	return nil
}

func DeletePreCommitHierarchy(pdb db.Database, blockHash []byte) {
	bucket, err := pdb.GetBucket(db.PrefixIScore)
	if err != nil {
		log.Printf("Error while getting preCommitHierarchy bucket")
		return
	}
	bucket.Delete(blockHash)
}

type PreCommitInfo map[[BlockHashSize]byte]map[[BlockHashSize]byte]bool

func LoadPreCommitInfo(pdb db.Database) (preCommitInfo PreCommitInfo) {
	preCommitInfo = make(PreCommitInfo, 0)
	iter, err := pdb.GetIterator()
	if err != nil {
		log.Printf("Error while getting PreCommitInfo DB iterator")
		return
	}
	var hashKey [BlockHashSize]byte
	preCommitHierarchy := new(PreCommitHierarchy)
	iter.New(nil, nil)
	for iter.Next() {
		copy(hashKey[:], iter.Key()[:BlockHashSize])
		preCommitInfo[hashKey] = make(map[[BlockHashSize]byte]bool, 0)
		preCommitHierarchy.SetBytes(iter.Value())
		for _, child := range preCommitHierarchy.childrenBlockHashes {
			preCommitInfo[hashKey][child] = true
		}
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		log.Printf("Error while iterate PreCommitInfo DB")
		return make(PreCommitInfo, 0)
	}
	return
}
