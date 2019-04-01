package main

import (
	"encoding/binary"
	"fmt"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
)



func (cli *CLI) claim(conn ipc.Connection, address string, blockHeight uint64) {
	var req rewardcalculator.ClaimMessage
	var resp rewardcalculator.ResponseClaim

	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, 8)
	binary.BigEndian.PutUint64(req.BlockHash, blockHeight)

	conn.SendAndReceive(msgClaim, &req, &resp)
	fmt.Printf("CLAIM command get response: %s\n", Display(resp))

	var commit rewardcalculator.CommitBlock
	var commitResp rewardcalculator.CommitBlock

	commit.Success = true
	commit.BlockHash = req.BlockHash
	commit.BlockHeight = blockHeight

	fmt.Printf("Send COMMIT_BLOCK message: %s\n", Display(commit))
	conn.SendAndReceive(msgCommitBlock, &commit, &commitResp)
	fmt.Printf("COMMIT_BLOCK message get response: %s\n", Display(commitResp))
}