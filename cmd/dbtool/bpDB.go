package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"os"
	"path/filepath"
)

type BpDB struct {
	dbPath string
}

func (bpDB BpDB) query(blockHeight uint64) {
	if bpDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(bpDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if blockHeight == 0 {
		iteratePrintDB(DBTypeBPInfo, qdb, nil, 0, "")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixIISSBPInfo)
	if err != nil {
		fmt.Printf("Failed to get Bucket")
		return
	}

	bp := new(core.IISSBlockProduceInfo)
	bp.BlockHeight = blockHeight
	value, err := bucket.Get(bp.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixIISSBPInfo)
	copy(qKey[2:], bp.ID())

	printBP(qKey, value, blockHeight)
}

func printBP(key []byte, value []byte, blockHeight uint64) bool {
	if string(key[:BlockProduceInfoPrefixLen]) != "BP" {
		return false
	}
	bpInfo := new(core.IISSBlockProduceInfo)
	bpInfo.SetBytes(value)
	bpInfo.BlockHeight = common.BytesToUint64(key[BlockProduceInfoPrefixLen:])

	if blockHeight != 0 && bpInfo.BlockHeight != blockHeight {
		return false
	}
	fmt.Println(bpInfo.String())
	return true
}
