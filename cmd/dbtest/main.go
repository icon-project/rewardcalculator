package main

func main() {
	cli := CLI{}
	cli.Run()
}

	//argc :=len(os.Args)
	//if argc < 2 {
	//	usage()
	//	return
	//}
	//
	//dbName := os.Args[1]
	//
	//lvlDB := db.Open(DBDir, DBType, dbName)
	//defer lvlDB.Close()
	//
	//bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)
	//
	//start := time.Now()
	//
	//switch os.Args[2] {
	//case "create":
	//	if argc != 5 {
	//		usage()
	//		return
	//	}
	//	dbCount, err := strconv.Atoi(os.Args[3])
	//	if err != nil {
	//		fmt.Printf("Failed to convert DB count(%s) to int", os.Args[3])
	//		return
	//	}
	//	entryCount, err := strconv.Atoi(os.Args[4])
	//	if err != nil {
	//		fmt.Printf("Failed to convert entry count(%s) to int", os.Args[4])
	//		return
	//	}
	//
	//	// Create I-Score DB
	//	createDB(DBDir, dbName, dbCount, entryCount)
	//
	//	// Write I-Score DB Info. at global DB
	//	dbInfo := new(rewardcalculator.DBInfo)
	//	dbInfo.DbCount = dbCount
	//	dbInfo.AccountCount.Value = uint64(entryCount)
	//	//dbInfo.Test = 7
	//
	//	data, _ := dbInfo.Bytes()
	//	fmt.Printf("dbinfo: %s \n data: len: %d, bytes: %b\n", dbInfo.String(), len(data), data)
	//	bucket.Set(dbInfo.ID(), data)
	//
	//case "delete":
	//	path := DBDir + "/" + dbName
	//	os.RemoveAll(path)
	//	fmt.Printf("Delete %s\n", path)
	//
	//case "query":
	//	if argc != 4 {
	//		usage()
	//		return
	//	}
	//	key := os.Args[3]
	//	fmt.Printf("Get value %s for %s\n", queryData(bucket, key), key)
	//
	//case "calculate":
	//	if argc != 6 {
	//		usage()
	//		return
	//	}
	//
	//	blockHeight, err := strconv.ParseUint(os.Args[3], 10, 0)
	//	if err != nil {
	//		fmt.Printf("Block height 'TO' must be an integer > 0. (%s)", os.Args[3])
	//		return
	//	}
	//	worker, err := strconv.Atoi(os.Args[4])
	//	if err != nil || worker > 256 {
	//		fmt.Printf("Worker 'N' must be an integer < 256. (%s)", os.Args[4])
	//		return
	//	}
	//	batchCount, err := strconv.ParseUint(os.Args[5], 10, 0)
	//	if err != nil {
	//		fmt.Printf("Write batch count 'BATCH' must be an integer. (%s)", os.Args[5])
	//		return
	//	}
	//
	//	// make global options
	//	opts := new(rewardcalculator.GlobalOptions)
	//
	//	opts.BlockHeight.Value = blockHeight
	//	for i := 0 ; i < rewardcalculator.NumDelegate; i++ {
	//		daddr := common.NewAccountAddress([]byte{byte(i+1)})
	//		opts.Validators[i] = *daddr
	//	}
	//	opts.GV.RewardRep.Value = 1
	//	//fmt.Printf("Global options : %s\n", opts.String())
	//
	//	var count, entries uint64
	//	var wait sync.WaitGroup
	//
	//	wait.Add(worker)
	//	start = time.Now()
	//
	//	// run calculation
	//	for i :=0; i < worker; i++ {
	//		start, limit := getPrefix(db.PrefixIScore, i, worker)
	//
	//		go func(start []byte, limit []byte) {
	//			//runtime.LockOSThread()
	//			defer wait.Done()
	//			c, e := calculate(lvlDB, bucket, start, limit, opts, batchCount)
	//			count += c; entries += e
	//		} (start, limit)
	//	}
	//	wait.Wait()
	//	fmt.Printf("Total>block height: %d, worker: %d, batch: %d, calculation %d for %d entries\n",
	//		blockHeight, worker, batchCount, count, entries)
	//case "calculateExt":
	//	if argc != 4 {
	//		usage()
	//		return
	//	}
	//	batchCount, err := strconv.ParseUint(os.Args[3], 10, 0)
	//	if err != nil {
	//		fmt.Printf("Write batch count 'BATCH' must be an integer. (%s)", os.Args[3])
	//		return
	//	}
	//
	//	dbInfo := new(rewardcalculator.DBInfo)
	//
	//	data, _ := bucket.Get(dbInfo.ID())
	//	err = dbInfo.SetBytes(data)
	//	if err != nil {
	//		fmt.Printf("Failed to read DB Info. err=%+v\n", err)
	//		return
	//	}
	//
	//	fmt.Printf("DBInfo %s\n", dbInfo.String())
	//
	//	// make global options
	//	opts := new(rewardcalculator.GlobalOptions)
	//
	//	opts.BlockHeight.Value = dbInfo.BlockHeight.Value
	//	for i := 0 ; i < rewardcalculator.NumDelegate; i++ {
	//		daddr := common.NewAccountAddress([]byte{byte(i+1)})
	//		opts.Validators[i] = *daddr
	//	}
	//	opts.GV.RewardRep.Value = 1
	//
	//	var totalCount, totalEntry uint64
	//	var wait sync.WaitGroup
	//	wait.Add(dbInfo.DbCount)
	//
	//	dbDir := fmt.Sprintf("%s/%s", DBDir, dbName)
	//	for i:= 0; i< dbInfo.DbCount; i++ {
	//		go func(index int) {
	//			dbNameTemp := fmt.Sprintf("%d_%d", index + 1, dbInfo.DbCount)
	//			lvlDB := db.Open(dbDir, DBType, dbNameTemp)
	//			defer lvlDB.Close()
	//			defer wait.Done()
	//
	//			bucket, _ := lvlDB.GetBucket(db.PrefixIScore)
	//			c, e := calculate(lvlDB, bucket, nil, nil, opts, batchCount)
	//
	//			fmt.Printf("Calculate DB %s with %d/%d entries.\n", dbNameTemp, c, e)
	//			totalCount += c
	//			totalEntry += e
	//		} (i)
	//	}
	//	wait.Wait()
	//	fmt.Printf("Total>block height: %d, worker: %d, batch: %d, calculation %d for %d entries\n",
	//		dbInfo.BlockHeight.Value, dbInfo.DbCount, batchCount, totalCount, totalEntry)
	//
	//
	//default:
	//	usage()
	//	return
	//}
	//
	//end := time.Now()
	//
	//diff := end.Sub(start)
	//fmt.Printf("Duration : %v\n", diff)
