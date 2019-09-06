package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)



func (cli *CLI) query(conn ipc.Connection, address string) *core.ResponseQuery {
	var addr common.Address
	resp := new(core.ResponseQuery)

	addr.SetString(address)

	conn.SendAndReceive(core.MsgQuery, cli.id, &addr, resp)
	fmt.Printf("QUERY command get response: %s\n", resp.String())

	return resp
}