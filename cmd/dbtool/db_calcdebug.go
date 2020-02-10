package main

import (
	"errors"
	"fmt"
	cmdCommon "github.com/icon-project/rewardcalculator/cmd/common"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryCalcDebugDB(input cmdCommon.Input) (err error) {
	if input.Path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.Address == "" && input.Height == 0 {
		err = cmdCommon.PrintDB(input.Path, util.BytesPrefix([]byte(db.PrefixClaim)), cmdCommon.PrintCalcDebugResult)
	} else {
		dir, name := filepath.Split(input.Path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.Address)
		err = cmdCommon.QueryCalcDebugResult(qdb, address, input.Height)
	}
	return
}
