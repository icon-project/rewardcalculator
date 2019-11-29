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

	// Send CALCULATE and get response
	conn.SendAndReceive(core.MsgCalculate, cli.id, &req, &resp)
	fmt.Printf("CALCULATE command get response: %s\n", resp.String())
	if resp.Status != core.CalcRespStatusOK {
		return
	}

	// Get CALCULATE_DONE
	var respDone core.CalculateDone
	msg, id, _ := conn.Receive(&respDone)
	if msg == core.MsgCalculateDone {
		fmt.Printf("CALCULATE command get calculate result: %s\n", respDone.String())
	} else {
		fmt.Printf("CALCULATE command get invalid response : (msg:%d, id:%d)\n", msg, id)
	}

}

func (cli *CLI) queryCalculateStatus(conn ipc.Connection) {
	var resp core.QueryCalculateStatusResponse

	// Send QUERY_CALCULATE_STATUS and get response
	conn.SendAndReceive(core.MsgQueryCalculateStatus, cli.id, nil, &resp)

	fmt.Printf("QUERY_CALCULATE_STATUS command get response: %s\n", resp.String())
}

func (cli *CLI) queryCalculateResult(conn ipc.Connection, blockHeight uint64) {
	var resp core.QueryCalculateResultResponse

	// Send QUERY_CALCULATE_RESULT and get response
	conn.SendAndReceive(core.MsgQueryCalculateResult, cli.id, &blockHeight, &resp)

	fmt.Printf("QUERY_CALCULATE_RESULT command get response: %s\n", resp.String())
}
