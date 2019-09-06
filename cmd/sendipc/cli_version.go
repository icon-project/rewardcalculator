package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) version(conn ipc.Connection) {
	var buf uint16

	conn.SendAndReceive(core.MsgVersion, cli.id, nil, &buf)
	fmt.Printf("VERSION command get response: %d\n", buf)
}