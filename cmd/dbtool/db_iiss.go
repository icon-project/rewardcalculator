package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"path/filepath"
)

func queryIISSDB(input Input) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	switch input.data {
	case "":
		fmt.Println("==============print block produce info==============")
		queryBP(qdb, 0)
		fmt.Println("==============print prep info==============")
		queryPRep(qdb, 0)
		fmt.Println("==============print header==============")
		printHeader(qdb)
		fmt.Println("==============print governance variables==============")
		printIISSGV(qdb)
		fmt.Println("==============print transactions==============")
		queryTX(qdb, 0)
	case DataTypeGV:
		printIISSGV(qdb)
	case DataTypeHeader:
		printHeader(qdb)
	case DataTypeBP:
		queryBP(qdb, input.height)
	case DataTypePRep:
		queryPRep(qdb, input.height)
	case DataTypeTX:
		queryTX(qdb, input.height)
	default:
		fmt.Println("invalid iiss data type")
		os.Exit(1)
	}
}

func queryBP(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSBPInfo)))
		printEntries(entries, printBP)
	} else {
		bp := runQueryBP(qdb, blockHeight)
		if bp != nil {
			fmt.Println(bp.String())
		}
	}
}

func queryPRep(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixPRep)))
		printEntries(entries, printPRep)
	} else {
		prep := runQueryPRep(qdb, blockHeight)
		if prep != nil {
			fmt.Println(prep.String())
		}
	}
}

func queryTX(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSTX)))
		printEntries(entries, printTX)
	} else {
		transactions := runQueryTX(qdb, blockHeight)
		if len(transactions) == 0 {
			fmt.Println("There is no transaction data in block : ", blockHeight)
		}
		for _, v := range transactions{
			fmt.Println(v.String())
		}
	}
}

func printHeader(qdb db.Database) {
	header := getHeader(qdb)
	fmt.Println("iiss header info : ", header.String())
}

func printIISSGV(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSGV))
	header := getHeader(qdb)
	version := header.Version
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		gv := new(core.IISSGovernanceVariable)
		key := make([]byte, len(iter.Key()))
		value := make([]byte, len(iter.Value()))
		copy(key, iter.Key())
		copy(value, iter.Value())
		gv.BlockHeight = common.BytesToUint64(key)
		gv.SetBytes(value, version)
		fmt.Println("Governance Variable : ", gv.String())
	}
	iter.Release()
}

func printBP(key []byte, value []byte){
	bpInfo := getBP(key, value)
	fmt.Println(bpInfo.String())
}
func printPRep(key []byte, value []byte) {
	prep := getPRep(key, value)
	fmt.Println(prep.String())
}
func printTX(key []byte, value []byte) {
	tx := getTX(key, value)
	fmt.Printf("%s\n", tx.String())
}

func runQueryPRep(qdb db.Database, blockHeight uint64) *core.PRep{
	bucket, err := qdb.GetBucket(db.PrefixIISSPRep)
	if err != nil {
		fmt.Printf("Failed to get Bucket")
		os.Exit(1)
	}

	pRep := new(core.PRep)
	pRep.BlockHeight = blockHeight
	value, err := bucket.Get(pRep.ID())
	if err != nil {
		fmt.Println("Error while getting prep info")
		os.Exit(1)
	}
	if value == nil {
		fmt.Println("There is no prep info at ", blockHeight)
		return nil
	}
	prep := getPRep(pRep.ID(), value)
	return prep
}

func runQueryTX(qdb db.Database, blockHeight uint64) []*core.IISSTX{
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)

	var transactions []*core.IISSTX
	for iter.Next() {
		tx := new(core.IISSTX)
		key := make([]byte, len(iter.Key()))
		value := make([]byte, len(iter.Value()))
		copy(key, iter.Key())
		copy(value, iter.Value())
		tx.Index = common.BytesToUint64(key)
		tx.SetBytes(value)
		if tx.BlockHeight == blockHeight {
			transactions = append(transactions, tx)
		}
	}
	iter.Release()

	return transactions
}

func runQueryBP(qdb db.Database, blockHeight uint64) *core.IISSBlockProduceInfo{
	bucket, err := qdb.GetBucket(db.PrefixIISSBPInfo)
	if err != nil {
		fmt.Printf("error while getting block produce info bucket")
		os.Exit(1)
	}

	bp := new(core.IISSBlockProduceInfo)
	bp.BlockHeight = blockHeight
	value, err := bucket.Get(bp.ID())
	if err != nil {
		fmt.Println("Error while getting block produce data")
		os.Exit(1)
	}
	if value == nil {
		fmt.Println("There is no block produce info at ", blockHeight)
		return nil
	}
	result := getBP(bp.ID(), value)
	return result
}

func getBP(key []byte, value []byte) *core.IISSBlockProduceInfo{
	bpInfo := new(core.IISSBlockProduceInfo)
	bpInfo.SetBytes(value)
	bpInfo.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSBPInfo):])
	return bpInfo
}

func getPRep(key []byte, value []byte) *core.PRep{
	prep := new(core.PRep)
	prep.SetBytes(value)
	prep.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSPRep):])
	return prep
}

func getTX(key []byte, value []byte) *core.IISSTX{
	tx := new(core.IISSTX)
	tx.Index = common.BytesToUint64(key)
	tx.SetBytes(value)
	return tx
}

func getHeader(qdb db.Database) *core.IISSHeader {
	bucket, err := qdb.GetBucket(db.PrefixIISSHeader)
	if err != nil {
		fmt.Println("error while getting iiss header bucket")
		os.Exit(1)
	}
	header := new(core.IISSHeader)
	value, err := bucket.Get(header.ID())
	if err != nil {
		fmt.Println("error while Get value of iiss header info")
	}
	header.SetBytes(value)
	return header
}
