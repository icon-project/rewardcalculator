package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
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
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixCalcResult)))
		printEntries(entries, printCalcResult)
	} else {
		runQueryCalcResult(qdb, input.height)
	}
}

func runQueryCalcResult(qdb db.Database, blockHeight uint64) {
	bucket, err := qdb.GetBucket(db.PrefixCalcResult)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	value, err := bucket.Get(common.Uint64ToBytes(input.height))
	if value == nil || err != nil {
		return
	}
	printCalcResult(common.Uint64ToBytes(input.height), value)

}

func printCalcResult(key []byte, value []byte) {
	var cr core.CalculationResult
	err := cr.SetBytes(value)
	if err != nil {
		log.Printf("Failed to make calculateResult instance")
		return
	}
	cr.BlockHeight = common.BytesToUint64(key)

	fmt.Printf("%s\n", cr.String())
}
