package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)



func (cli *CLI) calculate(conn ipc.Connection, iissData string, blockHeight uint64) {
	var req rewardcalculator.CalculateRequest
	var resp rewardcalculator.CalculateResponse

	req.Path = iissData
	req.BlockHeight = blockHeight

	conn.SendAndReceive(msgCalculate, &req, &resp)
	fmt.Printf("CALCULATE command get response: %s\n", Display(resp))
}