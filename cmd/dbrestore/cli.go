package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
)

type CLI struct{
	cmd *flag.FlagSet
}

const (
	IISSDBPath = "iiss/current_db"
	RCDBPath = "rc"
)

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [score_DB_path] [command] [[options]] \n", os.Args[0])
	fmt.Printf("score_DB_path    SCORE DB path to check and restore\n")
	fmt.Printf("command          Command to run\n")
	fmt.Printf("\t check                  Check validation of IISS data DB\n")
	fmt.Printf("\t restore                Restore SCORE DB\n")
	fmt.Printf("\t get                    Get hash value from IISS data DB\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() bool {
	cli.validateArgs()

	checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
	checkValue := checkCmd.String("value", "", "IISS data DB CALCULATION_DONE value (required)")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restoreValue := restoreCmd.String("value", "", "IISS data DB CALCULATION_DONE value (required)")
	restoreValidDB := restoreCmd.String("valid-db", "", "Valid DB (required)")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)

	dbRootPath := os.Args[1]
	cmd := os.Args[2]

	switch cmd {
	case "check":
		err := checkCmd.Parse(os.Args[3:])
		if err != nil {
			checkCmd.Usage()
			os.Exit(1)
		}
	case "restore":
		err := restoreCmd.Parse(os.Args[3:])
		if err != nil {
			restoreCmd.Usage()
			os.Exit(1)
		}
	case "get":
		err := getCmd.Parse(os.Args[3:])
		if err != nil {
			getCmd.Usage()
			os.Exit(1)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	ret := true

	if getCmd.Parsed() {
		ret = getIISSDataDBHash(dbRootPath)
	}

	if checkCmd.Parsed() {
		if *checkValue == "" {
			checkCmd.Usage()
			os.Exit(1)
		}
		// check IISS data DB
		ret = checkIISSDataDB(dbRootPath, *checkValue)
	}

	if restoreCmd.Parsed() {
		if *restoreValue == "" || *restoreValidDB == "" {
			restoreCmd.Usage()
			os.Exit(1)
		}

		// restore IISS data DB
		restore := restoreIISSDataDB(dbRootPath, *restoreValue)

		// restore SCORE DB
		if restore {
			restoreSCOREDB(dbRootPath, *restoreValidDB)
		}
	}

	return ret
}

func checkDBPath(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	if stat.IsDir() != true {
		return false
	}

	return true
}

type getResult struct {
	BlockHeight uint64
	HashValue   string
}

func getIISSDataDBHash(dbRootPath string) bool {
	dbName := path.Join(dbRootPath, IISSDBPath)
	if checkDBPath(dbName) != true {
		fmt.Printf("Can't find IISS data DB. %s\n", dbName)
		return false
	}

	dir, name := filepath.Split(dbName)

	key := []byte("calc_response_from_rc")

	iissDB := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer iissDB.Close()

	bucket, _ := iissDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(key)
	if bs != nil {
		result := getResult{
			BlockHeight: getBlockHeightFromIISSValue(bs),
			HashValue:   hex.EncodeToString(bs),
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Failed to get result. %v\n", err)
		}
		fmt.Printf("%s\n", string(b))
	} else {
		fmt.Printf("Invalid IISS data DB. Can't get hash\n")
	}

	return true
}

const (
	iScoreLenIndex = 3

	typeUint8  = 204
	typeUint16 = 205
	typeUint32 = 206
	typeUint64 = 207
)
func getBlockHeightFromIISSValue(value []byte) uint64 {
	blockHeightIndex := iScoreLenIndex + 1 + 1 + value[iScoreLenIndex]
	blockHeightType := value[blockHeightIndex]

	blockHeightValueIndex := blockHeightIndex + 1
	var blockHeight uint64
	switch blockHeightType {
	case typeUint8:
		blockHeight = uint64(value[blockHeightValueIndex])
	case typeUint16:
		blockHeight = uint64(binary.BigEndian.Uint16(value[blockHeightValueIndex:blockHeightValueIndex+2]))
	case typeUint32:
		blockHeight = uint64(binary.BigEndian.Uint32(value[blockHeightValueIndex:blockHeightValueIndex+4]))
	case typeUint64:
		blockHeight = binary.BigEndian.Uint64(value[blockHeightValueIndex:blockHeightValueIndex+8])
	}

	return blockHeight
}

func checkIISSDataDB(dbRootPath string, calcDoneValue string) bool {
	dbName := path.Join(dbRootPath, IISSDBPath)
	if checkDBPath(dbName) != true {
		fmt.Printf("Can't find IISS data DB. %s\n", dbName)
		return false
	}

	dir, name := filepath.Split(dbName)

	key := []byte("calc_response_from_rc")
	value, err := hex.DecodeString(calcDoneValue)
	if err != nil {
		fmt.Printf("Failed to convert valid IISS data DB value. %v\n", err)
		return false
	}

	inputBlockHeight := getBlockHeightFromIISSValue(value)

	iissDB := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer iissDB.Close()

	bucket, _ := iissDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(key)
	if bs != nil {
		iissBlockHeight := getBlockHeightFromIISSValue(bs)
		if inputBlockHeight != iissBlockHeight {
			fmt.Printf("Invalid value. block height mismatch. IISS data DB %d vs Input %d\n",
				iissBlockHeight, inputBlockHeight)
			return true
		}
		if bytes.Compare(value, bs) == 0 {
			fmt.Printf("Valid DB\n")
			return true
		}
		fmt.Printf("Value in DB : %s\n", hex.EncodeToString(bs))
		fmt.Printf("Input value : %s\n", hex.EncodeToString(value))
	}

	fmt.Printf("Invalid SCORE DB. Need to restore.\n")

	return false
}

func restoreIISSDataDB(dbRootPath string, calcDoneValue string) bool {
	dbName := path.Join(dbRootPath, IISSDBPath)
	if checkDBPath(dbName) != true {
		fmt.Printf("Can't find IISS data DB. %s\n", dbName)
		return false
	}

	dir, name := filepath.Split(dbName)

	key := []byte("calc_response_from_rc")
	value, err := hex.DecodeString(calcDoneValue)
	if err != nil {
		fmt.Printf("Failed to convert valid IISS data DB value. %v\n", err)
	}
	inputBlockHeight := getBlockHeightFromIISSValue(value)

	iissDB := db.Open(dir, string(db.GoLevelDBBackend), name)
	defer iissDB.Close()

	bucket, _ := iissDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(key)
	if bs != nil {
		//fmt.Printf("Original values : %s\n%v\n", hex.EncodeToString(bs), bs)
		if bytes.Compare(value, bs) == 0 {
			fmt.Printf("No need to restore DB\n")
			return true
		}

		iissBlockHeight := getBlockHeightFromIISSValue(bs)
		if inputBlockHeight != iissBlockHeight {
			fmt.Printf("Invalid value. block height mismatch. IISS data DB %d vs Input %d\n",
				iissBlockHeight, inputBlockHeight)
			return false
		}
	}

	err = bucket.Set(key, value)
	if err != nil {
		fmt.Printf("Failed to restore IISS data DB. %s\n", dbName)
		return false
	}

	// write valid value to IISS data DB
	fmt.Printf("Fixup IISS data DB: %s -> %s\n", hex.EncodeToString(bs), hex.EncodeToString(value))

	return true
}

func restoreSCOREDB(dbRootPath string, validDBPath string) bool {
	validDB, err := filepath.Abs(validDBPath)
	if err != nil {
		fmt.Printf("Check valid DB path %s. %v", validDBPath, err)
		return false
	}
	if checkDBPath(validDB) != true {
		fmt.Printf("Can't find valid valid DB path. %s\n", validDB)
		return false
	}

	dbName := path.Join(dbRootPath, RCDBPath)
	if checkDBPath(dbName) != true {
		fmt.Printf("Can't find SCORE data DB to fix. %s\n", dbName)
		return false
	}


	// backup original RC DB
	fmt.Printf("Backup DB. %s -> %s\n", dbName, dbName + "_bak")
	err = os.Rename(dbName, dbName + "_bak")
	if err != nil || checkDBPath(dbName) == true {
		fmt.Printf("Failed to backup. %v\n", err)
		return false
	}

	// overwrite valid RC DB
	fmt.Printf("Fixup SCORE DB. %s -> %s\n", validDB, dbName)
	os.Rename(validDB, dbName)
	if err != nil || checkDBPath(validDB) == true || checkDBPath(dbName) == false {
		fmt.Printf("Failed to fixup. %v\n", err)

		// restore backup
		os.Rename(dbName + "_bak", dbName)
		return false
	}

	// restore node specific DB
	backupDir := []string {
		"IScore/claim",
		"IScore/preCommit",
	}
	for _, rs := range backupDir {
		src := filepath.Join(dbName + "_bak", rs)
		dst := filepath.Join(dbName, rs)
		fmt.Printf("Restore node specific data. %s -> %s\n", src, dst)
		os.RemoveAll(dst)
		err = os.Rename(src, dst)
	}

	// remove backup
	os.RemoveAll(dbName + "_bak")

	return true
}
