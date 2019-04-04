package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

func (cli *CLI) version(conn ipc.Connection) {
	var buf rewardcalculator.VersionMessage

	conn.SendAndReceive(msgVERSION, cli.id, nil, &buf)
	fmt.Printf("VERSION command get response: %s\n", Display(buf))
}