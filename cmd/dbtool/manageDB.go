package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
	"os"
	"path/filepath"
)

type ManagerDB struct {
	dbPath string
}

func (manageDB ManagerDB) query() {
	if manageDB.dbPath == "" {
		fmt.Println("Enter dbPath")
		os.Exit(1)
	}
	dir, name := filepath.Split(manageDB.dbPath)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	iteratePrintDB(DBTypeManagement, qdb, nil, 0, "")
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
	default:
		fmt.Println("Invalid prefix")
		return false
	}
	fmt.Println(result)
	return true
}
