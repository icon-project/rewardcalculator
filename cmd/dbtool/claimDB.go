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

type ClaimDB struct {
	dbPath string
}

func (claimDB ClaimDB) query(address string) {
	if claimDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(claimDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	if address == "" {
		iteratePrintDB(DBTypeClaim, qdb, nil, 0, "")
		return
	}

	addr := common.NewAddressFromString(address)
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
		fmt.Println("printAccount1")
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
