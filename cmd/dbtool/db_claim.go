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

func queryClaimDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if input.address == "" {
		iteratePrintDB(DataTypeClaim, qdb)
		return
	}

	addr := common.NewAddressFromString(input.address)
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}
	value, err := bucket.Get(addr.Bytes())
	if value == nil || err != nil {
		return
	}

	printClaim(addr.Bytes(), value, addr)
}

func printClaim(key []byte, value []byte, address *common.Address) bool {
	claim, err := core.NewClaimFromBytes(value)
	if err != nil {
		log.Printf("Failed to make claim instance")
		return false
	}
	claim.Address = *common.NewAddress(key)

	//check argument
	if address != nil && claim.Address.Equal(address) == false {
		return false
	}

	fmt.Printf("%s\n", claim.String())

	return true
}
