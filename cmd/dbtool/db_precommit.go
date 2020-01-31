package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryPreCommitDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.address == "" && input.height == 0 {
		err = printDB(input.path, util.BytesPrefix([]byte(db.PrefixClaim)), printPreCommit)
	} else {
		dir, name := filepath.Split(input.path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.address)
		err = queryPreCommits(qdb, address, input.height)
	}
	return
}

func queryPreCommits(qdb db.Database, address *common.Address, blockHeight uint64) error {
	qPreCommitKeys, err := getKeys(qdb, address, blockHeight)
	if err != nil {
		return err
	}
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		fmt.Println("Failed to get preCommit Bucket")
		return err
	}

	for _, key := range qPreCommitKeys {
		value, err := bucket.Get(key)
		if err != nil {
			fmt.Println("Error while get preCommit")
			return err
		}
		if value == nil {
			continue
		}
		pc, err := newPreCommit(key, value)
		if err != nil {
			return err
		} else {
			printPreCommitInstance(pc)
		}
	}
	return nil
}

func printPreCommit(key []byte, value []byte) error {
	if pc, e := newPreCommit(key, value); e != nil {
		return e
	} else {
		printPreCommitInstance(pc)
		return nil
	}
}

func printPreCommitInstance(pc *core.PreCommit) {
	fmt.Printf("%s\n", pc.String())
}

func newPreCommit(key []byte, value []byte) (pc *core.PreCommit, err error) {
	pc = new(core.PreCommit)

	err = pc.SetBytes(value)
	if err != nil {
		fmt.Printf("Failed to initialize preCommit instance\n")
		return nil, err

	}
	pc.BlockHeight = common.BytesToUint64(key[:core.BlockHeightSize])
	pc.BlockHash = make([]byte, core.BlockHashSize)
	copy(pc.BlockHash, key[core.BlockHeightSize:core.BlockHeightSize+core.BlockHashSize])
	return pc, nil
}

func getKeys(qdb db.Database, address *common.Address, blockHeight uint64) ([][]byte, error) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("Failed to get precommit db iterator")
		return nil, err
	}

	preCommitKeys := make([][]byte, 0)
	iter.New(nil, nil)
	keyExist := false
	tmpAddress := new(common.Address)
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		if address.Equal(tmpAddress) == false && blockHeight != 0 {
			if bytes.Equal(key[core.BlockHeightSize-len(blockHeightBytesValue):core.BlockHeightSize], blockHeightBytesValue) &&
				bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], address.Bytes()) {
				keyExist = true
				preCommitKeys = append(preCommitKeys, key)
				break
			}
		} else {
			if address.Equal(tmpAddress) == false &&
				bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], address.Bytes()) {
				keyExist = true
				preCommitKeys = append(preCommitKeys, key)
			} else if blockHeight != 0 && bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue) {
				keyExist = true
				preCommitKeys = append(preCommitKeys, key)
			}
		}
	}
	iter.Release()

	if keyExist == false {
		fmt.Println("Can not find key using given information")
		return nil, errors.New("preCommit key does not exiest")
	}
	err = iter.Error()
	if err != nil {
		fmt.Println("Error while iterate")
		return nil, err
	}

	return preCommitKeys, err
}
