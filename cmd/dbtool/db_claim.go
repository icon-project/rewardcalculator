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

func queryClaimDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if input.address == "" {
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixClaim)), printClaim)
	} else {
		address := common.NewAddressFromString(input.address)
		runQueryClaim(qdb, address)
	}
}

func runQueryClaim(qdb db.Database, address *common.Address) {
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		fmt.Printf("Failed to get claim Bucket")
		os.Exit(1)
	}
	value, err := bucket.Get(address.Bytes())
	if err != nil {
		fmt.Printf("Error while get claim value")
		os.Exit(1)
	}
	printClaim(address.Bytes(), value)
}

func printClaim(key []byte, value []byte) {
	claim := getClaim(key, value)
	fmt.Printf("%s\n", claim.String())
}

func getClaim(key []byte, value []byte) *core.Claim {
	claim, err := core.NewClaimFromBytes(value)
	if err != nil {
		fmt.Printf("Failed to make claim instance")
		os.Exit(1)
	}
	claim.Address = *common.NewAddress(key)

	fmt.Printf("%s\n", claim.String())
	return claim
}
