package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

const (
	DBDir     = "/Users/eunsoopark/test/rc_test"
	DBType    = "goleveldb"
	DBName    = "test"
)

func createAddress(prefix []byte) (*common.Address, error) {
	data := make([]byte, common.AddressIDBytes - len(prefix))
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	buf := make([]byte, common.AddressIDBytes)
	copy(buf, prefix)
	copy(buf[len(prefix):], data)

	addr := common.NewAccountAddress(buf)
	//fmt.Printf("Created an address : %s", addr.String())

	return addr, nil
}

func createIScoreData(prefix []byte) *rewardcalculator.IScoreAccount {
	addr, err := createAddress(prefix)
	if err != nil {
		fmt.Printf("Failed to create Address err=%+v\n", err)
		return nil
	}

	ia := new(rewardcalculator.IScoreAccount)

	stake := rand.Uint64()
	delegate := stake / rewardcalculator.NumDelegate

	ia.Stake.SetUint64(stake)
	for i := 0; i < rewardcalculator.NumDelegate; i++ {
		var daddr *common.Address

		daddr = common.NewAccountAddress([]byte{byte(i+1)})
		ia.Delegations[i].Address = *daddr
		ia.Delegations[i].Delegate.SetUint64(delegate)
	}
	ia.Address = *addr

	//fmt.Printf("Result: %s", ia.String())

	return ia
}

func createData(bucket db.Bucket, prefix []byte, count int) int {
	// Governance Variable

	// PRep list

	// Account
	for i := 0; i < count; i++ {
		data := createIScoreData(prefix)
		if data == nil {
			return i
		}

		key := data.ID()
		value, _ := data.Bytes()
		//fmt.Printf("size of data: %d\n", len(value))

		bucket.Set(key, value)
	}


	return count
}

func createDB(dbDir string, dbName string, dbCount int, totalEntryCount int) {
	dbDir = fmt.Sprintf("%s/%s", dbDir, dbName)
	os.MkdirAll(dbDir, os.ModePerm)

	dbEntryCount := totalEntryCount / dbCount
	totalCount := 0

	var wait sync.WaitGroup
	wait.Add(dbCount)

	for i := 0; i < dbCount; i++ {
		go func(index int) {
			dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbCount)
			lvlDB := db.Open(dbDir, DBType, dbNameTemp)
			defer lvlDB.Close()
			defer wait.Done()

			bucket, _ := lvlDB.GetBucket(rewardcalculator.PrefixIScore)
			count := createData(bucket, []byte(strconv.FormatInt(int64(index), 16)), dbEntryCount)

			fmt.Printf("Create DB %s with %d entries.\n", dbNameTemp, count)
			totalCount += count
		} (i)
	}
	wait.Wait()

	fmt.Printf("Create %d DBs with total %d/%d entries.\n", dbCount, totalCount, totalEntryCount)
}

func queryData(bucket db.Bucket, key string) string {
	addr := common.NewAddressFromString(key)
	result, _ := bucket.Get(addr.ID())
	ia, err := rewardcalculator.NewIScoreAccountFromBytes(result)
	if err != nil {
		return "NODATA"
	}
	ia.Address = *addr

	return ia.String()
}

func calculateIScore(ia *rewardcalculator.IScoreData, opts *rewardcalculator.GlobalOptions) bool {
	// IScore = old + period * G.V * sum(valid delegations)
	if opts.BlockHeight.Value == 0 {
		opts.BlockHeight.Value = ia.BlockHeight.Value + 1
	}
	period := opts.BlockHeight.Value - ia.BlockHeight.Value
	gv := opts.GV.RewardRep.Value
	if period == 0 || gv == 0 {
		return false
	}

	multiplier := big.NewInt(int64(period * gv))

	var delegations common.HexInt
	for i := 0; i < rewardcalculator.NumDelegate; i++ {
		for j := 0; j < rewardcalculator.NumPRep; j++ {
			if ia.Delegations[i].Address.Equal(&opts.Validators[j]) {
				delegations.Add(&delegations.Int, &ia.Delegations[i].Delegate.Int)
				continue
			}
		}
	}
	if delegations.Int.Sign() == 0 {
		// there is no delegations
		return false
	}
	delegations.Int.Mul(&delegations.Int, multiplier)

	//fmt.Printf("period: %U, gv: %U, multiplier: %s, delegations: %s",
	//	period, gv, multiplier.String(), delegations.Int.String())

	// increase value
	ia.IScore.Add(&ia.IScore.Int, &delegations.Int)

	// BlockHeight
	ia.BlockHeight.Value = opts.BlockHeight.Value

	return true
}

