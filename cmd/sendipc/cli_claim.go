package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)



func (cli *CLI) claim(conn ipc.Connection, address string, blockHeight uint64, blockHash string,
	txIndex uint64, txHash string, noCommitClaim bool, noCommitBlock bool) {
	var req core.ClaimMessage
	var resp core.ResponseClaim

	// Send CLAIM and get response
	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, core.BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			fmt.Printf("Failed to send CLAIM. Invalid block hash. %v\n", err)
			return
		}
		copy(req.BlockHash, bh)
	}
	req.TXIndex = txIndex
	req.TXHash = make([]byte, core.TXHashSize)
	if len(txHash) == 0 {
		binary.BigEndian.PutUint64(req.TXHash, blockHeight + txIndex)
	} else {
		th, err := hex.DecodeString(txHash)
		if err != nil {
			fmt.Printf("Failed to send CLAIM. Invalid TX hash. %v\n", err)
		}
		copy(req.TXHash, th)
	}

	fmt.Printf("Send CLAIM message: %s\n", req.String())
	conn.SendAndReceive(core.MsgClaim, cli.id, &req, &resp)
	cli.id++
	fmt.Printf("Get CLAIM response: %s\n", resp.String())

	// send COMMIT_CLAIM and get ack
	if noCommitClaim == false {
		cli.commitClaim(conn, true, address, blockHeight, blockHash, txIndex, txHash)
		cli.id++
	}

	// send COMMIT_BLOCK and get response
	if noCommitBlock == false {
		cli.commitBlock(conn, true, blockHeight, blockHash)
	}
}

func (cli *CLI) commitClaim(conn ipc.Connection, success bool, address string, blockHeight uint64, blockHash string,
	txIndex uint64, txHash string) {
	var req core.CommitClaim
	var resp core.CommitClaim
	req.Success = success
	req.Address.SetString(address)
	req.BlockHeight = blockHeight
	req.BlockHash = make([]byte, core.BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			fmt.Printf("Failed to send COMMIT_CLAIM. Invalid block hash. %v\n", err)
			return
		}
		copy(req.BlockHash, bh)
	}
	req.TXIndex = txIndex
	req.TXHash = make([]byte, core.TXHashSize)
	if len(txHash) == 0 {
		binary.BigEndian.PutUint64(req.TXHash, blockHeight + txIndex)
	} else {
		th, err := hex.DecodeString(txHash)
		if err != nil {
			fmt.Printf("Failed to send COMMIT_CLAIM. Invalid TX hash. %v\n", err)
			return
		}
		copy(req.TXHash, th)
	}

	fmt.Printf("Send COMMIT_CLAIM message: %s\n", req.String())
	conn.SendAndReceive(core.MsgCommitClaim, cli.id, &req, &resp)
	fmt.Printf("Get COMMIT_CLAIM ack\n")
}

func (cli *CLI) startBlock(conn ipc.Connection, blockHeight uint64, blockHash string) {
	var req core.StartBlock
	var resp core.StartBlock
	req.BlockHash = make([]byte, core.BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			fmt.Printf("Failed to START_BLOCK. Invalid block hash. %v\n", err)
			return
		}
		copy(req.BlockHash, bh)
	}
	req.BlockHeight = blockHeight

	fmt.Printf("Send START_BLOCK message: %s\n", req.String())
	conn.SendAndReceive(core.MsgStartBlock, cli.id, &req, &resp)
	fmt.Printf("Get START_BLOCK response: %s\n", resp.String())
}

func (cli *CLI) commitBlock(conn ipc.Connection, success bool, blockHeight uint64, blockHash string) {
	var req core.CommitBlock
	var resp core.CommitBlock
	req.Success = success
	req.BlockHash = make([]byte, core.BlockHashSize)
	if len(blockHash) == 0 {
		binary.BigEndian.PutUint64(req.BlockHash, blockHeight)
	} else {
		bh, err := hex.DecodeString(blockHash)
		if err != nil {
			fmt.Printf("Failed to COMMIT_BLOCK. Invalid block hash. %v\n", err)
			return
		}
		copy(req.BlockHash, bh)
	}
	req.BlockHeight = blockHeight

	fmt.Printf("Send COMMIT_BLOCK message: %s\n", req.String())
	conn.SendAndReceive(core.MsgCommitBlock, cli.id, &req, &resp)
	fmt.Printf("Get COMMIT_BLOCK response: %s\n", resp.String())
}
