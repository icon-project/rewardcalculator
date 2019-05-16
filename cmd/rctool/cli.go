package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

type CLI struct {
	id uint32
	conn ipc.Connection
}

func Display(data interface{}) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (cli *CLI) printUsage() {
	fmt.Printf("Read Info. from ICON reward calculator\n")
	fmt.Printf("Usage: %s COMMAND\n", os.Args[0])
	fmt.Printf("COMMAND\n")
	fmt.Printf("\t stats              Read statistics\n")
	fmt.Printf("\t dbinfo             Read DB Info.\n")
	fmt.Printf("\t prep               Read P-Rep\n")
	fmt.Printf("\t prepcandidate      Read P-Rep Candidate list\n")
	fmt.Printf("\t gv                 Read governance variable\n")
	fmt.Printf("\t logctx             Log CTX Info.\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) != 2 || os.Args[1] == "-h" {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	address := core.DebugAddress
	cmd := os.Args[1]

	// Connect to server
	net := "unix"
	conn, err := ipc.Dial(net, address)
	if err != nil {
		fmt.Printf("Failed to dial %s:%s err=%+v\n", net, address, err)
		os.Exit(1)
	}
	defer conn.Close()

	cli.conn = conn

	// Send message to server
	switch cmd {
	case "stats":
		cli.stats()
	case "dbinfo":
		cli.DBInfo()
	case "prep":
		cli.PRep()
	case "prepcandidate":
		cli.PRepCandidate()
	case "gv":
		cli.gv()
	case "logctx":
		cli.logCtx()
	default:
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) stats() {
	var req core.DebugMessage
	req.Cmd = core.DebugStatistics
	var resp core.ResponseDebugStats

	cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	fmt.Printf("stats command get response:\n%s\n", Display(resp))
}

func (cli *CLI) DBInfo() {
	var req core.DebugMessage
	req.Cmd = core.DebugDBInfo
	var resp core.ResponseDebugDBInfo

	cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	fmt.Printf("dbinfo command get response:\n%s\n", Display(resp))
}

func (cli *CLI) PRep() {
	var req core.DebugMessage
	req.Cmd = core.DebugPRep
	var resp core.ResponseDebugPRep

	cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	fmt.Printf("prep command get response:\n%s\n", Display(resp))
}

func (cli *CLI) PRepCandidate() {
	var req core.DebugMessage
	req.Cmd = core.DebugPRepCandidate
	var resp core.ResponseDebugPRepCandidate

	cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	fmt.Printf("prepcandidate command get response:\nTotal P-Rep candidate count: %d\n%s\n",
		len(resp.PRepCandidates), Display(resp))
}

func (cli *CLI) gv() {
	var req core.DebugMessage
	req.Cmd = core.DebugGV
	var resp core.ResponseDebugGV

	cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	fmt.Printf("gv command get response:\n%s\n", Display(resp))
}

func (cli *CLI) logCtx() {
	var req core.DebugMessage
	req.Cmd = core.DebugLogCTX

	cli.conn.Send(core.MsgDebug, cli.id, req)
}
