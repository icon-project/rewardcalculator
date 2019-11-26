package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"os"
	"path/filepath"
)

type PrepDB struct {
	dbPath string
}

func (prepDB PrepDB) query(blockHeight uint64) {
	if prepDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(prepDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if blockHeight == 0 {
		iteratePrintDB(DBTypePRep, qdb, nil, 0, "")
		return
	}

	bucket, err := qdb.GetBucket(db.PrefixIISSPRep)
	if err != nil {
		fmt.Printf("Failed to get Bucket")
		return
	}

	pRep := new(core.PRep)
	pRep.BlockHeight = blockHeight
	value, err := bucket.Get(pRep.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixIISSPRep)
	copy(qKey[2:], pRep.ID())

	printPRep(qKey, value, blockHeight)
}

func printPRep(key []byte, value []byte, blockHeight uint64) bool {
	if string(key[:PRepPrefixLen]) != "PR" {
		return false
	}
	prep := new(core.PRep)
	prep.SetBytes(value)
	prep.BlockHeight = common.BytesToUint64(key[PRepPrefixLen:])

	if blockHeight != 0 && prep.BlockHeight != blockHeight {
		return false
	}
	fmt.Println(prep.String())
	return true
}
