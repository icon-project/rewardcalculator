package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
	"path/filepath"
)

func iteratePrintDB(qdb db.Database, prefix *util.Range, printFunc func([]byte, []byte)) {
	// iterate
	iter, err := qdb.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get iterator")
		os.Exit(1)
	}

	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		printFunc(iter.Key(), iter.Value())
	}
	iter.Release()

	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate. %+v", err)
		os.Exit(1)
	}
}

func printAllEntriesInPath(path string, prefix *util.Range, printFunc func([]byte, []byte)) {
	fmt.Printf("=====================Querying data in %s=============\n", path)
	dir, name := filepath.Split(path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()
	iteratePrintDB(qdb, prefix, printFunc)
}
