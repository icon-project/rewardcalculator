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

type AccountDB struct{}

func (accountDB AccountDB) query(dbPath string, rcRootPath string, address string) {
	if (dbPath != "" && address != "") || (dbPath == "" && address == "") {
		fmt.Println("address or dbPath required")
		os.Exit(1)
	}
	if dbPath != "" {
		dir, name := filepath.Split(dbPath)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()

		iteratePrintDB(DBTypeAccount, qdb, nil, 0, "")
	} else {
		// get db info
		if rcRootPath == "" {
			fmt.Printf("rcPath required when querying with address")
			os.Exit(1)
		}
		dbInfo := getDBInfo(rcRootPath)
		addr := common.NewAddressFromString(address)
		accountDBPath := getAccountDBPath(rcRootPath, addr, dbInfo)
		dir, name := filepath.Split(accountDBPath)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()

		bucket, err := qdb.GetBucket(db.PrefixIScore)
		if err != nil {
			fmt.Printf("Failed to get bucket")
			return
		}

		value, err := bucket.Get(addr.Bytes())
		if value == nil || err != nil {
			fmt.Printf("failed to get value in DB")
			return
		}

		printAccount(addr.Bytes(), value, addr)
	}
}

func printAccount(key []byte, value []byte, address *common.Address) bool {
	account, err := core.NewIScoreAccountFromBytes(value)
	if err != nil {
		log.Printf("Failed to make claim instance")
		fmt.Println("printAccount1")
		return false
	}
	account.Address = *common.NewAddress(key)

	if address != nil && account.Address.Equal(address) == false {
		return false
	}
	fmt.Printf("%s\n", account.String())

	return true
}

func getDBInfo(dbRoot string) *core.DBInfo {
	var dbInfo *core.DBInfo
	mdb := db.Open(dbRoot, string(db.GoLevelDBBackend), "")
	bucket, err := mdb.GetBucket(db.PrefixManagement)
	dbInfo = new(core.DBInfo)
	data, err := bucket.Get(dbInfo.ID())
	if data != nil {
		err = dbInfo.SetBytes(data)
		if err != nil {
			fmt.Println("Failed to set DB Information structure")
			os.Exit(1)
		}
	} else {
		fmt.Println("Invalid IScore DB")
		os.Exit(1)
	}
	return dbInfo
}

func getAccountDBPath(dbRoot string, address *common.Address, dbInfo *core.DBInfo) string {
	if address == nil || dbInfo == nil {
		fmt.Println("Can not get AccountDB without Address and DBInfo")
		os.Exit(1)
	}
	prefix := int(address.ID()[0]) % dbInfo.DBCount
	index := prefix + 1
	if dbInfo.QueryDBIsZero {
		accountDBPath := fmt.Sprintf(core.AccountDBNameFormat, index, dbInfo.DBCount, 0)
		dbRoot = filepath.Join(dbRoot, accountDBPath)
	} else {
		accountDBPath := fmt.Sprintf(core.AccountDBNameFormat, index, dbInfo.DBCount, 1)
		dbRoot = filepath.Join(dbRoot, accountDBPath)
	}
	return dbRoot
}
