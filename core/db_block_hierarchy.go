package core

import (
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"log"
)

type ChildrenHashInfo struct {
	blockHash           [BlockHashSize]byte
	childrenBlockHashes [][BlockHashSize]byte
}

func (chi *ChildrenHashInfo) ID() [BlockHashSize]byte {
	return chi.blockHash
}

func (chi *ChildrenHashInfo) Bytes() ([]byte, error) {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(&chi.childrenBlockHashes); err != nil {
		return nil, err
	} else {
		bytes = bs
	}
	return bytes, nil
}

func (chi *ChildrenHashInfo) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &chi.childrenBlockHashes)
	if err != nil {
		return err
	}
	return nil
}

func NewChildrenHashInfoFromBytes(bs []byte) (*ChildrenHashInfo, error) {
	chi := new(ChildrenHashInfo)
	if err := chi.SetBytes(bs); err != nil {
		return nil, err
	}
	return chi, nil
}

func AppendChildHashInDB(pdb db.Database, prevBlockHash [BlockHashSize]byte, blockHash [BlockHashSize]byte) error {
	bucket, _ := pdb.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(prevBlockHash[:])
	if err != nil {
		log.Printf("Error while getting PrecommitHieriachy")
		return err
	}
	childrenHashInfo := new(ChildrenHashInfo)
	childrenHashInfo.SetBytes(bs)
	childrenHashInfo.childrenBlockHashes = append(childrenHashInfo.childrenBlockHashes, blockHash)
	data, _ := childrenHashInfo.Bytes()
	bucket.Set(prevBlockHash[:], data)
	return nil
}

func DeleteChildrenHashInfo(pdb db.Database, blockHash []byte) {
	bucket, err := pdb.GetBucket(db.PrefixIScore)
	if err != nil {
		log.Printf("Error while getting childrenHashInfo bucket")
		return
	}
	bucket.Delete(blockHash)
}

type BlockHierarchy map[[BlockHashSize]byte]map[[BlockHashSize]byte]bool

func LoadBlockHierarchy(pdb db.Database) (blockHierarchy BlockHierarchy) {
	blockHierarchy = make(BlockHierarchy, 0)
	iter, err := pdb.GetIterator()
	if err != nil {
		log.Printf("Error while getting BlockHierarchy DB iterator")
		return
	}
	var hashKey [BlockHashSize]byte
	childrenHashInfo := new(ChildrenHashInfo)
	iter.New(nil, nil)
	for iter.Next() {
		copy(hashKey[:], iter.Key()[:BlockHashSize])
		blockHierarchy[hashKey] = make(map[[BlockHashSize]byte]bool, 0)
		childrenHashInfo.SetBytes(iter.Value())
		for _, child := range childrenHashInfo.childrenBlockHashes {
			blockHierarchy[hashKey][child] = true
		}
	}
	iter.Release()

	if err := iter.Error(); err != nil {
		log.Printf("Error while iterate BlockHierarchy DB")
		return make(BlockHierarchy, 0)
	}
	return
}
