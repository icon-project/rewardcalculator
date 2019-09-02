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

	// flush READY message
	for true {
		var m core.ResponseVersion
		msg, _, _ := cli.conn.Receive(m)
		if msg == core.MsgReady {
			break
		}
	}

	// Send message to server
	switch cmd {
	case "stats":
		err = cli.stats()
	case "dbinfo":
		err = cli.DBInfo()
	case "prep":
		err = cli.PRep()
	case "prepcandidate":
		err = cli.PRepCandidate()
	case "gv":
		err = cli.gv()
	case "logctx":
		err = cli.logCtx()
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Failed to handle command. (%+v)\n", err)
	}
}

func (cli *CLI) stats() error {
	var req core.DebugMessage
	req.Cmd = core.DebugStatistics
	var resp core.ResponseDebugStats

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("stats command get response:\n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) DBInfo() error {
	var req core.DebugMessage
	req.Cmd = core.DebugDBInfo
	var resp core.ResponseDebugDBInfo

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("dbinfo command get response:\n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) PRep() error {
	var req core.DebugMessage
	req.Cmd = core.DebugPRep
	var resp core.ResponseDebugPRep

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("prep command get response:\n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) PRepCandidate() error {
	var req core.DebugMessage
	req.Cmd = core.DebugPRepCandidate
	var resp core.ResponseDebugPRepCandidate

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("prepcandidate command get response:\nTotal P-Rep candidate count: %d\n%s\n",
			len(resp.PRepCandidates), Display(resp))
	}

	return err
}

func (cli *CLI) gv() error {
	var req core.DebugMessage
	req.Cmd = core.DebugGV
	var resp core.ResponseDebugGV

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("gv command get response:\n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) logCtx() error {
	var req core.DebugMessage
	req.Cmd = core.DebugLogCTX

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}
