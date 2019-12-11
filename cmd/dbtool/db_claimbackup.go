package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryClaimBackupDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}

	if input.address == "" {
		err = printDB(input.path, util.BytesPrefix([]byte(db.PrefixClaim)), printClaimBackup)
	} else {
		dir, name := filepath.Split(input.path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()
		//address := common.NewAddressFromString(input.address)
		//if claim, err := getClaimBackup(qdb, address); err != nil {
		//} else {
		//	printClaimBackupValue(claim)
		//}
	}
	return
}

func getClaimBackup(qdb db.Database, address *common.Address) (*core.Claim, error) {
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

func isManageKey(key []byte) bool {
	return len(key) == len(db.PrefixManagement) && bytes.Equal(key, []byte(db.PrefixManagement))
}

func printClaimBackupInfo(value []byte) error {
	var cbInfo core.ClaimBackupInfo
	err := cbInfo.SetBytes(value)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", cbInfo.String())
	return nil
}

func printClaimBackup(key []byte, value []byte) (err error) {
	if isManageKey(key) {
		printClaimBackupInfo(value)
		return nil
	}

	if claim, e := newClaimFromBackup(key, value); e != nil {
		return e
	} else {
		fmt.Printf("Key(%s), Value(%s)\n", core.ClaimBackupKeyString(key), claim.String())
		return nil
	}
}

func newClaimFromBackup(key []byte, value []byte) (*core.Claim, error) {
	if isManageKey(key) {
		return nil, nil
	}
	if claim, err := core.NewClaimFromBytes(value); err != nil {
		fmt.Println("Failed to make claim instance")
		return nil, err
	} else {
		claim.Address = *common.NewAddress(key[core.BlockHeightSize:])
		return claim, nil
	}
}
