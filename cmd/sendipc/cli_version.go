package main

import (
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
)

func (cli *CLI) version(conn ipc.Connection) {
	var buf uint16

	conn.SendAndReceive(msgVERSION, cli.id, nil, &buf)
	fmt.Printf("VERSION command get response: %d\n", buf)
}