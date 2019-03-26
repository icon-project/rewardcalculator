package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)


//func makePrefix(id db.BucketID, value uint8, last bool) []byte {
//	buf := make([]byte, len(id) + 1)
//	copy(buf, id)
//	if last {
//		buf[len(id)-1]++
//	} else {
//		buf[len(id)] = value
//	}
//
//	return buf
//}
//
//func getPrefix(id db.BucketID, index int, worker int) ([]byte, []byte) {
//	if worker == 1 {
//		return nil, nil
//	}
//
//	unit := uint8(256 / worker)
//	start := makePrefix(id, unit * uint8(index), false)
//	limit := makePrefix(id, unit * uint8(index + 1), index == worker - 1)
//
//	return start, limit
//}

func queryData(bucket db.Bucket, key string) string {
	addr := common.NewAddressFromString(key)
	result, _ := bucket.Get(addr.Bytes())
	ia, err := rewardcalculator.NewIScoreAccountFromBytes(result)
	if err != nil {
		return "NODATA"
	}
	ia.Address = *addr

	return ia.String()
}

func (cli *CLI) query(dbName string, key string) {
	fmt.Printf("Query account. DB name: %s Address: %s", dbName, key)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)

	// TODO find RC account DB

	fmt.Printf("Get value %s for %s\n", queryData(bucket, key), key)
}
