package main

import (
	"encoding/hex"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
	"log"
)

func (cli *CLI) query(dbName string, key string, txHash []byte) {
	fmt.Printf("Query account. DB name: %s Address: %s TXHash: %s\n",
		dbName, key, hex.EncodeToString(txHash))

	ctx, err := core.NewContext(DBDir, DBType, dbName, 0, "")
	if nil != err {
		log.Printf("Failed to initialize IScore DB")
		return
	}

	req := &core.Query{
		Address: *common.NewAddressFromString(key),
		TXHash: txHash,
	}

	resp := core.DoQuery(ctx, req)

	fmt.Printf("Get value %s for %s\n", resp.IScore.String(), key)
}
