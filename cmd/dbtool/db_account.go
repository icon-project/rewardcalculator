package main

import (
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func queryAccountDB(input Input) (err error) {
	if input.path != "" {
		err = queryAccountDBWithPath(input.path, input.address)
		return
	} else if input.rcDBRoot != "" {
		err = queryAccountDBWithRCRoot(input.rcDBRoot, input.address, input.accountType)
		return
	}
	return errors.New("invalid db path")
}

func queryAccountDBWithPath(path string, address string) error {
	if address == "" {
		err := printDB(path, util.BytesPrefix([]byte(db.PrefixIScore)), printAccount)
		return err
	} else {
		dir, name := filepath.Split(path)
		qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
		defer qdb.Close()

		if account, err := getIScoreAccount(qdb, address); err != nil {
			return err
		} else {
			if account != nil {
				printIScoreAccount(account)
			} else {
				fmt.Printf("There is no %s in %s", address, path)
			}
		}
	}
	return nil
}

func queryAccountDBWithRCRoot(rcRoot string, address string, accountDBType string) error {
	accountDBCount, queryDBSuffix, err := getAccountDBInfo(rcRoot)

	index := -1
	if address != "" {
		addr := common.NewAddressFromString(address)
		index = int(addr.ID()[0])%accountDBCount + 1
	}

	dbType := -1
	if accountDBType == AccountDBTypeCalculate {
		dbType = 1 - queryDBSuffix
	} else if accountDBType == AccountDBTypeQuery {
		dbType = queryDBSuffix
	}

	// get DB path
	pathSlice := make([]string, 0)
	if index != -1 {
		if dbType == -1 { // no DB Type. query first
			pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, index, accountDBCount, queryDBSuffix))
			pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, index, accountDBCount, 1-queryDBSuffix))
		} else {
			pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, index, accountDBCount, dbType))
		}
	} else {
		if dbType == -1 { // no DB Type. query first
			for i := 1; i <= accountDBCount; i++ {
				pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, i, accountDBCount, queryDBSuffix))
			}
			for i := 1; i <= accountDBCount; i++ {
				pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, i, accountDBCount, 1-queryDBSuffix))
			}
		} else {
			for i := 1; i <= accountDBCount; i++ {
				pathSlice = append(pathSlice, getAccountDBPathWithIndex(rcRoot, i, accountDBCount, dbType))
			}
		}
	}

	// do query
	for _, path := range pathSlice {
		if err = queryAccountDBWithPath(path, address); err != nil {
			return err
		}
	}

	return nil
}

func getIScoreAccount(qdb db.Database, address string) (*core.IScoreAccount, error) {
	addr := common.NewAddressFromString(address)

	bucket, err := qdb.GetBucket(db.PrefixIScore)
	if err != nil {
		fmt.Printf("Failed to get bucket")
		return nil, err
	}

	key := addr.Bytes()
	value, e := bucket.Get(addr.Bytes())
	if e != nil {
		fmt.Printf("Error while get account from DB")
		return nil, e
	}
	if value == nil {
		return nil, nil
	}
	return newIScoreAccount(key, value)
}

func printIScoreAccount(account *core.IScoreAccount) {
	if account != nil {
		fmt.Printf("%s\n", account.String())
	}
}

func printAccount(key []byte, value []byte) error {
	if account, err := newIScoreAccount(key, value); err != nil {
		return err
	} else {
		printIScoreAccount(account)
		return nil
	}
}

func newIScoreAccount(key []byte, value []byte) (account *core.IScoreAccount, err error) {
	account, err = core.NewIScoreAccountFromBytes(value)
	if err != nil {
		fmt.Printf("Failed to make IScore account")
		return nil, err
	}
	account.Address = *common.NewAddress(key)
	return
}

func getFirstAccount(dbPath string) (*core.IScoreAccount, error) {
	var account *core.IScoreAccount
	dir, name := filepath.Split(dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get iterator")
		return nil, err
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
		return nil, err
	}
	if key != nil {
		account, _ = newIScoreAccount(key, value)
	}
	return account, nil
}

func getAccountDBPathWithIndex(rcRootPath string, index int, dbCount int, suffix int) string {
	name := fmt.Sprintf(core.AccountDBNameFormat, index, dbCount, suffix)
	return filepath.Join(rcRootPath, name)
}

func getAccountDBInfo(rcRoot string) (accountDBCount int, queryDBSuffix int, err error) {
	if accountDBCount, err = getAccountDBCount(rcRoot); err != nil {
		return 0, 0, err
	}
	var account0, account1 *core.IScoreAccount

	for i := 1; i <= accountDBCount; i++ {
		dbPath0 := getAccountDBPathWithIndex(rcRoot, i, accountDBCount, 0)
		account0, err = getFirstAccount(dbPath0)
		dbPath1 := getAccountDBPathWithIndex(rcRoot, i, accountDBCount, 1)
		account1, err = getFirstAccount(dbPath1)
		if account0 != nil {
			break
		}
	}

	if account0 == nil && account1 == nil {
		queryDBSuffix = 0
		return
	} else if account0 == nil {
		queryDBSuffix = 1
		return
	} else if account1 == nil {
		queryDBSuffix = 0
		return
	}

	if account0.BlockHeight >= account1.BlockHeight {
		queryDBSuffix = 1
	} else {
		queryDBSuffix = 0
	}
	return
}

func getAccountDBCount(path string) (int, error) {
	contents, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println("Failed to read directory")
		return 0, err
	}
	count := 0
	for _, f := range contents {
		if strings.HasPrefix(f.Name(), "calculate_") {
			count++
		}
	}
	result := count / 2
	return result, nil
}
