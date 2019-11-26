package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type TxDB struct {
	dbPath string
}

func (txDB TxDB) query(index string) {
	if txDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(txDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if index == "" {
		iteratePrintDB(DBTypeTX, qdb, nil, 0, "")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixIISSTX)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}
	idx, err := strconv.ParseUint(index, 10, 64)
	if err != nil {
		fmt.Println("invalid index. index must be an unsigned integer")
		return
	}

	tx := new(core.IISSTX)
	tx.Index = idx
	value, err := bucket.Get(tx.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixIISSTX)
	copy(qKey[2:], tx.ID())

	printTransaction(qKey, value, index)
}

func printTransaction(key []byte, value []byte, index string) bool {
	if string(key[:TransactionPrefixLen]) != "TX" {
		return false
	}
	tx := new(core.IISSTX)
	tx.SetBytes(value)
	tx.Index = common.BytesToUint64(key[TransactionPrefixLen:])
	uIndex, err := strconv.ParseUint(index, 10, 64)
	if index == "" {
		uIndex = 0
		err = nil
	}
	if err != nil {
		fmt.Println("Invalid index. index must be unsigned integer")
		os.Exit(1)
	}

	if index != "" && tx.Index != uIndex {
		return false
	}
	fmt.Println(tx.String())
	return true
}
