package main

import (
	"log"

	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) calculate(dbName string, blockHeight uint64, batchCount uint64) {
	log.Printf("Start calculate DB. name: %s, block height: %d, batch count: %d\n", dbName, blockHeight, batchCount)

	ctx, err := core.NewContext(DBDir, DBType, dbName, 0, "")
	if nil != err {
		log.Printf("Failed to initialize IScore DB")
		return
	}

	ctx.Print()

	var req core.CalculateRequest
	req.BlockHeight = blockHeight
	req.Path = "noiissdata"

	core.DoCalculate(ctx.CancelCalculation.GetChannel(), ctx, &req, nil, 0)
}
