package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
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
	} else if !(input.rcDBRoot == "" && input.accountType == "") {
		// get db info
		queryAccountWithRCRootPath(input.rcDBRoot, input.address, input.accountType)
		return
	}
	fmt.Println("DBPath or (rcDBPath and accountType) required")
	os.Exit(1)
}

func printAccount(key []byte, value []byte, address *common.Address) bool {
	account, err := core.NewIScoreAccountFromBytes(value)
	if err != nil {
		log.Printf("Failed to make account instance")
		return false
	}
	account.Address = *common.NewAddress(key)

	if address != nil && account.Address.Equal(address) == false {

		return false
	}
	fmt.Printf("%s\n", account.String())

	return true
}

func queryAccountWithAccountDBPath(path string, address string) {
	dir, name := filepath.Split(path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	if address == "" {
		iteratePrintDB(DataTypeAccount, qdb)
		return
	}
	addr := common.NewAddressFromString(address)
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

func queryAccountWithRCRootPath(path string, address string, accountType string) {
	if accountType != "query" && accountType != "calculate" {
		fmt.Println("invalid accountType")
		os.Exit(1)
	}
	accountDBNum := getAccountDBCount(path)
	if address != "" {
		addr := common.NewAddressFromString(address)
		prefix := int(addr.ID()[0]) % accountDBNum
		index := prefix + 1
		accountDBPath0 := fmt.Sprintf(core.AccountDBNameFormat, index, accountDBNum, 0)
		dbRoot0 := filepath.Join(path, accountDBPath0)
		account0 := getAccount(dbRoot0, addr)
		accountDBPath1 := fmt.Sprintf(core.AccountDBNameFormat, index, accountDBNum, 1)
		dbRoot1 := filepath.Join(path, accountDBPath1)
		account1 := getAccount(dbRoot1, addr)
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
				fmt.Println("account : ", queryAccount.String())
			} else {
				fmt.Println("account : ", calculateAccount.String())
			}
		}
		return
	}

	dbIndex := 0
	var accountDBPath string
	found := false
	for i := 1; i <= accountDBNum; i++ {
		if !found {
			accountDBPath0 := fmt.Sprintf(core.AccountDBNameFormat, i, accountDBNum, 0)
			dbPath0 := filepath.Join(path, accountDBPath0)
			accounts0 := getAccounts(dbPath0)
			accountDBPath1 := fmt.Sprintf(core.AccountDBNameFormat, i, accountDBNum, 1)
			dbPath1 := filepath.Join(path, accountDBPath1)
			accounts1 := getAccounts(dbPath1)
			if len(accounts0) > 0 {
				found = true
				if accounts0[0].BlockHeight >= accounts1[0].BlockHeight {
					if accountType == "query" {
						dbIndex = 1
						accountDBPath = accountDBPath1
					} else {
						dbIndex = 0
						accountDBPath = accountDBPath0
					}
				} else {
					if accountType == "query" {
						dbIndex = 0
						accountDBPath = accountDBPath0
					} else {
						dbIndex = 1
						accountDBPath = accountDBPath1
					}
				}
			}

		}
		accountDBPath = fmt.Sprintf(core.AccountDBNameFormat, i, accountDBNum, dbIndex)
		dbPath := filepath.Join(path, accountDBPath)
		fmt.Println(dbPath)
		printAccountsInDB(dbPath)
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

func getAccount(dbPath string, address *common.Address) *core.IScoreAccount {
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	bucket, err := qdb.GetBucket(db.PrefixIScore)
	if err != nil {
		fmt.Printf("Failed to get account bucket")
		os.Exit(1)
	}

	value, err := bucket.Get(address.Bytes())
	if err != nil {
		fmt.Printf("error while get value in DB")
		os.Exit(1)
	}
	if value == nil {
		fmt.Printf("failed to get value in DB")
		return nil
	}
	account, err := core.NewIScoreAccountFromBytes(value)
	account.Address = *address
	return account
}

func getAccounts(dbPath string) []*core.IScoreAccount {
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	accounts := []*core.IScoreAccount{}
	iter, err := qdb.GetIterator()
	if err != nil {

	}
	prefix := getPrefix(DataTypeAccount)
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		account, err := core.NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			fmt.Println("Error while init IscoreAccount")
			os.Exit(1)
		}
		accounts = append(accounts, account)
	}
	iter.Release()
	return accounts
}

func printAccountsInDB(dbPath string) {
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	iteratePrintDB(DataTypeAccount, qdb)
}
