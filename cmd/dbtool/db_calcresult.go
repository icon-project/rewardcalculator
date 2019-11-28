package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
	"os"
	"path/filepath"
)

func queryCalcResultDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if input.height == 0 {
		iteratePrintDB(DataTypeCalcResult, qdb)
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixCalcResult)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	value, err := bucket.Get(common.Uint64ToBytes(input.height))
	if value == nil || err != nil {
		return
	}
	printCalcResult(common.Uint64ToBytes(input.height), value, input.height)
}

func printCalcResult(key []byte, value []byte, blockHeight uint64) bool {
	var cr core.CalculationResult
	err := cr.SetBytes(value)
	if err != nil {
		log.Printf("Failed to make calculateResult instance")
		return false
	}
	cr.BlockHeight = common.BytesToUint64(key)

	if blockHeight != 0 && blockHeight != cr.BlockHeight {
		return false
	}

	fmt.Printf("%s\n", cr.String())

	return true
}
