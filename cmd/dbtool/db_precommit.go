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

	if input.address == "" && input.height == 0 {
		iteratePrintDB(DataTypePreCommit, qdb)
		return
	}

	addr := common.NewAddressFromString(input.address)
	qPreCommitKeys := getKeys(qdb, *addr, input.height)

	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	for _, key := range qPreCommitKeys {
		value, err := bucket.Get(key)
		if value == nil || err != nil {
			continue
		}
		printPreCommit(key, value, addr, input.height)
	}
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

func getKeys(qdb db.Database, address common.Address, blockHeight uint64) [][]byte{
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get precommit db iterator")
		os.Exit(1)
	}

	precommitKeys := [][]byte{}
	iter.New(nil, nil)
	keyExist := false
	tmpAddress := new(common.Address)
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := iter.Key()
		if address.Equal(tmpAddress) == false && blockHeight != 0 {
			if bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue) &&
				bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], address.Bytes()) {
				keyExist = true
				precommitKeys = append(precommitKeys, key)
				break
			}
		}else{
			if address.Equal(tmpAddress) == false &&
				bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], address.Bytes()){
				keyExist = true
				precommitKeys = append(precommitKeys, key)
			} else if blockHeight != 0 && bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue){
				keyExist = true
				precommitKeys = append(precommitKeys, key)
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

	return precommitKeys
}
