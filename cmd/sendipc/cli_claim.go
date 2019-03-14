package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)



func (cli *CLI) claim(conn ipc.Connection, address string) {
	var addr common.Address
	var resp rewardcalculator.ResponseIScore

	addr.SetString(address)

	conn.SendAndReceive(msgClaim, &addr, &resp)
	fmt.Printf("CLAIM command get response: %s\n", Display(resp))
}