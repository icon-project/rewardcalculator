package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

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

func (cli *CLI) query(dbName string, key string) {
	fmt.Printf("Query account. DB name: %s Address: %s", dbName, key)

	lvlDB := db.Open(DBDir, DBType, dbName)
	defer lvlDB.Close()

	bucket, _ := lvlDB.GetBucket(db.PrefixGovernanceVariable)

	// TODO find RC account DB

	fmt.Printf("Get value %s for %s\n", queryData(bucket, key), key)
}
