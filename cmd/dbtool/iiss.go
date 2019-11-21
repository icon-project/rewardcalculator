package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
)

func queryGovernanceVariables(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		fmt.Println("Can not query GovernanceVariable with block height 0")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixGovernanceVariable)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	gv := new(core.IISSGovernanceVariable)
	gv.BlockHeight = blockHeight
	value, err := bucket.Get(gv.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixGovernanceVariable)
	copy(qKey[2:], gv.ID())

	printGv(qKey, value)
}

func queryPRep(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		fmt.Println("Can not query PRep information with block height 0")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixGovernanceVariable)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	pRep := new(core.PRep)
	pRep.BlockHeight = blockHeight
	value, err := bucket.Get(pRep.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixGovernanceVariable)
	copy(qKey[2:], pRep.ID())

	printPRep(qKey, value)
}

func queryBPInfo(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		fmt.Println("Can not query BP information with block height 0")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixGovernanceVariable)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	bp := new(core.IISSBlockProduceInfo)
	bp.BlockHeight = blockHeight
	value, err := bucket.Get(bp.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixGovernanceVariable)
	copy(qKey[2:], bp.ID())

	printBP(qKey, value)
}
func queryTransaction(qdb db.Database, index int64) {
	if index < 0 {
		fmt.Println("Can not query transaction with negative index")
		return
	}
	bucket, err := qdb.GetBucket(db.PrefixIISSTX)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}
	idx := uint64(index)
	tx := new(core.IISSTX)
	tx.Index = idx
	value, err := bucket.Get(tx.ID())
	if value == nil || err != nil {
		return
	}
	qKey := make([]byte, 10)
	copy(qKey[:2], db.PrefixIISSTX)
	copy(qKey[2:], tx.ID())

	printTransaction(qKey, value)
}
func printHeader(key []byte, value []byte) bool {
	if string(key[:2]) != "HD" {
		return false
	}
	var header core.IISSHeader
	header.SetBytes(value)
	fmt.Println(header.String())
	return true
}

func printGv(key []byte, value []byte) bool {
	if string(key[:GVPrefixLen]) != "GV" {
		return false
	}
	gv := new(core.GovernanceVariable)
	gv.SetBytes(value)
	gv.BlockHeight = common.BytesToUint64(key[GVPrefixLen:])
	fmt.Println("Governance variable set", gv.GVData, " at ", gv.BlockHeight)
	return true
}

func printBP(key []byte, value []byte) bool {
	if string(key[:BlockProduceInfoPrefixLen]) != "BP" {
		return false
	}
	bpInfo := new(core.IISSBlockProduceInfo)
	bpInfo.SetBytes(value)
	bpInfo.BlockHeight = common.BytesToUint64(key[BlockProduceInfoPrefixLen:])
	fmt.Println(bpInfo.String())
	return true
}

func printPRep(key []byte, value []byte) bool {
	if string(key[:PRepPrefixLen]) != "PR" {
		return false
	}
	prep := new(core.PRep)
	prep.SetBytes(value)
	prep.BlockHeight = common.BytesToUint64(key[PRepPrefixLen:])
	fmt.Println(prep.String())
	return true
}

func printTransaction(key []byte, value []byte) bool {
	if string(key[:TransactionPrefixLen]) != "TX" {
		return false
	}
	tx := new(core.IISSTX)
	tx.SetBytes(value)
	tx.Index = common.BytesToUint64(key[TransactionPrefixLen:])
	fmt.Println(tx.String())
	return true
}
