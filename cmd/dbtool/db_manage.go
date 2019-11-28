package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
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
		iteratePrintDB(DataTypeGV, qdb)
		iteratePrintDB(DataTypePC, qdb)
	case DataTypeDI:
		printDBInfo(qdb)
	case DataTypeGV:
		iteratePrintDB(DataTypeGV, qdb)
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

func printGV(key []byte, value []byte, blockHeight uint64) bool{
	gv := new(core.GovernanceVariable)
	gv.SetBytes(value)
	gv.BlockHeight = common.BytesToUint64(key[len(db.PrefixGovernanceVariable):])

	if blockHeight != 0 && gv.BlockHeight != blockHeight {
		return false
	}
	fmt.Println("Governance variable set", gv.GVData, " at ", gv.BlockHeight)
	return true
}

func queryPC(qdb db.Database, address string) {
	if address == "" {
		iteratePrintDB(DataTypePC, qdb)
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixPRepCandidate)
	if err != nil {
		fmt.Println("error while getting prep candidate bucket")
		os.Exit(1)
	}
	addr := common.NewAddressFromString(address)
	value, err := bucket.Get(addr.Bytes())
	if err != nil {
		fmt.Println("error while Get value of prep candidate")
	}
	pcPrefixLen := len(db.PrefixPRepCandidate)
	qKey := make([]byte, pcPrefixLen+common.AddressBytes)
	copy(qKey, db.PrefixPRepCandidate)
	copy(qKey[pcPrefixLen:], addr.Bytes())
	printPC(qKey, value, address)
}

func printPC(key []byte, value []byte, address string) bool {
	pc := new(core.PRepCandidate)
	pc.SetBytes(value)
	pc.Address = *common.NewAddress(key[len(db.PrefixPRepCandidate):])

	if address != "" && address != pc.Address.String() {
		return false
	}
	fmt.Println("PRep Candidate : ", pc.String())
	return true
}
