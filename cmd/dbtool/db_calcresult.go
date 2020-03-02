package main

import (
	"errors"
	"fmt"
	cmdCommon "github.com/icon-project/rewardcalculator/cmd/common"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryCalcResultDB(input cmdCommon.Input) error {
	if input.Path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.Height == 0 {
		err := cmdCommon.PrintDB(input.Path, util.BytesPrefix([]byte(db.PrefixCalcResult)), printCalcResult)
		return err
	} else {
		dir, name := filepath.Split(input.Path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()

		if cr, err := getCalcResult(qdb, input.Height); err != nil {
			return err
		} else {
			printCalculationResult(cr)
		}
	}
	return nil
}

func getCalcResult(qdb db.Database, blockHeight uint64) (*core.CalculationResult, error) {
	bucket, err := qdb.GetBucket(db.PrefixCalcResult)
	if err != nil {
		fmt.Printf("Failed to get Bucket\n")
		return nil, err
	}

	key := common.Uint64ToBytes(blockHeight)
	value, err := bucket.Get(key)
	if err != nil {
		fmt.Println("Error while get calculateResult value")
		return nil, err
	}
	if value == nil {
		fmt.Println("Failed to get calculateResult value")
		return nil, nil
	}
	return newCalcResult(key, value)

}

func printCalcResult(key []byte, value []byte) error {
	if cr, err := newCalcResult(key, value); err != nil {
		return err
	} else {
		printCalculationResult(cr)
		return nil
	}
}

func printCalculationResult(cr *core.CalculationResult) {
	if cr != nil {
		fmt.Printf("%s\n", cr.String())
	}
}

func newCalcResult(key []byte, value []byte) (*core.CalculationResult, error) {
	if cr, err := core.NewCalculationResultFromBytes(value); err != nil {
		fmt.Println("Error while initialize calcResult")
		return nil, err
	} else {
		cr.BlockHeight = common.BytesToUint64(key)
		return cr, nil
	}
}
