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
		fmt.Println("==============print Database info==============")
		printDBInfo(qdb)
		fmt.Println("==============print governance variables==============")
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)), printGV)
		fmt.Println("==============print PRep Candidate info==============")
		queryPC(qdb, "")
	case DataTypeDI:
		printDBInfo(qdb)
	case DataTypeGV:
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)), printGV)
	case DataTypePC:
		queryPC(qdb, input.address)
	default:
		fmt.Printf("Invalid data type : %s\n", input.data)
		os.Exit(1)
	}
}

func queryPC(qdb db.Database, address string) {
	if address == "" {
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixPRepCandidate)), printPC)
	} else {
		addr := common.NewAddressFromString(address)
		runQueryPC(qdb, addr)
	}
}

func runQueryPC(qdb db.Database, address *common.Address) {
	bucket, err := qdb.GetBucket(db.PrefixPRepCandidate)
	if err != nil {
		fmt.Println("error while getting prep candidate bucket")
		os.Exit(1)
	}
	value, err := bucket.Get(address.Bytes())
	if err != nil {
		fmt.Println("error while Get value of prep candidate")
		os.Exit(1)
	}
	pcPrefixLen := len(db.PrefixPRepCandidate)
	qKey := make([]byte, pcPrefixLen+common.AddressBytes)
	copy(qKey, db.PrefixPRepCandidate)
	copy(qKey[pcPrefixLen:], address.Bytes())
	printPC(qKey, value)
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
	fmt.Println(dbInfo.String())
}

func printGV(key []byte, value []byte) {
	gv := new(core.GovernanceVariable)
	gv.SetBytes(value)
	gv.BlockHeight = common.BytesToUint64(key[len(db.PrefixGovernanceVariable):])

	fmt.Println(gv.String())
}

func printPC(key []byte, value []byte) {
	pc := getPC(key, value)
	fmt.Println(pc.String())
}

func getPC(key []byte, value []byte) *core.PRepCandidate {
	pc := new(core.PRepCandidate)
	pc.SetBytes(value)
	pc.Address = *common.NewAddress(key[len(db.PrefixPRepCandidate):])
	return pc
}
