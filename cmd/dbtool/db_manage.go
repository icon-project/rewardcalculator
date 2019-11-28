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

func queryManagementDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	switch input.data {
	case "":
		printDBInfo(qdb)
		printGV(qdb)
		printPC(qdb)
	case DataTypeDI:
		printDBInfo(qdb)
	case DataTypeGV:
		printGV(qdb)
	case DataTypePC:
		queryPC(qdb, input.address)
	default:
		fmt.Printf("Invalid data type : %s\n", input.data)
		os.Exit(1)
	}
}

func printDBInfo(qdb db.Database) {
	bucket, err := qdb.GetBucket(db.PrefixManagement)
	if err != nil {
		fmt.Println("error while getting database info bucket")
		os.Exit(1)
	}
	dbInfo := new(core.DBInfo)
	value, err := bucket.Get(dbInfo.ID())
	if err != nil {
		fmt.Println("error while Get value of Database info")
	}
	dbInfo.SetBytes(value)
	fmt.Println("Database info : ", dbInfo.String())
}

func printGV(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting GV iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixGovernanceVariable))
	iter.New(prefix.Start, prefix.Limit)
	gv := new(core.GovernanceVariable)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		gv.BlockHeight = common.BytesToUint64(key)
		gv.SetBytes(value)
		fmt.Println("Governance Variable : ", gv.String())
	}
	iter.Release()
}

func queryPC(qdb db.Database, address string) {
	if address == "" {
		printPC(qdb)
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixPRepCandidate)
	if err != nil {
		fmt.Println("error while getting prep candidate bucket")
		os.Exit(1)
	}
	pc := new(core.PRepCandidate)
	value, err := bucket.Get([]byte(address[2:]))
	if err != nil {
		fmt.Println("error while Get value of prep candidate")
	}
	pc.SetBytes(value)
	fmt.Println("precandidate : ", pc.String())

}

func printPC(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting GV iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixPRepCandidate))
	iter.New(prefix.Start, prefix.Limit)
	pc := new(core.PRepCandidate)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		pc.Address = *common.NewAddress(key[2:])
		pc.SetBytes(value)
		fmt.Println("prep candidate : ", pc.String())
	}
	iter.Release()
}
