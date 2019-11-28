package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"os"
)

func iteratePrintDB(dataType string, qDB db.Database) {
	// iterate
	iter, err := qDB.GetIterator()
	if err != nil {
		log.Printf("Failed to get iterator")
		return
	}

	prefix := getPrefix(dataType)
	iter.New(prefix.Start, prefix.Limit)
	i := 0
	printCount := 0
	for iter.Next() {
		ret := printEntry(dataType, iter.Key(), iter.Value())
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

func printEntry(dataType string, key []byte, value []byte) bool {
	ret := false

	switch dataType {
	case DBNameAccount:
		ret = printAccount(key, value, nil)
	case DBNameClaim:
		ret = printClaim(key, value, nil)
	case DBNamePreCommit:
		ret = printPreCommit(key, value, nil, 0)
	case DBNameCalcResult:
		ret = printCalcResult(key, value, 0)
	//case DBNameIISS:
	//	ret = printHeader(key, value)
	default:
		fmt.Printf("invalid dbtype %s\n", "")
	}

	return ret
}

func getPrefix(dataType string) *util.Range {
	var prefix *util.Range
	switch dataType {
	case DataTypeAccount:
		prefix = util.BytesPrefix([]byte(db.PrefixIScore))
	case DataTypeClaim:
		prefix = util.BytesPrefix([]byte(db.PrefixClaim))
	case DataTypeCalcResult:
		prefix = util.BytesPrefix([]byte(db.PrefixCalcResult))
	case DataTypePreCommit:
		prefix = util.BytesPrefix([]byte(db.PrefixClaim))
	case DataTypeDI:
		prefix = util.BytesPrefix([]byte(db.PrefixManagement))
	case DataTypeGV:
		prefix = util.BytesPrefix([]byte(db.PrefixGovernanceVariable))
	case DataTypePC:
		prefix = util.BytesPrefix([]byte(db.PrefixPRepCandidate))
	case DataTypeTX:
		prefix = util.BytesPrefix([]byte(db.PrefixIISSTX))
	case DataTypeBP:
		prefix = util.BytesPrefix([]byte(db.PrefixIISSBPInfo))
	case DataTypeHeader:
		prefix = util.BytesPrefix([]byte(db.PrefixIISSHeader))
	case DataTypePRep:
		prefix = util.BytesPrefix([]byte(db.PrefixIISSPRep))
	default:
		fmt.Printf("Invalid dataType : %s", dataType)
		os.Exit(1)
	}
	return prefix
}
