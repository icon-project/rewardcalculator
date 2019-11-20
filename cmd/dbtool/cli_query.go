package main

import (
	"bytes"
	//"encoding/hex"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"log"
	"os"
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

	// covert arguments
	var addr *common.Address
	if address != "" {
		addr = common.NewAddressFromString(address)
	} else {
		addr = nil
	}
	var dbInfo *core.DBInfo
	if dbType == DBTypeAccount {
		mdb := getDB(dbName, DBTypeManagement, nil, nil)
		bucket, err := mdb.GetBucket(db.PrefixManagement)
		dbInfo = new(core.DBInfo)
		data, err := bucket.Get(dbInfo.ID())
		if data != nil {
			err = dbInfo.SetBytes(data)
			if err != nil {
				fmt.Println("Failed to set DB Information structure")
				os.Exit(1)
			}
		} else {
			fmt.Println("Invalid IScore DB")
			os.Exit(1)
		}
	}

	fmt.Printf("### Results\n")

	if addr == nil && blockHeight == 0 { // print ALL
		if dbType == DBTypeAccount {
			for i := 0; i < dbInfo.DBInfoData.DBCount; i++ {
				addr = new(common.Address)
				tmpBytes := make([]byte, common.AddressBytes)
				tmpBytes[0] = 0
				tmpBytes[1] = byte(i)
				addr.SetBytes(tmpBytes)
				qdb := getDB(dbName, dbType, addr, dbInfo)
				iteratePrintDB(qdb, dbType, addr, blockHeight)
			}
		} else {
			qdb := getDB(dbName, dbType, addr, dbInfo)
			iteratePrintDB(qdb, dbType, addr, blockHeight)
		}
	} else { // query with address or blockHeight
		qDB := getDB(dbName, dbType, addr, dbInfo)
		queryEntry(qDB, dbType, addr, blockHeight)
	}
}

func queryEntry(qDB db.Database, dbType string, address *common.Address, blockHeight uint64) {
	if address == nil && blockHeight == 0 {
		return
	}
	switch dbType {
	case DBTypeManagement:
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
	bucket, err := qDB.GetBucket(db.PrefixIScore)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}
	value, err := bucket.Get(address.Bytes())
	if value == nil || err != nil {
		return
	}

	printClaim(address.Bytes(), value, address)
}

func queryPreCommit(qDB db.Database, address *common.Address, blockHeight uint64) {
	if address == nil || blockHeight == 0 {
		fmt.Printf("Can not query precommit DB without address and blockHeight")
	}

	iter, err := qDB.GetIterator()
	if err != nil {
		log.Printf("Failed to get iterator")
		return
	}

	iter.New(nil, nil)
	preCommitKey := make([]byte, core.BlockHashSize+core.BlockHeightSize+common.AddressBytes)
	keyExist := false
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := iter.Key()
		if bytes.Equal(key[:core.BlockHeightSize], blockHeightBytesValue) &&
			bytes.Equal(key[core.BlockHeightSize+core.BlockHashSize:], address.Bytes()) {
			keyExist = true
			copy(preCommitKey, key)
		}
	}
	iter.Release()

	if keyExist == false {
		fmt.Println("Can not find key using given information")
		return
	}
	err = iter.Error()
	if err != nil {
		fmt.Printf("Error while iterate")
		return
	}

	bucket, err := qDB.GetBucket(db.PrefixClaim)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	value, err := bucket.Get(preCommitKey)
	if value == nil || err != nil {
		return
	}
	printPreCommit(preCommitKey, value, address, blockHeight)
}

func queryCalcResult(qDB db.Database, blockHeight uint64) {
	if blockHeight == 0 {
		fmt.Printf("Can't query calculation result DB with block height 0\n")
		return
	}
	bucket, err := qDB.GetBucket(db.PrefixCalcResult)
	if err != nil {
		log.Printf("Failed to get Bucket")
		return
	}

	value, err := bucket.Get(common.Uint64ToBytes(blockHeight))
	if value == nil || err != nil {
		return
	}
	printCalcResult(common.Uint64ToBytes(blockHeight), value, blockHeight)
}

func printEntry(key []byte, value []byte, dbType string, address *common.Address, blockHeight uint64) bool {
	ret := false

	switch dbType {
	case DBTypeManagement:
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
	var result string
	switch string(key[:2]) {
	case "MI":
		dbi := new(core.DBInfo)
		dbi.SetBytes(value)
		result = fmt.Sprint("dbInfo : ", dbi.String())
	case "GV":
		gv := new(core.GovernanceVariable)
		gv.BlockHeight = common.BytesToUint64(key[GVPrefixLen:])
		gv.SetBytes(value)
		result = fmt.Sprint("Governance Variables : ", gv.String())
	case "PC":
		pc := new(core.PRepCandidate)
		pc.SetBytes(value)
		pc.Address = *common.NewAddress(key[PRepCandidatePrefixLen:])
		result = fmt.Sprint("PRepCandidate : ", pc.String())
	}
	fmt.Println(result)

	return true
}

func printAccount(key []byte, value []byte) bool {
	account, err := core.NewIScoreAccountFromBytes(value)
	if err != nil {
		log.Printf("Failed to make claim instance")
		fmt.Println("printAccount1")
		return false
	}
	account.Address = *common.NewAddress(key)
	fmt.Printf("%s\n", account.String())

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
	var cr core.CalculationResult
	err := cr.SetBytes(value)
	if err != nil {
		log.Printf("Failed to make calculateResult instance")
		return false
	}
	cr.BlockHeight = common.BytesToUint64(key)

	if blockHeight != 0 && blockHeight != cr.BlockHeight {
		return false
	}

	fmt.Printf("%s\n", cr.String())

	return true
}

func getDB(dbName string, dbType string, address *common.Address, dbInfo *core.DBInfo) db.Database {
	var dbRoot string
	switch dbType {
	case DBTypeManagement:
		dbRoot = dbName
	case DBTypeCalcResult:
		dbRoot = filepath.Join(dbName, CalcResultPath)
	case DBTypePreCommit:
		dbRoot = filepath.Join(dbName, PreCommitPath)
	case DBTypeClaim:
		dbRoot = filepath.Join(dbName, ClaimPath)
	case DBTypeAccount:
		if address == nil || dbInfo == nil {
			fmt.Println("Can not get AccountDB without Address and DBInfo")
			os.Exit(1)
		}
		prefix := int(address.ID()[0]) % dbInfo.DBCount
		index := prefix + 1
		if dbInfo.QueryDBIsZero {
			accountDBPath := fmt.Sprintf(core.CalculateDBNameFormat, index, dbInfo.DBCount, 0)
			dbRoot = filepath.Join(dbName, accountDBPath)
		} else {
			accountDBPath := fmt.Sprintf(core.CalculateDBNameFormat, index, dbInfo.DBCount, 1)
			dbRoot = filepath.Join(dbName, accountDBPath)
		}
	default:
		fmt.Println("Invalid DB Type")
		os.Exit(1)
	}
	qdb := getDBByRootPath(dbRoot)
	return qdb
}

func getDBByRootPath(dbRoot string) db.Database {
	dir, name := filepath.Split(dbRoot)
	return db.Open(dir, string(db.GoLevelDBBackend), name)
}

func iteratePrintDB(qDB db.Database, dbType string, addr *common.Address, blockHeight uint64) {
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
}
