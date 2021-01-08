package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)



func (cli *CLI) query(conn ipc.Connection, address string, txHash []byte) *core.ResponseQuery {
	req := &core.Query{
		Address: *common.NewAddressFromString(address),
		TXHash: txHash,
	}
	resp := new(core.ResponseQuery)

	conn.SendAndReceive(core.MsgQuery, cli.id, req, resp)
	fmt.Printf("QUERY command get response: %s\n", resp.String())

	return resp
}