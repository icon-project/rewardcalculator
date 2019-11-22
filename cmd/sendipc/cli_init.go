package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) init(conn ipc.Connection, blockHeight uint64) {
	var resp core.ResponseInit

	conn.SendAndReceive(core.MsgINIT, cli.id, &blockHeight, &resp)
	fmt.Printf("INIT command get response: %s\n", resp.String())
}