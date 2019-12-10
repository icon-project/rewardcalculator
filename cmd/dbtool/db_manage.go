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

func queryManagementDB(input Input) (err error) {
	if input.path == "" {
		fmt.Println("Enter dbPath")
		return errors.New("invalid db ")
	}
	dir, name := filepath.Split(input.path)
	qdb := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer qdb.Close()

	switch input.data {
	case "":
		fmt.Println("============== Database info ==============")
		if err = queryDBInfo(qdb); err != nil {
			return
		}
		fmt.Println("\n============== Governance variables ==============")
		if err = iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)), printGV); err != nil {
			return
		}
		fmt.Println("\n============== P-Rep ==============")
		if err = queryPRep(qdb, 0); err != nil {
			return
		}
		fmt.Println("\n============== P-Rep Candidates ==============")
		if err = queryPC(qdb, ""); err != nil {
			return
		}
	case DataTypeDI:
		if err = queryDBInfo(qdb); err != nil {
			return
		}
	case DataTypeGV:
		if err = iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixGovernanceVariable)), printGV); err != nil {
			return
		}
	case DataTypePRep:
		if err = queryPRep(qdb, input.height); err != nil {
			return
		}
	case DataTypePC:
		if err = queryPC(qdb, input.address); err != nil {
			return
		}
	default:
		return errors.New("invalid data type")
	}
	return nil
}

func queryPC(qdb db.Database, address string) error {
	if address == "" {
		err := iteratePrintDB(qdb, util.BytesPrefix([]byte(db.PrefixPRepCandidate)), printPC)
		return err
	} else {
		addr := common.NewAddressFromString(address)
		if pc, err := getPC(qdb, addr); err != nil {
			return err
		} else {
			printPRepCandidate(pc)
		}
	}
	return nil
}

func queryDBInfo(qdb db.Database) error {
	bucket, err := qdb.GetBucket(db.PrefixManagement)
	if err != nil {
		fmt.Println("error while getting database info bucket")
		return err
	}
	dbInfo := new(core.DBInfo)
	value, e := bucket.Get(dbInfo.ID())
	if e != nil {
		fmt.Println("error while Get value of Database info")
		return e
	}
	if err = dbInfo.SetBytes(value); err != nil {
	} else {
		fmt.Println(dbInfo.String())
	}
	return nil
}

func getPC(qdb db.Database, address *common.Address) (*core.PRepCandidate, error) {
	bucket, err := qdb.GetBucket(db.PrefixPRepCandidate)
	if err != nil {
		fmt.Println("error while getting prep candidate bucket")
		return nil, err
	}
	value, e := bucket.Get(address.Bytes())
	if e != nil {
		fmt.Println("error while Get value of prep candidate")
		return nil, e
	}
	pcPrefixLen := len(db.PrefixPRepCandidate)
	qKey := make([]byte, pcPrefixLen+common.AddressBytes)
	copy(qKey, db.PrefixPRepCandidate)
	copy(qKey[pcPrefixLen:], address.Bytes())
	pc, eValue := newPC(qKey, value)
	return pc, eValue

}

func printGV(key []byte, value []byte) error {
	gv := new(core.GovernanceVariable)
	if err := gv.SetBytes(value); err != nil {
		fmt.Printf("Error while initialize GovernanceVariable")
		return err
	}
	gv.BlockHeight = common.BytesToUint64(key[len(db.PrefixGovernanceVariable):])
	fmt.Println(gv.String())
	return nil
}

func printPC(key []byte, value []byte) error {
	if pc, err := newPC(key, value); err != nil {
		fmt.Printf("failed to initialize PRepCandidate")
		return err
	} else {
		printPRepCandidate(pc)
		return err
	}
}

func printPRepCandidate(pc *core.PRepCandidate) {
	if pc != nil {
		fmt.Printf("%s\n", pc.String())
	}
}

func newPC(key []byte, value []byte) (pc *core.PRepCandidate, err error) {
	pc = new(core.PRepCandidate)
	if err = pc.SetBytes(value); err != nil {
		fmt.Printf("Error while initilize PrepCandidate")
		return nil, err
	}
	pc.Address = *common.NewAddress(key[len(db.PrefixPRepCandidate):])
	return
}
