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
		queryBP(qdb, input.height)
		queryPRep(qdb, input.height)
		printHeader(qdb)
		printIISSGV(qdb)
		queryTX(qdb, input.height)
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
		printBP(qdb)
		return
	}
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
	bp.SetBytes(value)
	fmt.Println("block produce info :", bp.String())
}

func queryPRep(qdb db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		printPRep(qdb)
		return
	}
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
	pRep.SetBytes(value)
	fmt.Println("Prep : ", pRep.String())
}

func queryTX(qdb db.Database, height uint64) {
	if height == 0 {
		printTX(qdb)
		return
	}
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)

	var transactions []string
	txExistInHeihgt := false
	tx := new(core.IISSTX)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		tx.Index = common.BytesToUint64(key)
		tx.SetBytes(value)
		if tx.BlockHeight == height {
			txExistInHeihgt = true
			transactions = append(transactions, tx.String())
		}
	}
	iter.Release()

	if txExistInHeihgt {
		fmt.Println("transactions : ", transactions)
	} else {
		fmt.Println("No iiss related transaction in block", height)
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

func printBP(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSBPInfo))
	iter.New(prefix.Start, prefix.Limit)
	bp := new(core.IISSBlockProduceInfo)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		bp.BlockHeight = common.BytesToUint64(key)
		bp.SetBytes(value)
		fmt.Println("Block produce info : ", bp.String())
	}
	iter.Release()
}

func printPRep(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSPRep))
	iter.New(prefix.Start, prefix.Limit)
	prep := new(core.PRep)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		prep.BlockHeight = common.BytesToUint64(key)
		prep.SetBytes(value)
		fmt.Println("prep : ", prep.String())
	}
	iter.Release()
}

func printTX(qdb db.Database) {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		os.Exit(1)
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	tx := new(core.IISSTX)
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		tx.Index = common.BytesToUint64(key)
		tx.SetBytes(value)
		fmt.Println("Transaction : ", tx.String())
	}
	iter.Release()
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
