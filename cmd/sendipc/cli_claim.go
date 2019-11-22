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
	req.BlockHash = make([]byte, core.BlockHashSize)
	binary.BigEndian.PutUint64(req.BlockHash, blockHeight)

	fmt.Printf("send CLAIM message: %s\n", req.String())
	conn.SendAndReceive(core.MsgClaim, cli.id, &req, &resp)
	cli.id++
	fmt.Printf("CLAIM message get response: %s\n", resp.String())

	// send COMMIT_CLAIM and get ack
	var commitClaim core.CommitClaim
	commitClaim.Success = true
	commitClaim.Address = req.Address
	commitClaim.BlockHeight = req.BlockHeight
	commitClaim.BlockHash = make([]byte, core.BlockHashSize)
	copy(commitClaim.BlockHash, req.BlockHash)

	fmt.Printf("Send COMMIT_CLAIM message: %s\n", commitClaim.String())
	conn.SendAndReceive(core.MsgCommitClaim, cli.id, &commitClaim, &resp)
	cli.id++
	fmt.Printf("COMMIT_CLAIM message get ack\n")

	// send COMMIT_BLOCK and get response
	var commit core.CommitBlock
	var commitResp core.CommitBlock
	commit.Success = true
	commit.BlockHash = req.BlockHash
	commit.BlockHeight = blockHeight

	fmt.Printf("Send COMMIT_BLOCK message: %s\n", commit.String())
	conn.SendAndReceive(core.MsgCommitBlock, cli.id, &commit, &commitResp)
	fmt.Printf("COMMIT_BLOCK message get response: %s\n", commitResp.String())
}