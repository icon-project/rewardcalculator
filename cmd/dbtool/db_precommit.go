package main

import (
	"bytes"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"path/filepath"
)

func queryPreCommitDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}

	if input.address == "" && input.height == 0 {
		printAllEntriesInPath(input.path, util.BytesPrefix([]byte(db.PrefixClaim)), printPreCommit)
	} else {
		dir, name := filepath.Split(input.path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.address)
		runQueryPreCommits(qdb, address, input.height)
	}
}

func runQueryPreCommits(qdb db.Database, address *common.Address, blockHeight uint64) {
	qPreCommitKeys := getKeys(qdb, address, blockHeight)

	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		fmt.Printf("Failed to get preCommit Bucket")
		os.Exit(1)
	}

	for _, key := range qPreCommitKeys {
		value, err := bucket.Get(key)
		if err != nil {
			fmt.Println("Error while get preCommit")
			os.Exit(1)
		}
		if value == nil {
			continue
		}
		printPreCommit(key, value)
	}
}

func printPreCommit(key []byte, value []byte) {
	pc := getPreCommit(key, value)

	fmt.Printf("%s\n", pc.String())
}

func getPreCommit(key []byte, value []byte) *core.PreCommit {
	pc := new(core.PreCommit)

	err := pc.SetBytes(value)
	if err != nil {
		fmt.Printf("Failed to initialize preCommit instance")
		os.Exit(1)

	}
	pc.BlockHeight = common.BytesToUint64(key[:core.BlockHeightSize])
	pc.BlockHash = make([]byte, core.BlockHashSize)
	copy(pc.BlockHash, key[core.BlockHeightSize:core.BlockHeightSize+core.BlockHashSize])
	return pc
}

func getKeys(qdb db.Database, address *common.Address, blockHeight uint64) [][]byte {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get precommit db iterator")
		os.Exit(1)
	}

	preCommitKeys := [][]byte{}
	iter.New(nil, nil)
	keyExist := false
	tmpAddress := new(common.Address)
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		if address.Equal(tmpAddress) == false && blockHeight != 0 {
			if bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue) &&
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
		fmt.Printf("Can not find key using given information")
		os.Exit(1)
	}
	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate")
		os.Exit(1)
	}

	return preCommitKeys
}
