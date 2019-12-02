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
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)))
		printEntries(entries, printGV)
		entries = getEntries(qdb, util.BytesPrefix([]byte(db.PrefixPRepCandidate)))
		printEntries(entries, printPC)
	case DataTypeDI:
		printDBInfo(qdb)
	case DataTypeGV:
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)))
		printEntries(entries, printGV)
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

func printGV(key []byte, value []byte) {
	gv := new(core.GovernanceVariable)
	gv.SetBytes(value)
	gv.BlockHeight = common.BytesToUint64(key[len(db.PrefixGovernanceVariable):])

	fmt.Println("Governance variable set", gv.GVData, " at ", gv.BlockHeight)
}

func queryPC(qdb db.Database, address string) {
	if address == "" {
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixPRepCandidate)))
		printEntries(entries, printPC)
	} else {
		addr := common.NewAddressFromString(address)
		runQueryPC(qdb, addr)
	}
}

func printPC(key []byte, value []byte) {
	pc := new(core.PRepCandidate)
	pc.SetBytes(value)
	pc.Address = *common.NewAddress(key[len(db.PrefixPRepCandidate):])

	fmt.Println("PRep Candidate : ", pc.String())
}

func runQueryPC(qdb db.Database, address *common.Address){
	bucket, err := qdb.GetBucket(db.PrefixPRepCandidate)
	if err != nil {
		fmt.Println("error while getting prep candidate bucket")
		os.Exit(1)
	}
	value, err := bucket.Get(address.Bytes())
	if err != nil {
		fmt.Println("error while Get value of prep candidate")
	}
	pcPrefixLen := len(db.PrefixPRepCandidate)
	qKey := make([]byte, pcPrefixLen+common.AddressBytes)
	copy(qKey, db.PrefixPRepCandidate)
	copy(qKey[pcPrefixLen:], address.Bytes())
	printPC(qKey, value)
}
