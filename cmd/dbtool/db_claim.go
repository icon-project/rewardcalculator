package main

import (
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryClaimDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.address == "" {
		err = printDB(input.path, util.BytesPrefix([]byte(db.PrefixClaim)), printClaim)
	} else {
		dir, name := filepath.Split(input.path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		address := common.NewAddressFromString(input.address)
		if claim, err := getClaim(qdb, address); err != nil {
		} else {
			printClaimInstance(claim)
		}
	}
	return
}

func getClaim(qdb db.Database, address *common.Address) (*core.Claim, error) {
	bucket, err := qdb.GetBucket(db.PrefixClaim)
	if err != nil {
		fmt.Println("Failed to get claim Bucket")
		return nil, err
	}
	key := address.Bytes()
	value, e := bucket.Get(key)
	if e != nil {
		fmt.Println("Error while get claim value")
		return nil, e
	}
	if value != nil {
		return newClaim(key, value)
	}
	return nil, nil
}

func printClaim(key []byte, value []byte) (err error) {
	if claim, e := newClaim(key, value); e != nil {
		return e
	} else {
		printClaimInstance(claim)
		return nil
	}
}

func printClaimInstance(claim *core.Claim) {
	if claim != nil {
		fmt.Printf("%s\n", claim.String())
	}
}

func newClaim(key []byte, value []byte) (*core.Claim, error) {
	if claim, err := core.NewClaimFromBytes(value); err != nil {
		fmt.Println("Failed to make claim instance")
		return nil, err
	} else {
		claim.Address = *common.NewAddress(key)
		return claim, nil
	}
}
