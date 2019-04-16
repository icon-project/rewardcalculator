package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
)

func (cli *CLI) query(dbName string, key string) {
	fmt.Printf("Query account. DB name: %s Address: %s\n", dbName, key)

	ctx, err := rewardcalculator.NewContext(DBDir, DBType, dbName, 0)
	if nil != err {
		log.Printf("Failed to initialize IScore DB")
		return
	}

	resp := rewardcalculator.DoQuery(ctx, *common.NewAddressFromString(key))

	fmt.Printf("Get value %s for %s\n", resp.IScore.String(), key)
}
