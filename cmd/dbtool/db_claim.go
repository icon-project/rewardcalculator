package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"os"
	"path/filepath"
)

func queryClaimDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if input.address == "" {
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixClaim)))
		printEntries(entries, printClaim)
	} else {
		address := common.NewAddressFromString(input.address)
		runQueryClaim(qdb, address)
	}
}

func runQueryClaim(qdb db.Database, address *common.Address){
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}
	value, err := bucket.Get(address.Bytes())
	if value == nil || err != nil {
		return
	}

	printClaim(address.Bytes(), value)
}

func printClaim(key []byte, value []byte) {
	claim, err := core.NewClaimFromBytes(value)
	if err != nil {
		log.Printf("Failed to make claim instance")
		return
	}
	claim.Address = *common.NewAddress(key)

	//check argument
	fmt.Printf("%s\n", claim.String())
	return
}
