package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
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
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixCalcResult)), printCalcResult)
	} else {
		runQueryCalcResult(qdb, input.height)
	}
}

func runQueryCalcResult(qdb db.Database, blockHeight uint64) {
	bucket, err := qdb.GetBucket(db.PrefixCalcResult)
	if err != nil {
		fmt.Printf("Failed to get Bucket")
		os.Exit(1)
	}

	value, err := bucket.Get(common.Uint64ToBytes(blockHeight))
	if err != nil {
		fmt.Println("Error while get calculateResult value")
		os.Exit(1)
	}
	if value == nil {
		fmt.Println("Failed to get calculateResult value")
	}
	printCalcResult(common.Uint64ToBytes(blockHeight), value)
}

func printCalcResult(key []byte, value []byte) {
	cr := getCalcResult(key, value)
	fmt.Printf("%s\n", cr.String())
}

func getCalcResult(key []byte, value []byte) *core.CalculationResult {
	cr, err := core.NewCalculationResultFromBytes(value)
	if err != nil {
		fmt.Println("Error while initialize calcResult")
		os.Exit(1)
	}
	cr.BlockHeight = common.BytesToUint64(key)
	return cr
}
