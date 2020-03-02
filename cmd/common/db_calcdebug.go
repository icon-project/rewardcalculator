package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func QueryCalcDebugResult(qdb db.Database, address *common.Address, blockHeight uint64) error {
	qCalcDebugKeys, err := core.GetCalcDebugResultKeys(qdb, blockHeight)
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
		dr, err := core.NewCalcDebugResult(key, value)
		if err != nil {
			return err
		} else {
			for _, calcResult := range dr.Results {
				if address.Equal(nilAddress) {
					printCalcDebugResultInstance(dr)
				} else if calcResult.Address.Equal(address) {
					printCalcDebugResultInstance(dr)
				}
			}
		}
	}
	return nil
}

func PrintCalcDebugResult(key []byte, value []byte) error {
	if cb, e := core.NewCalcDebugResult(key, value); e != nil {
		return e
	} else {
		printCalcDebugResultInstance(cb)
		return nil
	}
}

func printCalcDebugResultInstance(dr *core.CalcDebugResult) {
	b, _ := json.MarshalIndent(dr, "", "  ")
	fmt.Printf("%s\n", string(b))
}

func QueryCalcDebugDB(input Input) (err error) {
	if input.Path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.Address == "" && input.Height == 0 {
		err = PrintDB(input.Path, util.BytesPrefix([]byte(db.PrefixClaim)), PrintCalcDebugResult)
	} else {
		dir, name := filepath.Split(input.Path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.Address)
		err = QueryCalcDebugResult(qdb, address, input.Height)
	}
	return
}
