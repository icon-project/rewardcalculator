package main

import (
	"encoding/hex"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"log"
	"os"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func (cli *CLI) read(dbDir string, dbName string) {
	path := filepath.Join(dbDir, dbName)
	fmt.Printf("Start read IISS data DB. Name: %s\n", path)

	if _ , err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("There is no DB %s\n", path)
			return
		}
	}

	iissDB := core.OpenIISSData(path)
	core.LoadIISSData(iissDB)
	ReadIISSBP(iissDB)
	ReadIISSTX(iissDB)

	//checkIISSTX(iissDB, lossDB)

	checkALL(iissDB)

	iissDB.Close()
}

func ReadIISSBP(iissDB db.Database) {
	var bpInfo core.IISSBlockProduceInfo

	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSBPInfo))
	iter.New(prefix.Start, prefix.Limit)
	var entries, startBH, miss uint64
	for entries = 0; iter.Next(); entries++ {
		err := bpInfo.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to load IISS Block Produce information.")
			continue
		}
		bpInfo.BlockHeight = common.BytesToUint64(iter.Key()[len(db.PrefixIISSBPInfo):])
		if entries == 0 {
			startBH = bpInfo.BlockHeight
		}
		if startBH + entries + miss != bpInfo.BlockHeight {
			fmt.Printf("Miss BP block height %d\n", startBH + entries + miss)
			miss++
			for  ; startBH + entries + miss != bpInfo.BlockHeight; miss++ {
				fmt.Printf("Miss BP block height %d\n", startBH + entries + miss)
			}
		}
	}
	log.Printf(">> BP total count %d, miss %d", entries, miss)
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while read IISS BP iteration. %+v", err)
	}
}

func ReadIISSTX(iissDB db.Database) {
	var tx core.IISSTX

	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	entries := 0
	for entries = 0; iter.Next(); entries++ {
		err := tx.SetBytes(iter.Value())
		if err != nil {
			log.Printf("Failed to load IISS TX data")
			continue
		}
		//log.Printf("[IISSTX] TX : %s", tx.String())
	}
	log.Printf(">> TX total count %d", entries)
	iter.Release()
	err := iter.Error()
	if err != nil {
		log.Printf("There is error while read IISS TX iteration. %+v", err)
	}
}

func checkIISSTX(iissDB db.Database, result db.Database) {
	var tx core.IISSTX

	bucket, _ := result.GetBucket(db.PrefixIISSTX)

	iter, _ := iissDB.GetIterator()
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)
	total := 0
	for total = 0; iter.Next(); total++ {
		err := tx.SetBytes(iter.Value())
		if err != nil {
			fmt.Printf("Failed to load IISS TX data")
			continue
		}
		//fmt.Printf("[IISSTX] TX : %s", tx.String())
		v, err := bucket.Get(iter.Key()[len(db.PrefixIISSTX):])
		if v == nil || err != nil {
			fmt.Printf("Loss TX %d: %s", total, tx.String())
		}
	}
	fmt.Printf(">> TX total count %d", total)
	iter.Release()
}

func checkALL(iissDB db.Database) {
	iter, _ := iissDB.GetIterator()
	iter.New(nil, nil)
	stats := struct {
		total   uint64
		header  uint64
		gv      uint64
		bp      uint64
		pRep    uint64
		tx      uint64
		unknown uint64
	}{}

	fmt.Printf(">> Start to check IISS Data\n")

	for stats.total = 0; iter.Next(); stats.total++ {
		key := iter.Key()
		keyID := key[0:2]
		value := iter.Value()
		switch string(keyID) {
		case string(db.PrefixIISSHeader):
			stats.header++
		case string(db.PrefixIISSGV):
			stats.gv++
		case string(db.PrefixIISSBPInfo):
			stats.bp++
		case string(db.PrefixIISSPRep):
			stats.pRep++
		case string(db.PrefixIISSTX):
			stats.tx++
		default:
			fmt.Printf("Unknown key : %s / %s, value : %s\n",
				hex.EncodeToString(key), string(key), hex.EncodeToString(value))
			stats.unknown++
		}
	}
	iter.Release()

	fmt.Printf("Total  : %32d\n", stats.total)
	fmt.Printf("header : %32d\n", stats.header)
	fmt.Printf("gv     : %32d\n", stats.gv)
	fmt.Printf("bp     : %32d\n", stats.bp)
	fmt.Printf("prep   : %32d\n", stats.pRep)
	fmt.Printf("tx     : %32d\n", stats.tx)
	fmt.Printf("unknown: %32d\n", stats.unknown)
}
