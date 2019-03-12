package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

func createAddress(prefix []byte) (*common.Address, error) {
	data := make([]byte, common.AddressIDBytes - len(prefix))
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	buf := make([]byte, common.AddressIDBytes)
	copy(buf, prefix)
	copy(buf[len(prefix):], data)

	addr := common.NewAccountAddress(buf)
	//fmt.Printf("Created an address : %s", addr.String())

	return addr, nil
}

func createIScoreData(prefix []byte) *rewardcalculator.IScoreAccount {
	addr, err := createAddress(prefix)
	if err != nil {
		fmt.Printf("Failed to create Address err=%+v\n", err)
		return nil
	}

	ia := new(rewardcalculator.IScoreAccount)

	stake := rand.Uint64()
	delegate := stake / rewardcalculator.NumDelegate

	ia.Stake.SetUint64(stake)
	for i := 0; i < rewardcalculator.NumDelegate; i++ {
		var daddr *common.Address

		daddr = common.NewAccountAddress([]byte{byte(i+1)})
		ia.Delegations[i].Address = *daddr
		ia.Delegations[i].Delegate.SetUint64(delegate)
	}
	ia.Address = *addr

	//fmt.Printf("Result: %s", ia.String())

	return ia
}

func createData(bucket db.Bucket, prefix []byte, count int) int {
	// Governance Variable

	// PRep list

	// Account
	for i := 0; i < count; i++ {
		data := createIScoreData(prefix)
		if data == nil {
			return i
		}

		key := data.ID()
		value, _ := data.Bytes()
		//fmt.Printf("data: %s \n value: len: %d, bytes: %v\n", data.String(), len(value), value)

		bucket.Set(key, value)
	}


	return count
}

func createDB(dbDir string, dbName string, dbCount int, totalEntryCount int) {
	dbDir = fmt.Sprintf("%s/%s", dbDir, dbName)
	os.MkdirAll(dbDir, os.ModePerm)

	dbEntryCount := totalEntryCount / dbCount
	totalCount := 0

	var wait sync.WaitGroup
	wait.Add(dbCount)

	for i := 0; i < dbCount; i++ {
		go func(index int) {
			dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbCount)
			lvlDB := db.Open(dbDir, DBType, dbNameTemp)
			defer lvlDB.Close()
			defer wait.Done()

			bucket, _ := lvlDB.GetBucket(db.PrefixIScore)
			count := createData(bucket, []byte(strconv.FormatInt(int64(index), 16)), dbEntryCount)

			fmt.Printf("Create DB %s with %d entries.\n", dbNameTemp, count)
			totalCount += count
		} (i)
	}
	wait.Wait()

	fmt.Printf("Create %d DBs with total %d/%d entries.\n", dbCount, totalCount, totalEntryCount)
}

func (cli *CLI) create(dbName string, dbCount int, entryCount int) {
	fmt.Printf("Start create DB. name: %s, DB count: %d, Account count: %d\n", dbName, dbCount, entryCount)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)

	// Create I-Score DB
	createDB(DBDir, dbName, dbCount, entryCount)

	// Write I-Score DB Info. at global DB
	dbInfo := new(rewardcalculator.DBInfo)
	dbInfo.DbCount = dbCount
	dbInfo.AccountCount.Value = uint64(entryCount)

	data, _ := dbInfo.Bytes()
	//fmt.Printf("dbinfo: %s \n data: len: %d, bytes: %b\n", dbInfo.String(), len(data), data)
	bucket.Set(dbInfo.ID(), data)
}
