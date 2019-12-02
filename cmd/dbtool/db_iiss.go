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
		entries := getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSBPInfo)))
		printEntries(entries, printBP)
		entries = getEntries(qdb, util.BytesPrefix([]byte(db.PrefixPRep)))
		printEntries(entries, printPRep)
		printHeader(qdb)
		printIISSGV(qdb)
		entries = getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSTX)))
		printEntries(entries, printTX)
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
		getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSBPInfo)))
	} else {
		runQueryBP(qdb, blockHeight)
	}
}

func queryPRep(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		getEntries(qdb, util.BytesPrefix([]byte(db.PrefixPRep)))
		return
	} else {
		runQueryPRep(qdb, blockHeight)
	}
}

func queryTX(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		getEntries(qdb, util.BytesPrefix([]byte(db.PrefixIISSTX)))
	} else {
		runQueryTX(qdb, blockHeight)
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
	gv := new(core.IISSGovernanceVariable)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		gv.BlockHeight = common.BytesToUint64(key)
		gv.SetBytes(value, version)
		fmt.Println("Governance Variable : ", gv.String())
	}
	iter.Release()
}

func printBP(key []byte, value []byte){
	bpInfo := new(core.IISSBlockProduceInfo)
	bpInfo.SetBytes(value)
	bpInfo.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSBPInfo):])

	fmt.Println(bpInfo.String())
}

func printPRep(key []byte, value []byte) {
	prep := new(core.PRep)
	prep.SetBytes(value)
	prep.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSPRep):])

	fmt.Println(prep.String())
}

func printTX(key []byte, value []byte) {
	tx := new(core.IISSTX)
	tx.Index = common.BytesToUint64(key)
	tx.SetBytes(value)

	fmt.Printf("%s\n", tx.String())
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

func runQueryPRep(qdb db.Database, blockHeight uint64){
	bucket, err := qdb.GetBucket(db.PrefixIISSPRep)
	if err != nil {
		fmt.Printf("Failed to get Bucket")
		return
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
		return
	}
	printPRep(pRep.ID(), value)
}

func runQueryTX(qdb db.Database, blockHeight uint64){
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)

	var transactions []*core.IISSTX
	txExistInHeihgt := false
	tx := new(core.IISSTX)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		tx.Index = common.BytesToUint64(key)
		tx.SetBytes(value)
		if tx.BlockHeight == blockHeight {
			txExistInHeihgt = true
			transactions = append(transactions, tx)
		}
	}
	iter.Release()

	if txExistInHeihgt {
		for _, value := range transactions {
			byteValue, _ := tx.Bytes()
			printTX(common.Uint64ToBytes(value.Index), byteValue)
		}
	} else {
		fmt.Println("No iiss related transaction in block", blockHeight)
	}
}

func runQueryBP(qdb db.Database, blockHeight uint64){
	bucket, err := qdb.GetBucket(db.PrefixIISSBPInfo)
	if err != nil {
		fmt.Printf("error while getting block produce info bucket")
		os.Exit(1)
		return
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
		return
	}
	printBP(bp.ID(), value)
}
