package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
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

func createIScoreData(prefix []byte, pRepList []*core.PRepCandidate) *core.IScoreAccount {
	addr, err := createAddress(prefix)
	if err != nil {
		fmt.Printf("Failed to create Address err=%+v\n", err)
		return nil
	}

	ia := new(core.IScoreAccount)

	// set delegations
	for i := 0; i < core.NumDelegate; i++ {
		dg := new (core.DelegateData)
		dg.Address = pRepList[i].Address
		dg.Delegate.SetUint64(core.MinDelegation)
		ia.Delegations = append(ia.Delegations, dg)
	}
	ia.Address = *addr

	//fmt.Printf("Result: %s\n", ia.String())

	return ia
}

func createData(bucket db.Bucket, prefix []byte, count int, ctx *core.Context) int {
	pRepList := make([]*core.PRepCandidate, core.NumDelegate)
	i := 0
	for _, v := range ctx.PRepCandidates {
		pRepList[i] = v
		i++
		if i == core.NumDelegate {
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


func createAccountDB(dbDir string, dbCount int, entryCount int, ctx *core.Context) {
	dbEntryCount := entryCount / dbCount
	totalCount := 0

	var wait sync.WaitGroup
	wait.Add(dbCount)

	for i := 0; i < dbCount; i++ {
		aDBList := ctx.DB.GetCalcDBList()
		go func(index int) {
			bucket, _ := aDBList[index].GetBucket(db.PrefixIScore)

			count := createData(bucket, []byte(strconv.FormatInt(int64(index), 16)), dbEntryCount, ctx)

			fmt.Printf("Create DB %d with %d entries.\n", index, count)
			totalCount += count

			wait.Done()
		} (i)

	}
	wait.Wait()

	fmt.Printf("Create %d DBs with total %d/%d entries.\n", dbCount, totalCount, entryCount)
}

func (cli *CLI) create(dbName string, dbCount int, entryCount int) {
	fmt.Printf("Start create DB. name: %s, DB count: %d, Account count: %d\n", dbName, dbCount, entryCount)
	dbDir := filepath.Join(DBDir, dbName)
	os.MkdirAll(dbDir, os.ModePerm)

	lvlDB := db.Open(DBDir, DBType, dbName)

	// make governance variable
	gvList := make([]*core.GovernanceVariable, 0)
	gv := new(core.GovernanceVariable)
	gv.BlockHeight = 0
	gv.MainPRepCount.SetUint64(core.NumMainPRep)
	gv.SubPRepCount.SetUint64(core.NumSubPRep)
	gv.CalculatedIncentiveRep.SetUint64(1)
	gv.RewardRep.SetUint64(1)
	gvList = append(gvList, gv)

	// write to management DB
	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)
	for _, v := range gvList {
		value, _ := v.Bytes()
		bucket.Set(v.ID(), value)
		fmt.Printf("Write Governance variables: %+v, %s\n", v.ID(), v.String())
	}

	// make P-Rep candidate list
	pRepMap := make(map[common.Address]*core.PRepCandidate)
	for i := 0; i < 100; i++ {
		pRep := new(core.PRepCandidate)
		pRep.Address = *common.NewAccountAddress([]byte{byte(i+1)})
		pRep.Start = 0
		pRep.End = 0
		pRepMap[pRep.Address] = pRep
	}

	// write to management DB
	bucket, _ = lvlDB.GetBucket(db.PrefixPRepCandidate)
	for _, v := range pRepMap {
		value, _ := v.Bytes()
		bucket.Set(v.ID(), value)
		fmt.Printf("Write P-Rep candidate: %s\n", v.String())
	}

	// make P-Rep
	pRep := new(core.PRep)
	pRep.BlockHeight = 0
	pRep.TotalDelegation.SetUint64(100 * 100)
	pRep.List = make([]core.PRepDelegationInfo, len(pRepMap))
	for i := 0; i < len(pRep.List); i++ {
		var dInfo core.PRepDelegationInfo
		dInfo.Address = *common.NewAccountAddress([]byte{byte(i+1)})
		dInfo.DelegatedAmount.SetUint64(100)

		pRep.List[i] = dInfo
	}
	bucket, _ = lvlDB.GetBucket(db.PrefixPRep)
	value, _ := pRep.Bytes()
	bucket.Set(pRep.ID(), value)
	fmt.Printf("Write P-Rep : %s\n", pRep.String())

	lvlDB.Close()

	ctx, _ := core.NewContext(DBDir, DBType, dbName, dbCount, "")

	// create account DB
	createAccountDB(dbDir, dbCount, entryCount, ctx)
}
