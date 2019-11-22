package main

import (
	"encoding/hex"
	"fmt"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) rollback(conn ipc.Connection, blockHeight uint64, blockHash string) {
	var req core.RollBackRequest
	var resp core.RollBackResponse

	hash, err := hex.DecodeString(blockHash)
	if err != nil {
		fmt.Printf("Failed to decode blockHash. %v\n", err)
		return
	}

	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, core.BlockHashSize)
	copy(req.BlockHash, hash[0:core.BlockHashSize])

	conn.SendAndReceive(core.MsgRollBack, cli.id, &req, &resp)
	fmt.Printf("ROLLBACK command get response: %s\n", resp.String())
}