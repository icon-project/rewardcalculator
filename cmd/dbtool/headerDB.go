package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"os"
	"path/filepath"
)

type HeaderDB struct {
	dbPath string
}

func (headerDB HeaderDB) query() {
	if headerDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(headerDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	iteratePrintDB(DBTypeHeader, qdb, nil, 0, "")
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
