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

const LastBlock = 1000000000

func createAddress(prefix []byte) (*common.Address, error) {
	data := make([]byte, common.AddressIDBytes - len(prefix))
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	buf := make([]byte, common.AddressIDBytes)
	copy(buf, prefix)
	copy(buf[len(prefix):], data)

	addr := common.NewAccountAddress(buf)
	//fmt.Printf("Created an address : %s\n", addr.String())

	return addr, nil
}

func createIScoreData(prefix []byte, pRepList []*rewardcalculator.PRepCandidate) *rewardcalculator.IScoreAccount {
	addr, err := createAddress(prefix)
	if err != nil {
		fmt.Printf("Failed to create Address err=%+v\n", err)
		return nil
	}

	ia := new(rewardcalculator.IScoreAccount)

	// set delegations
	for i := 0; i < rewardcalculator.NumDelegate; i++ {
		dg := new (rewardcalculator.DelegateData)
		dg.Address = pRepList[i].Address
		dg.Delegate.SetUint64(uint64(i))
		ia.Delegations = append(ia.Delegations, dg)
	}
	ia.Address = *addr

	//fmt.Printf("Result: %s\n", ia.String())

	return ia
}

func createData(bucket db.Bucket, prefix []byte, count int, opts *rewardcalculator.Context) int {
	pRepList := make([]*rewardcalculator.PRepCandidate, rewardcalculator.NumDelegate)
	i := 0
	for _, v := range opts.PRepCandidates {
		pRepList[i] = v
		i++
		if i == rewardcalculator.NumDelegate {
			break
		}
	}

	// Account
	for i := 0; i < count; i++ {
		data := createIScoreData(prefix, pRepList)
		if data == nil {
			return i
		}

		bucket.Set(data.ID(), data.Bytes())
	}

	return count
}


func createAccountDB(dbDir string, dbCount int, entryCount int, opts *rewardcalculator.Context) {
	dbEntryCount := entryCount / dbCount
	totalCount := 0

	var wait sync.WaitGroup
	wait.Add(dbCount)

	for i := 0; i < dbCount; i++ {
		dbNameTemp := fmt.Sprintf("calculate_%d_%d_0", i + 1, dbCount)
		go func(index int, dbName string) {
			lvlDB := db.Open(dbDir, DBType, dbName)
			defer lvlDB.Close()
			defer wait.Done()

			bucket, _ := lvlDB.GetBucket(db.PrefixIScore)
			count := createData(bucket, []byte(strconv.FormatInt(int64(index), 16)), dbEntryCount, opts)

			fmt.Printf("Create DB %s with %d entries.\n", dbName, count)
			totalCount += count
		} (i, dbNameTemp)

		dbNameTemp = fmt.Sprintf("calculate_%d_%d_1", i + 1, dbCount)
		lvlDB := db.Open(dbDir, DBType, dbNameTemp)
		lvlDB.Close()
	}
	wait.Wait()

	fmt.Printf("Create %d DBs with total %d/%d entries.\n", dbCount, totalCount, entryCount)
}

func (cli *CLI) create(dbName string, dbCount int, entryCount int) {
	fmt.Printf("Start create DB. name: %s, DB count: %d, Account count: %d\n", dbName, dbCount, entryCount)
	dbDir := fmt.Sprintf("%s/%s", DBDir, dbName)
	os.MkdirAll(dbDir, os.ModePerm)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	// Write DB Info. at global DB
	bucket, _ := lvlDB.GetBucket(db.PrefixManagement)
	dbInfo := new(rewardcalculator.DBInfo)
	dbInfo.DBCount = dbCount
	data, _ := dbInfo.Bytes()
	bucket.Set(dbInfo.ID(), data)

	// make global options
	ctx := new(rewardcalculator.Context)

	// make governance variable
	gvList := make([]*rewardcalculator.GovernanceVariable, 0)
	gv := new(rewardcalculator.GovernanceVariable)
	gv.BlockHeight = 0
	gv.IncentiveRep.Int.SetUint64(1)
	gv.IcxPrice.Int.SetUint64(100)
	gvList = append(gvList, gv)

	gv = new(rewardcalculator.GovernanceVariable)
	gv.BlockHeight = 500
	gv.IncentiveRep.Int.SetUint64(2)
	gv.IcxPrice.Int.SetUint64(200)
	gvList = append(gvList, gv)

	gv = new(rewardcalculator.GovernanceVariable)
	gv.BlockHeight = LastBlock
	gv.IncentiveRep.Int.SetUint64(3)
	gv.IcxPrice.Int.SetUint64(300)
	gvList = append(gvList, gv)

	ctx.GV = gvList

	// make P-Rep candidate list
	pRepMap := make(map[common.Address]*rewardcalculator.PRepCandidate)
	for i := 0; i < 16; i++ {
		pRep := new(rewardcalculator.PRepCandidate)
		pRep.Address = *common.NewAccountAddress([]byte{byte(i+1)})
		pRep.Start = 0
		pRep.End = 0
		pRepMap[pRep.Address] = pRep
	}

	ctx.PRepCandidates = pRepMap

	// write global options to DB
	bucket, _ = lvlDB.GetBucket(db.PrefixGovernanceVariable)
	for _, v := range ctx.GV {
		value, _ := v.Bytes()
		bucket.Set(v.ID(), value)
		fmt.Printf("Write Governance variables: %+v, %s\n", v.ID(), v.String())
	}

	bucket, _ = lvlDB.GetBucket(db.PrefixPrepCandidate)
	for _, v := range ctx.PRepCandidates {
		value, _ := v.Bytes()
		bucket.Set(v.ID(), value)
		fmt.Printf("Write P-Rep candidate: %s\n", v.String())
	}

	// create account DB
	createAccountDB(dbDir, dbCount, entryCount, ctx)
}
