package main

import (
	"encoding/binary"
	"fmt"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)



func (cli *CLI) claim(conn ipc.Connection, address string, blockHeight uint64) {
	var req core.ClaimMessage
	var resp core.ResponseClaim

	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, 8)
	binary.BigEndian.PutUint64(req.BlockHash, blockHeight)

	conn.SendAndReceive(msgClaim, cli.id, &req, &resp)
	cli.id++
	fmt.Printf("CLAIM command get response: %s\n", Display(resp))

	var commit core.CommitBlock
	var commitResp core.CommitBlock

	commit.Success = true
	commit.BlockHash = req.BlockHash
	commit.BlockHeight = blockHeight

	fmt.Printf("Send COMMIT_BLOCK message: %s\n", Display(commit))
	conn.SendAndReceive(msgCommitBlock, cli.id, &commit, &commitResp)
	fmt.Printf("COMMIT_BLOCK message get response: %s\n", Display(commitResp))
}