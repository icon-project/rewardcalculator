package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)



func (cli *CLI) calculate(conn ipc.Connection, iissData string, blockHeight uint64) {
	var req core.CalculateRequest
	var resp core.CalculateResponse

	req.Path = iissData
	req.BlockHeight = blockHeight

	conn.SendAndReceive(msgCalculate, cli.id, &req, &resp)
	fmt.Printf("CALCULATE command get response: %s\n", Display(resp))
}