package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) calculate(conn ipc.Connection, iissData string, blockHeight uint64) {
	var req core.CalculateRequest
	var resp core.CalculateDone

	req.Path = iissData
	req.BlockHeight = blockHeight

	// Send CALCULATE and get ack
	conn.SendAndReceive(core.MsgCalculate, cli.id, &req, &resp)

	// Get CALCULATE_DONE
	msg, id, _ := conn.Receive(&resp)
	if msg == core.MsgReady {
		fmt.Printf("CALCULATE command get calculate result: %s\n", Display(resp))
	} else {
		fmt.Printf("CALCULATE command get invalied response : (msg:%d, id:%d)\n", msg, id)
	}

}

func (cli *CLI) queryCalculateStatus(conn ipc.Connection) {
	var resp core.QueryCalculateStatusResponse

	// Send QUERY_CALCULATE_STATUS and get response
	conn.SendAndReceive(core.MsgQueryCalculateStatus, cli.id, nil, &resp)

	fmt.Printf("QUERY_CALCULATE_STATUS command get response: %s\n", Display(resp))
}
