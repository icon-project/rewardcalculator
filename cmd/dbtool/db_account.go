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
		result := queryAccountWithAccountDBPath(input.path, input.address)
		printIScoreAccountInstance(result)
		return
	} else if (input.rcDBRoot != "") && (input.accountType != "") {
		// get db info
		result := queryAccountWithRCRootPath(input.rcDBRoot, input.address, input.accountType)
		printIScoreAccountInstance(result)
		return
	}
	fmt.Println("DBPath or (rcDBPath and accountType) required")
	os.Exit(1)
}

func queryAccountWithAccountDBPath(path string, address string) *core.IScoreAccount{
	dir, name := filepath.Split(path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	if address == "" {
		accountEntries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIScore)))
		printEntries(accountEntries, printAccount)
		return nil
	} else {
		addr := common.NewAddressFromString(address)
		iScoreAccount := runQueryAccount(qdb, addr)
		return iScoreAccount
	}
}

func queryAccountWithRCRootPath(path string, address string, accountType string) *core.IScoreAccount{
	if accountType != "query" && accountType != "calculate" {
		fmt.Println("invalid accountType")
		os.Exit(1)
	}
	var result *core.IScoreAccount
	accountDBCount := getAccountDBCount(path)
	if address == "" {
		printAllAccountsInRcDB(path, accountDBCount, accountType)
	} else {
		account := runQueryAccountWithRCRoot(path, accountDBCount, address, accountType)
		result = account
	}
	return result
}

func runQueryAccount(qdb db.Database, address *common.Address) *core.IScoreAccount{
	bucket, err := qdb.GetBucket(db.PrefixIScore)
	if err != nil {
		fmt.Printf("Failed to get bucket")
		return nil
	}

	key := address.Bytes()
	value, err := bucket.Get(address.Bytes())
	if value == nil || err != nil {
		fmt.Printf("failed to get value in DB")
		return nil
	}
	iScoreAccount := getIScoreAccount(key, value)
	return iScoreAccount
}

func runQueryAccountWithRCRoot(rcRoot string, accountDBCount int, address string, accountType string) *core.IScoreAccount{
	addr := common.NewAddressFromString(address)
	prefix := int(addr.ID()[0]) % accountDBCount
	index := prefix + 1
	dbRoot0 := getNThAccountDBPathWithIndex(index, accountDBCount, rcRoot, 0)
	account0 := queryAccountWithAccountDBPath(dbRoot0, address)
	dbRoot1 := getNThAccountDBPathWithIndex(index, accountDBCount, rcRoot, 1)
	account1 := queryAccountWithAccountDBPath(dbRoot1, address)
	var calculateAccount *core.IScoreAccount
	var queryAccount *core.IScoreAccount
	var result *core.IScoreAccount
	if account0 != nil {
		if account0.BlockHeight >= account1.BlockHeight {
			calculateAccount = account0
			queryAccount = account1
		} else {
			calculateAccount = account1
			queryAccount = account0
		}
		if accountType == "query" {
			result = queryAccount
		} else {
			result = calculateAccount
		}
	}
	return result
}

func printAccount(key []byte, value []byte) {
	account := getIScoreAccount(key, value)
	printIScoreAccountInstance(account)
}

func printIScoreAccountInstance(account *core.IScoreAccount){
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

func getIScoreAccount(key []byte, value []byte) *core.IScoreAccount{
	account, err := core.NewIScoreAccountFromBytes(value)
	if err != nil {
		log.Printf("Failed to make account instance")
		return nil
	}
	account.Address = *common.NewAddress(key)
	return account
}

func getAllAccounts(dbPath string) []*core.IScoreAccount{
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	var accounts []*core.IScoreAccount
	entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIScore)))
	for _, v := range entries{
		account := getIScoreAccount(v.key, v.value)
		accounts = append(accounts, account)
	}
	return accounts
}

func printAllAccountsInRcDB(rcRoot string, accountDBCount int, accountType string){
	dbIndex := 0
	var dbPath string
	var accounts []*core.IScoreAccount
	found := false
	for i := 1; i <= accountDBCount; i++ {
		if !found {
			dbPath0 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 0)
			accounts0 := getAllAccounts(dbPath0)
			dbPath1 := getNThAccountDBPathWithIndex(i, accountDBCount, rcRoot, 1)
			accounts1 := getAllAccounts(dbPath1)
			dbPath = dbPath0
			if len(accounts0) > 0 {
				found = true
				if accounts0[0].BlockHeight >= accounts1[0].BlockHeight {
					if accountType == "query" {
						dbIndex = 1
						accounts = accounts1
						dbPath = dbPath1
					} else {
						dbIndex = 0
						accounts = accounts0
						dbPath = dbPath0
					}
				} else {
					if accountType == "query" {
						dbIndex = 0
						accounts = accounts0
						dbPath = dbPath0
					} else {
						dbIndex = 1
						accounts = accounts1
						dbPath = dbPath1
					}
				}
			}

		} else {
			dbPath = getNThAccountDBPathWithIndex(i, accountDBCount , rcRoot, dbIndex)
			accounts = getAllAccounts(dbPath)
		}
		fmt.Printf("=================Querying DB in %s=====================\n", dbPath)
		for _, v := range accounts{
			printIScoreAccountInstance(v)
		}
	}

}

func getNThAccountDBPathWithIndex(n int, accountDBCount int, rcRootPath string, index int) string {
	accountDBPath := fmt.Sprintf(core.AccountDBNameFormat, n, accountDBCount, index)
	accountDBRoot := filepath.Join(rcRootPath, accountDBPath)
	return accountDBRoot
}

