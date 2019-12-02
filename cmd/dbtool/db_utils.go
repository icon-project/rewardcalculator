package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
	"os"
)

type Entry struct {
	key []byte
	value []byte
}
func getEntries(qDB db.Database, prefix *util.Range) []Entry{
	// iterate
	iter, err := qDB.GetIterator()
	if err != nil {
		fmt.Printf("Failed to get iterator")
		os.Exit(1)
	}

	var entries []Entry
	iter.New(prefix.Start, prefix.Limit)
	for iter.Next() {
		entry := Entry{key:iter.Key(), value: iter.Value()}
		entries = append(entries, entry)
	}
	iter.Release()

	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate. %+v", err)
		os.Exit(1)
	}
	return entries
}

func printEntries(entries []Entry, printFunc func([]byte, []byte)){
	for _, v := range entries{
		printFunc(v.key, v.value)
	}
}
