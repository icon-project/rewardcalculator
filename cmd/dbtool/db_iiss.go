package main

import (
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func queryIISSDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db path")
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	switch input.data {
	case "":
		fmt.Println("============== Header ==============")
		if err = queryHeader(qdb); err != nil {
			return
		}
		fmt.Println("\n============== Governance variables ==============")
		if err = queryIISSGV(qdb); err != nil {
			return
		}
		fmt.Println("\n============== P-Rep ==============")
		if err = queryPRep(qdb, 0); err != nil {
			return
		}
		fmt.Println("\n============== Block produce info ==============")
		if err = queryBP(qdb, 0); err != nil {
			return
		}
		fmt.Println("\n============== Transactions ==============")
		if err = queryTX(qdb, 0); err != nil {
			return
		}
	case DataTypeGV:
		if err = queryIISSGV(qdb); err != nil {
			return
		}
	case DataTypeHeader:
		if err = queryHeader(qdb); err != nil {
			return
		}
	case DataTypeBP:
		if err = queryBP(qdb, input.height); err != nil {
			return
		}
	case DataTypePRep:
		if err = queryPRep(qdb, input.height); err != nil {
			return
		}
	case DataTypeTX:
		if err = queryTX(qdb, input.height); err != nil {
			return
		}
	default:
		return errors.New("invalid iiss data type")
	}
	return nil
}

func queryBP(qdb db.Database, blockHeight uint64) error {
	if blockHeight == 0 {
		err := iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixIISSBPInfo)), printBP)
		return err
	} else {
		if bp, err := getBP(qdb, blockHeight); err != nil {
			return err
		} else {
			printBlockProduceInfo(bp)
			return nil
		}
	}
}

func queryPRep(qdb db.Database, blockHeight uint64) error {
	if blockHeight == 0 {
		err := iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixPRep)), printPRep)
		return err
	} else {
		if prep, err := getPRep(qdb, blockHeight); err != nil {
			return err
		} else {
			fmt.Printf(prep.String())
			return nil
		}
	}
}

func queryTX(qdb db.Database, blockHeight uint64) (err error) {
	if blockHeight == 0 {
		err = iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixIISSTX)), printTX)
	} else {
		err = queryTransaction(qdb, blockHeight)
	}
	return
}

func queryHeader(qdb db.Database) (err error) {
	if header, e := newHeader(qdb); e != nil {
		return e
	} else {
		fmt.Println(header.String())
		return
	}
}

func queryIISSGV(qdb db.Database) error {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		return err
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSGV))
	header, e := newHeader(qdb)
	if e != nil {
		return e
	}
	version := header.Version
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		gv := new(core.IISSGovernanceVariable)
		key := make([]byte, len(iter.Key()))
		value := make([]byte, len(iter.Value()))
		copy(key, iter.Key())
		copy(value, iter.Value())
		gv.BlockHeight = common.BytesToUint64(key)
		if err = gv.SetBytes(value, version); err != nil {
			fmt.Println("Error while initialize IISS governance variable")
			return err
		}
		fmt.Println(gv.String())
	}
	iter.Release()
	return nil
}

func getPRep(qdb db.Database, blockHeight uint64) (prep *core.PRep, err error) {
	bucket, err := qdb.GetBucket(db.PrefixIISSPRep)
	if err != nil {
		fmt.Println("Failed to get Bucket")
		return nil, err
	}

	prep = new(core.PRep)
	prep.BlockHeight = blockHeight
	key := prep.ID()
	value, err := bucket.Get(key)
	if err != nil {
		fmt.Println("Error while getting prep info")
		return nil, err
	}
	if value == nil {
		fmt.Println("There is no prep info at ", blockHeight)
		return nil, nil
	}
	return newPRep(key, value)
}

func queryTransaction(qdb db.Database, blockHeight uint64) error {
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("error while getting iiss db iterator")
		return err
	}
	prefix := util.BytesPrefix([]byte(db.PrefixIISSTX))
	iter.New(prefix.Start, prefix.Limit)

	var transactions []*core.IISSTX
	found := false
	for iter.Next() {
		tx := new(core.IISSTX)
		key := make([]byte, len(iter.Key()))
		value := make([]byte, len(iter.Value()))
		copy(key, iter.Key())
		copy(value, iter.Value())
		tx.Index = common.BytesToUint64(key)
		tx.SetBytes(value)
		if tx.BlockHeight == blockHeight {
			found = true
			transactions = append(transactions, tx)
		}
	}
	iter.Release()

	if !found {
		fmt.Println("There is no transaction related iiss in block : ", blockHeight)
	} else {
		for _, tx := range transactions {
			printTXInstance(tx)
		}
	}
	return nil
}

func getBP(qdb db.Database, blockHeight uint64) (*core.IISSBlockProduceInfo, error) {
	bucket, err := qdb.GetBucket(db.PrefixIISSBPInfo)
	if err != nil {
		fmt.Println("error while getting block produce info bucket")
		return nil, err
	}

	bp := new(core.IISSBlockProduceInfo)
	bp.BlockHeight = blockHeight
	key := bp.ID()
	value, e := bucket.Get(key)
	if e != nil {
		fmt.Println("Error while getting block produce data")
		return nil, e
	}
	if value == nil {
		fmt.Println("There is no block produce info at ", blockHeight)
		return nil, nil
	}
	return newBP(key, value)
}

func printBP(key []byte, value []byte) error {
	if bpInfo, e := newBP(key, value); e != nil {
		return e
	} else {
		printBlockProduceInfo(bpInfo)
		return e
	}
}

func printBlockProduceInfo(bp *core.IISSBlockProduceInfo) {
	if bp != nil {
		fmt.Println(bp.String())
	}
}

func printPRep(key []byte, value []byte) (err error) {
	if prep, err := newPRep(key, value); err != nil {
		return err
	} else {
		fmt.Println(prep.String())
		return nil
	}
}

func printTX(key []byte, value []byte) (err error) {
	if tx, err := newTX(key, value); err != nil {
	} else {
		printTXInstance(tx)
	}
	return
}

func printTXInstance(tx *core.IISSTX) {
	fmt.Printf("%s\n", tx.String())
}

func newBP(key []byte, value []byte) (bp *core.IISSBlockProduceInfo, err error) {
	bp = new(core.IISSBlockProduceInfo)
	if err = bp.SetBytes(value); err != nil {
		fmt.Println("Error while initialize IISSBlockProduceInfo")
		return nil, err
	}
	bp.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSBPInfo):])
	return
}

func newPRep(key []byte, value []byte) (prep *core.PRep, err error) {
	prep = new(core.PRep)
	if err = prep.SetBytes(value); err != nil {
		return nil, err
	}
	prep.BlockHeight = common.BytesToUint64(key[len(db.PrefixIISSPRep):])
	return
}

func newTX(key []byte, value []byte) (tx *core.IISSTX, err error) {
	tx = new(core.IISSTX)
	tx.Index = common.BytesToUint64(key)
	if err = tx.SetBytes(value); err != nil {
		return nil, err
	}
	return
}

func newHeader(qdb db.Database) (*core.IISSHeader, error) {
	bucket, err := qdb.GetBucket(db.PrefixIISSHeader)
	if err != nil {
		fmt.Println("error while getting iiss header bucket")
		return nil, err
	}
	header := new(core.IISSHeader)
	value, e := bucket.Get(header.ID())
	if e != nil {
		fmt.Println("error while Get value of iiss header info")
		return nil, e
	}
	if err = header.SetBytes(value); e != nil {
		fmt.Println("error while initialize iiss header")
		return nil, e
	}
	return header, nil
}