func calculate(db db.Database, bucket db.Bucket, start []byte, limit []byte,
	opts *rewardcalculator.GlobalOptions, batchCount uint64) (count uint64, entries uint64) {
	iter, _ := db.GetIterator()
	batch, _ := db.GetBatch()
	entries = 0; count = 0

	batch.New()
	iter.New(start, limit)
	for entries = 0; iter.Next(); entries++ {
		// read
		key := iter.Key()[len(rewardcalculator.PrefixIScore):]
		ia, err := rewardcalculator.NewIScoreAccountFromBytes(iter.Value())
		if err != nil {
			fmt.Printf("Can't read data with iterator\n")
			return 0, 0
		}

		//fmt.Printf("Read data: %s\n", ia.String())

		// calculate
		if calculateIScore(&ia.IScoreData, opts) == false {
			continue
		}

		//fmt.Printf("Updated data: %s\n", ia.String())

		value, _ := ia.Bytes()

		if batchCount > 0 {
			batch.Set(iter.Key(), value)

			// write batch to DB
			if entries != 0 && (entries % batchCount) == 0 {
				err = batch.Write()
				if err != nil {
					fmt.Printf("Failed to write batch\n")
				}
				batch.Reset()
			}
		} else {
			bucket.Set(key, value)
		}

		count++
	}

	// write batch to DB
	if batchCount > 0 {
		err := batch.Write()
		if err != nil {
			fmt.Printf("Failed to write batch\n")
		}
		batch.Reset()
	}

	// finalize iterator
	iter.Release()
	err := iter.Error()
	if err != nil {
		fmt.Printf("There is error while iteration. %+v", err)
	}

	fmt.Printf("Calculate %d entries for prefix %v-%v %d entries\n", count, start, limit, entries)

	return count, entries
}

func makePrefix(id db.BucketID, value uint8, last bool) []byte {
	buf := make([]byte, len(id) + 1)
	copy(buf, id)
	if last {
		buf[len(id)-1]++
	} else {
		buf[len(id)] = value
	}

	return buf
}

func getPrefix(id db.BucketID, index int, worker int) ([]byte, []byte) {
	if worker == 1 {
		return nil, nil
	}

	unit := uint8(256 / worker)
	start := makePrefix(id, unit * uint8(index), false)
	limit := makePrefix(id, unit * uint8(index + 1), index == worker - 1)

	return start, limit
}

