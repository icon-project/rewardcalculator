package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryCalcDebugDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.address == "" && input.height == 0 {
		err = printDB(input.path, util.BytesPrefix([]byte(db.PrefixClaim)), printDebugOutput)
	} else {
		dir, name := filepath.Split(input.path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.address)
		err = queryCalcDebugOutput(qdb, address, input.height)
	}
	return
}

func queryCalcDebugOutput(qdb db.Database, address *common.Address, blockHeight uint64) error {
	qCalcDebugKeys, err := getDebugOutputKeys(qdb, blockHeight)
	if err != nil {
		return err
	}
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		fmt.Println("Failed to get debugResult Bucket")
		return err
	}

	nilAddress := new(common.Address)
	for _, key := range qCalcDebugKeys {
		value, err := bucket.Get(key)
		if err != nil {
			fmt.Println("Error while get debugResult")
			return err
		}
		if value == nil {
			continue
		}
		dr, err := newDebugOutput(key, value)
		if err != nil {
			return err
		} else {
			for _, calcResult := range dr.Results {
				if address.Equal(nilAddress) {
					printDebugOutputInstance(dr)
				} else if calcResult.Address.Equal(address) {
					printDebugOutputInstance(dr)
				}
			}
		}
	}
	return nil
}

func printDebugOutput(key []byte, value []byte) error {
	if pc, e := newDebugOutput(key, value); e != nil {
		return e
	} else {
		printDebugOutputInstance(pc)
		return nil
	}
}

func printDebugOutputInstance(dr *core.CalcDebugResult) {
	data, _ := json.MarshalIndent(dr.Results, "", "  ")
	fmt.Printf("blockHeight : %d\nblockHash : %s\n", dr.BlockHeight, dr.BlockHash)
	fmt.Printf("%s\n", string(data))
}

func newDebugOutput(key []byte, value []byte) (*core.CalcDebugResult, error) {
	dr := new(core.CalcDebugResult)

	err := dr.SetBytes(value)
	if err != nil {
		fmt.Printf("Failed to initialize debugResult instance\n")
		return nil, err

	}
	dr.BlockHeight = common.BytesToUint64(key[:core.BlockHeightSize])
	blockHash := make([]byte, core.BlockHashSize)
	copy(blockHash, key[core.BlockHeightSize:core.BlockHeightSize+core.BlockHashSize])
	dr.BlockHash = "0x" + hex.EncodeToString(blockHash)
	return dr, nil
}

func getDebugOutputKeys(qdb db.Database, blockHeight uint64) ([][]byte, error) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("Failed to get calcDebugResult db iterator")
		return nil, err
	}

	cDebugResultKeys := make([][]byte, 0)
	iter.New(nil, nil)
	keyExist := false
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		if bytes.Equal(key[core.BlockHeightSize-len(blockHeightBytesValue):core.BlockHeightSize], blockHeightBytesValue) {
			keyExist = true
			cDebugResultKeys = append(cDebugResultKeys, key)
		}
	}
	iter.Release()

	if keyExist == false {
		fmt.Println("Can not find key using given information")
		return nil, errors.New("calcDebugResult key does not exist")
	}
	err = iter.Error()
	if err != nil {
		fmt.Println("Error while iterate")
		return nil, err
	}

	return cDebugResultKeys, err
}
