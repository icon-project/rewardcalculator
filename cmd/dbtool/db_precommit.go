package main

import (
	"bytes"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
	"os"
	"path/filepath"
)

func queryPreCommitDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if input.address == "" || input.height == 0 {
		iteratePrintDB(DBNamePreCommit, qdb)
		return
	}

	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get iterator")
		return
	}

	addr := common.NewAddressFromString(input.address)

	iter.New(nil, nil)
	preCommitKey := make([]byte, core.BlockHashSize+core.BlockHeightSize+common.AddressBytes)
	keyExist := false
	blockHeightBytesValue := common.Uint64ToBytes(input.height)
	for iter.Next() {
		key := iter.Key()
		if bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue) &&
			bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], addr.Bytes()) {
			keyExist = true
			copy(preCommitKey, key)
		}
	}
	iter.Release()

	if keyExist == false {
		fmt.Printf("Can not find key using given information")
		return
	}
	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate")
		return
	}

	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	value, err := bucket.Get(preCommitKey)
	if value == nil || err != nil {
		return
	}
	printPreCommit(preCommitKey, value, addr, input.height)
}

func printPreCommit(key []byte, value []byte, address *common.Address, blockHeight uint64) bool {
	var pc core.PreCommit

	err := pc.SetBytes(value)
	if err != nil {
		log.Printf("Failed to make preCommit instance")
		return false
	}
	pc.BlockHeight = common.BytesToUint64(key[:core.BlockHeightSize])
	pc.BlockHash = make([]byte, core.BlockHashSize)
	copy(pc.BlockHash, key[core.BlockHeightSize:core.BlockHeightSize+core.BlockHashSize])

	// check argument
	if address != nil && pc.Address.Equal(address) == false {
		return false
	}

	if blockHeight != 0 && pc.BlockHeight != blockHeight {
		return false
	}

	fmt.Printf("%s\n", pc.String())

	return true
}