func usage() {
	fmt.Printf("Usage: %s [db_name] [command]\n\n commands\n", os.Args[0])
	fmt.Printf("\t create N NUM                 Create N DBs with NUM accounts\n")
	fmt.Printf("\t delete                       Delete DB\n")
	fmt.Printf("\t query KEY                    Query accounts value with KEY\n")
	fmt.Printf("\t calculate TO WORKE BATCH     Calculate I-Score of all account\n")
	fmt.Printf("\t           TO                 Block height to calculate. Set 0 if you want current block+1\n")
	fmt.Printf("\t           WORKER             The number of thread workers\n")
	fmt.Printf("\t           BATCH              The number of DB write batch count\n")
	fmt.Printf("\t calculateExt BATCH           Calculate I-Score of all account at seperated DBs\n")
	fmt.Printf("\t           BATCH              The number of DB write batch count\n")
}
func main() {
	argc :=len(os.Args)
	if argc < 2 {
		usage()
		return
	}

	dbName := os.Args[1]

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(rewardcalculator.PrefixGovernanceVariable)

	start := time.Now()

	switch os.Args[2] {
	case "create":
		if argc != 5 {
			usage()
			return
		}
		dbCount, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Printf("Failed to convert DB count(%s) to int", os.Args[3])
			return
		}
		entryCount, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Printf("Failed to convert entry count(%s) to int", os.Args[4])
			return
		}

		// Create I-Score DB
		createDB(DBDir, dbName, dbCount, entryCount)

		// Write I-Score DB Info. at global DB
		dbInfo := new(rewardcalculator.DBInfo)
		dbInfo.DbCount = dbCount
		dbInfo.EntryCount = entryCount

		data, _ := dbInfo.Bytes()
		bucket.Set(dbInfo.ID(), data)

	case "delete":
		path := DBDir + "/" + dbName
		os.RemoveAll(path)
		fmt.Printf("Delete %s\n", path)

	case "query":
		if argc != 4 {
			usage()
			return
		}
		key := os.Args[3]
		fmt.Printf("Get value %s for %s\n", queryData(bucket, key), key)

	case "calculate":
		if argc != 6 {
			usage()
			return
		}

		blockHeight, err := strconv.ParseUint(os.Args[3], 10, 0)
		if err != nil {
			fmt.Printf("Block height 'TO' must be an integer > 0. (%s)", os.Args[3])
			return
		}
		worker, err := strconv.Atoi(os.Args[4])
		if err != nil || worker > 256 {
			fmt.Printf("Worker 'N' must be an integer < 256. (%s)", os.Args[4])
			return
		}
		batchCount, err := strconv.ParseUint(os.Args[5], 10, 0)
		if err != nil {
			fmt.Printf("Write batch count 'BATCH' must be an integer. (%s)", os.Args[5])
			return
		}

		// make global options
		opts := new(rewardcalculator.GlobalOptions)

		opts.BlockHeight.Value = blockHeight
		for i := 0 ; i < rewardcalculator.NumDelegate; i++ {
			daddr := common.NewAccountAddress([]byte{byte(i+1)})
			opts.Validators[i] = *daddr
		}
		opts.GV.RewardRep.Value = 1
		//fmt.Printf("Global options : %s\n", opts.String())

		var count, entries uint64
		var wait sync.WaitGroup

		wait.Add(worker)
		start = time.Now()

		// run calculation
		for i :=0; i < worker; i++ {
			start, limit := getPrefix(rewardcalculator.PrefixIScore, i, worker)

			go func(start []byte, limit []byte) {
				//runtime.LockOSThread()
				defer wait.Done()
				c, e := calculate(lvlDB, bucket, start, limit, opts, batchCount)
				count += c; entries += e
			} (start, limit)
		}
		wait.Wait()
		fmt.Printf("Total>block height: %d, worker: %d, batch: %d, calculation %d for %d entries\n",
			blockHeight, worker, batchCount, count, entries)
	case "calculateExt":
		if argc != 4 {
			usage()
			return
		}
		batchCount, err := strconv.ParseUint(os.Args[3], 10, 0)
		if err != nil {
			fmt.Printf("Write batch count 'BATCH' must be an integer. (%s)", os.Args[3])
			return
		}

		dbInfo := new(rewardcalculator.DBInfo)

		data, _ := bucket.Get(dbInfo.ID())
		err = dbInfo.SetBytes(data)
		if err != nil {
			fmt.Printf("Failed to read DB Info. err=%+v\n", err)
			return
		}

		fmt.Printf("DBInfo %s\n", dbInfo.String())

		// make global options
		opts := new(rewardcalculator.GlobalOptions)

		opts.BlockHeight.Value = dbInfo.BlockHeight.Value
		for i := 0 ; i < rewardcalculator.NumDelegate; i++ {
			daddr := common.NewAccountAddress([]byte{byte(i+1)})
			opts.Validators[i] = *daddr
		}
		opts.GV.RewardRep.Value = 1

		var totalCount, totalEntry uint64
		var wait sync.WaitGroup
		wait.Add(dbInfo.DbCount)

		dbDir := fmt.Sprintf("%s/%s", DBDir, dbName)
		for i:= 0; i< dbInfo.DbCount; i++ {
			go func(index int) {
				dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbInfo.DbCount)
				lvlDB := db.Open(dbDir, DBType, dbNameTemp)
				defer lvlDB.Close()
				defer wait.Done()

				bucket, _ := lvlDB.GetBucket(rewardcalculator.PrefixIScore)
				c, e := calculate(lvlDB, bucket, nil, nil, opts, batchCount)

				fmt.Printf("Calculate DB %s with %d/%d entries.\n", dbNameTemp, c, e)
				totalCount += c
				totalEntry += e
			} (i)
		}
		wait.Wait()
		fmt.Printf("Total>block height: %d, worker: %d, batch: %d, calculation %d for %d entries\n",
			dbInfo.BlockHeight.Value, dbInfo.DbCount, batchCount, totalCount, totalEntry)


	default:
		usage()
		return
	}

	end := time.Now()

	diff := end.Sub(start)
	fmt.Printf("Duration : %v\n", diff)
}
