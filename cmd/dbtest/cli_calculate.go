package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
	"sync"
)

func calculatePRepReward(delegation *common.HexInt, start uint64, end uint64,
	gvList []*rewardcalculator.GovernanceVariable, pRep *rewardcalculator.PRepCandidate) *common.HexInt {
	// TODO implement calculation logic

	// adjust start and end with P-Rep candidate
	if start < pRep.Start {
		start = pRep.Start
	}

	if pRep.End != 0 && pRep.End < end {
		end = pRep.End
	}

	total := new(common.HexInt)

	// period in gv
	for i, gv := range gvList {
		var s, e = start, end

		if start < gv.BlockHeight {
			s = gv.BlockHeight
		}
		if i+1 < len(gvList) && gvList[i+1].BlockHeight < end {
			e = gvList[i+1].BlockHeight
		}

		if e - s <= 0 {
			continue
		}
		period := common.NewHexIntFromUint64(e - s)

		// reward = delegation amount * period * GV
		var reward common.HexInt
		reward.Mul(&delegation.Int, &period.Int)
		reward.Mul(&reward.Int, &gv.RewardRep.Int)

		//log.Printf("dg: %s, period: %s, Rep: %s. reward: %s\n",
		//	delegation.String(), period.String(), gv.RewardRep.String(), reward.String())

		// update total
		total.Add(&total.Int, &reward.Int)
	}

	return total
}

func calculateIScore(ia *rewardcalculator.IScoreData,  gvList []*rewardcalculator.GovernanceVariable,
	pRepCandidates map[common.Address]*rewardcalculator.PRepCandidate, blockHeight uint64) bool {
	// IScore = old + period * G.V * sum(valid dgAmount)
	if blockHeight == 0 {
		blockHeight = ia.BlockHeight + 1
	}

	if (blockHeight - ia.BlockHeight) == 0 {
		return false
	}

	// calculate I-Score
	var totalReward common.HexInt
	for _, dg := range ia.Delegations {
		if dg.Delegate.Int.Sign() == 0 {
			// there is no delegation
			continue
		}
		reward := calculatePRepReward(&dg.Delegate, ia.BlockHeight, blockHeight, gvList, pRepCandidates[dg.Address])

		// update totalReward
		totalReward.Add(&totalReward.Int, &reward.Int)
	}

	// increase value
	ia.IScore.Add(&ia.IScore.Int, &totalReward.Int)

	// update BlockHeight
	ia.BlockHeight = blockHeight

	return true
}

func calculateDB(dbi db.Database, gvList []*rewardcalculator.GovernanceVariable,
	pRepCandidates map[common.Address]*rewardcalculator.PRepCandidate,
	blockHeight uint64, batchCount uint64) (count uint64, entries uint64) {
	bucket, _ := dbi.GetBucket(db.PrefixIScore)
	iter, _ := dbi.GetIterator()
	batch, _ := dbi.GetBatch()
	entries = 0; count = 0

	batch.New()
	iter.New(nil, nil)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(db.PrefixIScore):]
		ia, err := rewardcalculator.NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			log.Printf("Can't read data with iterator\n")
			return 0, 0
		}

		//log.Printf("Read data: %s\n", ia.String())

		// calculate
		if calculateIScore(&ia.IScoreData, gvList, pRepCandidates, blockHeight) == false {
			continue
		}

		//log.Printf("Updated data: %s\n", ia.String())

		value, _ := ia.Bytes()

		if batchCount > 0 {
			batch.Set(iter.Key(), value)

			// write batch to DB
			if entries != 0 && (entries % batchCount) == 0 {
				err = batch.Write()
				if err != nil {
					log.Printf("Failed to write batch\n")
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
			log.Printf("Failed to write batch\n")
		}
		batch.Reset()
	}

	// finalize iterator
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while iteration. %+v", err)
	}

	//log.Printf("Calculate %d entries for prefix %v-%v %d entries\n", count, start, limit, entries)

	return count, entries
}

func (cli *CLI) calculate(dbName string, blockHeight uint64, batchCount uint64) {
	//log.Printf("Start calculate DB. name: %s, block height: %d, batch count: %d\n", dbName, blockHeight, batchCount)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	dbInfo, err := rewardcalculator.NewDBInfo(lvlDB, 0)
	if err != nil {
		log.Printf("Failed to read DB Info. err=%+v\n", err)
		return
	}
	//log.Printf("DBInfo %s\n", dbInfo.String())

	bInfo, err := rewardcalculator.NewBlockInfo(lvlDB)
	if err != nil {
		log.Printf("Failed to read Block Info. err=%+v\n", err)
		return
	}
	//log.Printf("BlockInfo %s\n", bInfo.String())

	// make global options
	opts := new(rewardcalculator.GlobalOptions)

	opts.BlockHeight = bInfo.BlockHeight
	if blockHeight != 0 && blockHeight <= bInfo.BlockHeight {
		log.Printf("I-Score was calculated to %d already\n", blockHeight)
		return
	}

	// load Governance variable to GV list
	opts.GV, err = rewardcalculator.LoadGovernanceVariable(lvlDB, opts.BlockHeight)
	if err != nil {
		log.Printf("Failed to load GV structure. err=%+v\n", err)
		return
	}

	// load P-Rep candidate list to PRepCandidates map
	opts.PRepCandidates, err = rewardcalculator.LoadPRepCandidate(lvlDB)
	if err != nil {
		log.Printf("Failed to load GV structure. err=%+v\n", err)
		return
	}

	gvList := opts.GetGVList(opts.BlockHeight, blockHeight)
	if len(gvList) == 0 {
		log.Printf("Can't get Governance variables for block (%d, %d)\n", opts.BlockHeight, blockHeight)
		return
	}

	var totalCount, totalEntry uint64
	var wait sync.WaitGroup
	wait.Add(dbInfo.DBCount)

	dbDir := fmt.Sprintf("%s/%s", DBDir, dbName)
	for i:= 0; i< dbInfo.DBCount; i++ {
		go func(index int) {
			dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbInfo.DBCount)
			acctDB := db.Open(dbDir, DBType, dbNameTemp)
			defer acctDB.Close()
			defer wait.Done()

			c, e := calculateDB(acctDB, gvList, opts.PRepCandidates, blockHeight, batchCount)

			log.Printf("Calculate DB %s with %d/%d entries.\n", dbNameTemp, c, e)
			totalCount += c
			totalEntry += e
		} (i)
	}
	wait.Wait()
	log.Printf("Total>block height: %d -> %d, worker: %d, batch: %d, calculation %d for %d entries\n",
		bInfo.BlockHeight, blockHeight, dbInfo.DBCount, batchCount, totalCount, totalEntry)

	// update blockInfo
	if blockHeight == 0 {
		bInfo.BlockHeight++
	} else {
		bInfo.BlockHeight = blockHeight
	}
	bucket, err := lvlDB.GetBucket(db.PrefixBlockInfo)
	value, _ := bInfo.Bytes()
	bucket.Set(bInfo.ID(), value)
}