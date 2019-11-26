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

type GvDB struct {
	dbPath string
}

func (gvDB GvDB) query(blockHeight uint64) {
	if gvDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(gvDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if blockHeight == 0 {
		iteratePrintDB(DBTypeGV, qdb, nil, 0, "")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixGovernanceVariable)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	gv := new(core.IISSGovernanceVariable)
	gv.BlockHeight = blockHeight
	value, err := bucket.Get(gv.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixGovernanceVariable)
	copy(qKey[2:], gv.ID())

	printGv(qKey, value, blockHeight)
}

func printGv(key []byte, value []byte, blockHeight uint64) bool {
	if string(key[:GVPrefixLen]) != "GV" {
		return false
	}
	gv := new(core.GovernanceVariable)
	gv.SetBytes(value)
	gv.BlockHeight = common.BytesToUint64(key[GVPrefixLen:])

	if blockHeight != 0 && gv.BlockHeight != blockHeight {
		return false
	}
	fmt.Println("Governance variable set", gv.GVData, " at ", gv.BlockHeight)
	return true
}
