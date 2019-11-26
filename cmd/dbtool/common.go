package main

import (
	"flag"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"log"
	"os"
)

func iteratePrintDB(dbType string, qDB db.Database, addr *common.Address, blockHeight uint64, index string) {
	// iterate
	iter, err := qDB.GetIterator()
	if err != nil {
		log.Printf("Failed to get iterator")
		return
	}

	iter.New(nil, nil)
	i := 0
	printCount := 0
	for iter.Next() {
		ret := printEntry(dbType, iter.Key(), iter.Value(), addr, blockHeight, index)
		if ret {
			printCount++
		}
		i++
	}
	iter.Release()

	fmt.Printf("Print %d entries in %d entries\n", printCount, i)

	err = iter.Error()
	if err != nil {
		log.Printf("Error while iterate. %+v", err)
		return
	}
}

func printEntry(dbType string, key []byte, value []byte, address *common.Address, blockHeight uint64, index string) bool {
	ret := false

	switch dbType {
	case DBTypeManagement:
		ret = printManagement(key, value)
	case DBTypeAccount:
		ret = printAccount(key, value, address)
	case DBTypeClaim:
		ret = printClaim(key, value, address)
	case DBTypePreCommit:
		ret = printPreCommit(key, value, address, blockHeight)
	case DBTypeCalcResult:
		ret = printCalcResult(key, value, blockHeight)
	case DBTypeHeader:
		ret = printHeader(key, value)
	case DBTypeGV:
		ret = printGv(key, value, blockHeight)
	case DBTypeBPInfo:
		ret = printBP(key, value, blockHeight)
	case DBTypePRep:
		ret = printPRep(key, value, blockHeight)
	case DBTypeTX:
		ret = printTransaction(key, value, index)
	default:
		fmt.Printf("invalid dbtype %s\n", "")
	}

	return ret
}

func validateInput(flagSet *flag.FlagSet, err error, flag bool) {
	if err != nil {
		flagSet.PrintDefaults()
		os.Exit(1)
	}
	if flag {
		flagSet.PrintDefaults()
		os.Exit(0)
	}
}
