package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
	"log"
)

func (cli *CLI) query(dbName string, key string) {
	fmt.Printf("Query account. DB name: %s Address: %s\n", dbName, key)

	ctx, err := core.NewContext(DBDir, DBType, dbName, 0, "")
	if nil != err {
		log.Printf("Failed to initialize IScore DB")
		return
	}

	resp := core.DoQuery(ctx, *common.NewAddressFromString(key))

	fmt.Printf("Get value %s for %s\n", resp.IScore.String(), key)
}
