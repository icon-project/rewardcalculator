package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
	"path/filepath"
)

func iteratePrintDB(qdb db.Database, prefix *util.Range, printFunc func([]byte, []byte) error) error {
	// iterate
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Println("Failed to get iterator")
		return err
	}

	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		if err = printFunc(iter.Key(), iter.Value()); err != nil {
			fmt.Println("Error while iterate")
			return err
		}
	}
	iter.Release()

	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate. %+v\n", err)
		return err
	}
	return nil
}

func printDB(path string, prefix *util.Range, printFunc func([]byte, []byte) error) (err error) {
	fmt.Printf("===================== Data in %s =============\n", path)
	dir, name := filepath.Split(path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	err = iteratePrintDB(qdb, prefix, printFunc)
	return
}
