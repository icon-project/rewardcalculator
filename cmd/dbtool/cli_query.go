package main

import (
	//"encoding/hex"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
	"path/filepath"
)

func (cli *CLI) query(dbName string, dbType string, address string, blockHeight uint64) {
	fmt.Printf("Query DB name: %s DB type: %s", dbName, dbType)
	if address != "" {
		fmt.Printf(", Address: %s", address)
	}
	if blockHeight != 0 {
		fmt.Printf(", BlockHeight: %d", blockHeight)
	}
	fmt.Printf("\n")

	dir, name := filepath.Split(dbName)

	qDB := db.Open(dir, string(db.GoLevelDBBackend), name)

	// covert arguments
	var addr *common.Address
	if address != "" {
		addr = common.NewAddressFromString(address)
	} else {
		addr = nil
	}

	fmt.Printf("### Results\n")

	if addr == nil && blockHeight == 0 {	// print ALL
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
			ret := printEntry(iter.Key(), iter.Value(), dbType, addr, blockHeight)
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
	} else { // query with address or blockHeight
		queryEntry(qDB, dbType, addr, blockHeight)
	}
}

func queryEntry(qDB db.Database, dbType string, address *common.Address, blockHeight uint64) {
	if address == nil && blockHeight == 0 {
		return
	}
	switch dbType {
	case DBTypeManagemment:
		fmt.Printf("Can't query management DB with address or blockheight")
	case DBTypeAccount:
		queryAccount(qDB, address)
	case DBTypeClaim:
		queryClaim(qDB, address)
	case DBTypePreCommit:
		queryPreCommit(qDB, address, blockHeight)
	case DBTypeCalcResult:
		queryCalcResult(qDB, blockHeight)
	default:
		fmt.Printf("invalid dbtype %s\n", dbType)
	}
}

func queryAccount(qDB db.Database, address *common.Address) {
	if address == nil {
		fmt.Printf("Can't query account DB with nil address\n")
		return
	}

	bucket, err := qDB.GetBucket(db.PrefixIScore)
	if err != nil {
		log.Printf("Failed to get bucket")
		return
	}

	value, err := bucket.Get(address.Bytes())
	if value == nil || err != nil {
		return
	}

	printAccount(address.Bytes(), value)
}

func queryClaim(qDB db.Database, address *common.Address) {
	if address == nil {
		fmt.Printf("Can't query claim DB with nil address\n")
		return
	}
}

func queryPreCommit(qDB db.Database, address *common.Address, blockHeight uint64) {
}

func queryCalcResult(qDB db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		fmt.Printf("Can't query calculation result DB with block height 0\n")
		return
	}
}

func printEntry(key []byte, value []byte, dbType string, address *common.Address, blockHeight uint64) bool {
	ret := false

	switch dbType {
	case DBTypeManagemment:
		ret = printManagement(key, value)
	case DBTypeAccount:
		ret = printAccount(key, value)
	case DBTypeClaim:
		ret = printClaim(key, value, address)
	case DBTypePreCommit:
		ret = printPreCommit(key, value, address, blockHeight)
	case DBTypeCalcResult:
		ret = printCalcResult(key, value, blockHeight)
	default:
		fmt.Printf("invalid dbtype %s\n", dbType)
	}

	return ret
}

func printManagement(key []byte, value []byte) bool {

	return true
}

func printAccount(key []byte, value []byte) bool {
	return true
}

func printClaim(key []byte, value []byte, address *common.Address) bool {
	claim, err := core.NewClaimFromBytes(value)
	if err != nil {
		log.Printf("Failed to make claim instance")
		return false
	}
	claim.Address = *common.NewAddress(key)

	// check argument
	if address != nil && claim.Address.Equal(address) == false {
		return false
	}

	fmt.Printf("%s\n", claim.String())
	return true
}

func printPreCommit(key []byte, value []byte, address *common.Address, blockHeight uint64) bool {
	var pc core.PreCommit

	err := pc.SetBytes(value)
	if err != nil {
		log.Printf("Failed to make preCommit instance")
		return false
	}
	pc.BlockHeight = common.BytesToUint64(key[:core.BlockHeightSize])
	pc.BlockHash = make([]byte, core.BlockHashSize)
	copy(pc.BlockHash, key[core.BlockHeightSize:core.BlockHeightSize+core.BlockHashSize])

	// check argument
	if address != nil && pc.Address.Equal(address) == false {
		return false
	}

	if blockHeight != 0 && pc.BlockHeight != blockHeight {
		return false
	}

	fmt.Printf("%s\n", pc.String())

	return true
}

func printCalcResult(key []byte, value []byte, blockHeight uint64) bool {

	return true
}
