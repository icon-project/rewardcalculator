package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func queryAccountDB(input Input) {
	if input.path != "" {
		queryAccountWithAccountDBPath(input.path, input.address)
		return
	} else if input.rcDBRoot != "" {
		queryAccountWithRCRootPath(input.rcDBRoot, input.address, input.accountType)
		return
	}
	fmt.Println("DBPath or rcDBPath required")
	os.Exit(1)
}

func queryAccountWithAccountDBPath(path string, address string) {
	dir, name := filepath.Split(path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	if address == "" {
		iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixIScore)), printAccount)
	} else {
		addr := common.NewAddressFromString(address)
		runQueryAccount(qdb, addr)
	}
}

func queryAccountWithRCRootPath(rcRoot string, address string, accountType string) {
	accountDBCount := getAccountDBCount(rcRoot)
	if (address == "") && (accountType == "") {
		for i := 1; i <= accountDBCount; i++ {
			path0 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 0)
			printAllEntriesInPath(path0, util.BytesPrefix([]byte(db.PrefixIScore)), printAccount)
			path1 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 1)
			printAllEntriesInPath(path1, util.BytesPrefix([]byte(db.PrefixIScore)), printAccount)
		}
	} else if (address == "") && (accountType != "") {
		printAllAccountsInRcDBWithSpecifiedType(rcRoot, accountDBCount, accountType)
	} else if (address != "") && (accountType == "") {
		fmt.Printf("=================Querying query data=================")
		runQueryAccountWithRCRoot(rcRoot, accountDBCount, address, "query")
		fmt.Printf("=====================Querying calculate data=====================")
		runQueryAccountWithRCRoot(rcRoot, accountDBCount, address, "calculate")
	} else {
		runQueryAccountWithRCRoot(rcRoot, accountDBCount, address, accountType)
	}
}

func runQueryAccount(qdb db.Database, address *common.Address) {
	iScoreAccount := getAccount(qdb, address)
	printIScoreAccountInstance(iScoreAccount)
}

func getAccount(qdb db.Database, address *common.Address) *core.IScoreAccount {
	bucket, err := qdb.GetBucket(db.PrefixIScore)
	if err != nil {
		fmt.Printf("Failed to get bucket")
		os.Exit(1)
	}

	key := address.Bytes()
	value, err := bucket.Get(address.Bytes())
	if err != nil {
		fmt.Printf("Error while get account value")
		os.Exit(1)
	}
	if value == nil {
		fmt.Printf("Failed to get account value")
		return nil
	}
	return getIScoreAccount(key, value)
}

func getAccountInSpecifiedPath(dbRoot string, address string) *core.IScoreAccount {
	dir, name := filepath.Split(dbRoot)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	addr := common.NewAddressFromString(address)
	account := getAccount(qdb, addr)
	return account
}

func runQueryAccountWithRCRoot(rcRoot string, accountDBCount int, address string, accountType string) {
	if accountType != "query" && accountType != "calculate" {
		fmt.Println("Invalid accountType")
		os.Exit(1)
	}
	addr := common.NewAddressFromString(address)
	prefix := int(addr.ID()[0]) % accountDBCount
	index := prefix + 1
	dbRoot0 := getNThAccountDBPathWithIndex(index, accountDBCount, rcRoot, 0)
	account0 := getAccountInSpecifiedPath(dbRoot0, address)
	dbRoot1 := getNThAccountDBPathWithIndex(index, accountDBCount, rcRoot, 1)
	account1 := getAccountInSpecifiedPath(dbRoot1, address)
	var calculateAccount *core.IScoreAccount
	var queryAccount *core.IScoreAccount
	if account0 != nil {
		if account0.BlockHeight >= account1.BlockHeight {
			calculateAccount = account0
			queryAccount = account1
		} else {
			calculateAccount = account1
			queryAccount = account0
		}
		if accountType == "query" {
			printIScoreAccountInstance(queryAccount)
		} else {
			printIScoreAccountInstance(calculateAccount)
		}
	}
}

func printAccount(key []byte, value []byte) {
	account := getIScoreAccount(key, value)
	printIScoreAccountInstance(account)
}

func printIScoreAccountInstance(account *core.IScoreAccount) {
	if account != nil {
		fmt.Printf("%s\n", account.String())
	}
}

func getAccountDBCount(path string) int {
	contents, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("Failed to read directory")
		os.Exit(1)
	}
	count := 0
	for _, f := range contents {
		if strings.HasPrefix(f.Name(), "calculate_") {
			count++
		}
	}
	result := count / 2
	return result
}

func getIScoreAccount(key []byte, value []byte) *core.IScoreAccount {
	account, err := core.NewIScoreAccountFromBytes(value)
	if err != nil {
		log.Printf("Failed to make account instance")
		return nil
	}
	account.Address = *common.NewAddress(key)
	return account
}

func getFirstAccount(dbPath string) *core.IScoreAccount {
	var account *core.IScoreAccount
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get iterator")
		os.Exit(1)
	}

	prefix := util.BytesPrefix([]byte(db.PrefixIScore))
	iter.New(prefix.Start, prefix.Limit)
	var key []byte
	var value []byte
	for iter.Next() {
		key = iter.Key()
		value = iter.Value()
		break
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate. %+v", err)
		os.Exit(1)
	}
	if key != nil {
		account = getIScoreAccount(key, value)
	}
	return account
}

func printAllAccountsInRcDBWithSpecifiedType(rcRoot string, accountDBCount int, accountType string) {
	dbIndex := 0
	var dbPath string
	found := false
	for i := 1; i <= accountDBCount; i++ {
		if !found {
			dbPath0 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 0)
			account0 := getFirstAccount(dbPath0)
			dbPath1 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 1)
			account1 := getFirstAccount(dbPath1)
			dbPath = dbPath0
			fmt.Printf("=================Querying DB in %s=================\n", dbPath)
			if account0 != nil {
				found = true
				if account0.BlockHeight >= account1.BlockHeight {
					if accountType == "query" {
						dbIndex = 1
						dbPath = dbPath1
					} else {
						dbIndex = 0
						dbPath = dbPath0
					}
				} else {
					if accountType == "query" {
						dbIndex = 0
						dbPath = dbPath0
					} else {
						dbIndex = 1
						dbPath = dbPath1
					}
				}
				queryAccountWithAccountDBPath(dbPath, "")
			}

		} else {
			dbPath = getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, dbIndex)
			fmt.Printf("=====================Querying DB in %s=================\n", dbPath)
			queryAccountWithAccountDBPath(dbPath, "")
		}
	}

}

func getNThAccountDBPathWithIndex(n int, accountDBCount int, rcRootPath string, index int) string {
	accountDBPath := fmt.Sprintf(core.AccountDBNameFormat, n, accountDBCount, index)
	accountDBRoot := filepath.Join(rcRootPath, accountDBPath)
	return accountDBRoot
}
