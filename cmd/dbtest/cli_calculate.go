package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"math/big"
	"sync"
)

func calculateIScore(ia *rewardcalculator.IScoreData, opts *rewardcalculator.GlobalOptions) bool {
	// IScore = old + period * G.V * sum(valid delegations)
	if opts.BlockHeight.Value == 0 {
		opts.BlockHeight.Value = ia.BlockHeight.Value + 1
	}
	period := opts.BlockHeight.Value - ia.BlockHeight.Value
	gv := opts.GV.RewardRep.Value
	if period == 0 || gv == 0 {
		return false
	}

	multiplier := big.NewInt(int64(period * gv))

	var delegations common.HexInt
	for i := 0; i < rewardcalculator.NumDelegate; i++ {
		for j := 0; j < rewardcalculator.NumPRep; j++ {
			if ia.Delegations[i].Address.Equal(&opts.Validators[j]) {
				delegations.Add(&delegations.Int, &ia.Delegations[i].Delegate.Int)
				continue
			}
		}
	}
	if delegations.Int.Sign() == 0 {
		// there is no delegations
		return false
	}
	delegations.Int.Mul(&delegations.Int, multiplier)

	//fmt.Printf("period: %U, gv: %U, multiplier: %s, delegations: %s",
	//	period, gv, multiplier.String(), delegations.Int.String())

	// increase value
	ia.IScore.Add(&ia.IScore.Int, &delegations.Int)

	// BlockHeight
	ia.BlockHeight.Value = opts.BlockHeight.Value

	return true
}

func makePrefix(id db.BucketID, value uint8, last bool) []byte {
	buf := make([]byte, len(id) + 1)
	copy(buf, id)
	if last {
		buf[len(id)-1]++
	} else {
		buf[len(id)] = value
	}

	return buf
}

func getPrefix(id db.BucketID, index int, worker int) ([]byte, []byte) {
	if worker == 1 {
		return nil, nil
	}

	unit := uint8(256 / worker)
	start := makePrefix(id, unit * uint8(index), false)
	limit := makePrefix(id, unit * uint8(index + 1), index == worker - 1)

	return start, limit
}

func calculateDB(dbi db.Database, bucket db.Bucket, start []byte, limit []byte,
	opts *rewardcalculator.GlobalOptions, batchCount uint64) (count uint64, entries uint64) {
	iter, _ := dbi.GetIterator()
	batch, _ := dbi.GetBatch()
	entries = 0; count = 0

	batch.New()
	iter.New(start, limit)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		ia, err := rewardcalculator.NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			fmt.Printf("Can't read data with iterator\n")
			return 0, 0
		}

		//fmt.Printf("Read data: %s\n", ia.String())

		// calculate
		if calculateIScore(&ia.IScoreData, opts) == false {
			continue
		}

		//fmt.Printf("Updated data: %s\n", ia.String())

		value, _ := ia.Bytes()

		if batchCount > 0 {
			batch.Set(iter.Key(), value)

			// write batch to DB
			if entries != 0 && (entries % batchCount) == 0 {
				err = batch.Write()
				if err != nil {
					fmt.Printf("Failed to write batch\n")
				}
				batch.Reset()
			}
		} else {
			bucket.Set(key, value)
		}

		count++
	}

	// write batch to DB
	if batchCount > 0 {
		err := batch.Write()
		if err != nil {
			fmt.Printf("Failed to write batch\n")
		}
		batch.Reset()
	}

	// finalize iterator
	iter.Release()
	err := iter.Error()
	if err != nil {
		fmt.Printf("There is error while iteration. %+v", err)
	}

	//fmt.Printf("Calculate %d entries for prefix %v-%v %d entries\n", count, start, limit, entries)

	return count, entries
}

func (cli *CLI) calculate(dbName string, blockHeight uint64, batchCount uint64) {
	fmt.Printf("Start calculate DB. name: %s, block height: %d, batch count: %d\n", dbName, blockHeight, batchCount)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)

	dbInfo := new(rewardcalculator.DBInfo)

	data, _ := bucket.Get(dbInfo.ID())
	err := dbInfo.SetBytes(data)
	if err != nil {
		fmt.Printf("Failed to read DB Info. err=%+v\n", err)
		return
	}

	fmt.Printf("DBInfo %s\n", dbInfo.String())

	// FIXME read from Global DB
	// make global options
	opts := new(rewardcalculator.GlobalOptions)

	opts.BlockHeight.Value = dbInfo.BlockHeight.Value
	for i := 0 ; i < rewardcalculator.NumDelegate; i++ {
		daddr := common.NewAccountAddress([]byte{byte(i+1)})
		opts.Validators[i] = *daddr
	}
	opts.GV.RewardRep.Value = 1

	var totalCount, totalEntry uint64
	var wait sync.WaitGroup
	wait.Add(dbInfo.DbCount)

	dbDir := fmt.Sprintf("%s/%s", DBDir, dbName)
	for i:= 0; i< dbInfo.DbCount; i++ {
		go func(index int) {
			dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbInfo.DbCount)
			acctDB := db.Open(dbDir, DBType, dbNameTemp)
			defer acctDB.Close()
			defer wait.Done()

			bucket, _ := acctDB.GetBucket(db.PrefixIScore)
			c, e := calculateDB(acctDB, bucket, nil, nil, opts, batchCount)

			fmt.Printf("Calculate DB %s with %d/%d entries.\n", dbNameTemp, c, e)
			totalCount += c
			totalEntry += e
		} (i)
	}
	wait.Wait()
	fmt.Printf("Total>block height: %d, worker: %d, batch: %d, calculation %d for %d entries\n",
		dbInfo.BlockHeight.Value, dbInfo.DbCount, batchCount, totalCount, totalEntry)

}