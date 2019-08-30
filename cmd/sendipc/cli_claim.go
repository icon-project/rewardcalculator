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

	// Send CLAIM and get response
	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, 8)
	binary.BigEndian.PutUint64(req.BlockHash, blockHeight)

	conn.SendAndReceive(core.MsgClaim, cli.id, &req, &resp)
	cli.id++
	fmt.Printf("CLAIM command get response: %s\n", Display(resp))

	// send COMMIT_CLAIM and get ack
	var commitClaim core.CommitClaim
	commitClaim.Address = req.Address
	commitClaim.BlockHeight = req.BlockHeight
	commitClaim.BlockHash = req.BlockHash
	commitClaim.Success = true

	fmt.Printf("Send COMMIT_CLAIM message: %s\n", Display(commitClaim))
	conn.SendAndReceive(core.MsgClaim, cli.id, &req, &resp)
	cli.id++
	fmt.Printf("COMMIT_CLAIM message get ack\n")

	// send COMMIT_BLOCK and get response
	var commit core.CommitBlock
	var commitResp core.CommitBlock
	commit.Success = true
	commit.BlockHash = req.BlockHash
	commit.BlockHeight = blockHeight

	fmt.Printf("Send COMMIT_BLOCK message: %s\n", Display(commit))
	conn.SendAndReceive(core.MsgCommitBlock, cli.id, &commit, &commitResp)
	fmt.Printf("COMMIT_BLOCK message get response: %s\n", Display(commitResp))
}