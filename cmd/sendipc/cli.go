package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/icon-project/rewardcalculator/core"
)

type CLI struct {
	id uint32
}

func Display(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [ADDRESS] [COMMAND] [[options]]\n", os.Args[0])
	fmt.Printf("ADDRESS         Unix domain socket path\n")
	fmt.Printf("COMMAND\n")
	fmt.Printf("\t version                   Send a VERSION message\n")
	fmt.Printf("\t init                      Send a INIT message\n")
	fmt.Printf("\t query                     Send a QUERY message to query I-Score\n")
	fmt.Printf("\t claim                     Send a CLAIM message to claim I-Score\n")
	fmt.Printf("\t commitclaim               Send a COMMIT_CLAIM message to commit CLAIM message\n")
	fmt.Printf("\t commitblock               Send a COMMIT_BLOCK message to commit block\n")
	fmt.Printf("\t calculate                 Send a CALCULATE message to update I-Score DB\n")
	fmt.Printf("\t query_calculate_status    Send a QUERY_CALCULATE_STATUS message\n")
	fmt.Printf("\t query_calculate_result    Send a QUERY_CALCULATE_RESULT message\n")
	fmt.Printf("\t monitor                   Monitor account in configuration file\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
	cli.id = 1
}

func (cli *CLI) Run() {
	cli.validateArgs()

	address := os.Args[1]
	cmd := os.Args[2]

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	initBlockHeight := initCmd.Uint64("blockheight", 0, "Block height")

	queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	queryAddress := queryCmd.String("address", "", "Account address")

	claimCmd := flag.NewFlagSet("claim", flag.ExitOnError)
	claimAddress := claimCmd.String("address", "", "Account address")
	claimBlockHeight := claimCmd.Uint64("blockheight", 0, "Block height")
	claimBlockHash := claimCmd.String("blockhash", "", "Block hash")
	claimTXIndex := claimCmd.Uint64("txindex", 0, "TX index")
	claimTXHash := claimCmd.String("txhash", "", "TX hash")
	claimNoCommit := claimCmd.Bool("no-commit", false, "Do not send COMMIT_CLAIM for CLAIM TX")
	claimNoCommitBlock := claimCmd.Bool("no-commit-block", false, "Do not send COMMIT_BLOCK for CLAIM Block")

	commitClaimCmd := flag.NewFlagSet("commitclaim", flag.ExitOnError)
	commitClaimFail := commitClaimCmd.Bool("fail", true, "Success")
	commitClaimAddress := commitClaimCmd.String("address", "", "Account address")
	commitClaimBlockHeight := commitClaimCmd.Uint64("blockheight", 0, "Block height")
	commitClaimBlockHash := commitClaimCmd.String("blockhash", "", "Block hash")
	commitClaimTXIndex := commitClaimCmd.Uint64("txindex", 0, "TX index")
	commitClaimTXHash := commitClaimCmd.String("txhash", "", "TX hash")

	commitBlockCmd := flag.NewFlagSet("commitblock", flag.ExitOnError)
	commitBlockFail := commitBlockCmd.Bool("fail", true, "Success")
	commitBlockBlockHeight := commitBlockCmd.Uint64("blockheight", 0, "Block height")
	commitBlockBlockHash := commitBlockCmd.String("blockhash", "", "Block hash")

	calculateCmd := flag.NewFlagSet("calculate", flag.ExitOnError)
	calculateIISSData := calculateCmd.String("iissdata", "", "IISS data DB path(Required)")
	calculateBlockHeight := calculateCmd.Uint64("blockheight", 0, "Block height to calculate. Set 0 if you want current block+1")

	monitorCmd := flag.NewFlagSet("monitor", flag.ExitOnError)
	monitorConfig := monitorCmd.String("config", "./monitor.json", "Monitoring configuration file path")
	monitorURL := monitorCmd.String("url", "http://localhost:9091", "Push URL")

	queryCalculateStatusCmd := flag.NewFlagSet("query_calculate_status", flag.ExitOnError)

	queryCRCmd := flag.NewFlagSet("query_calculate_result", flag.ExitOnError)
	queryCRBlockHeight := queryCRCmd.Uint64("blockheight", 0, "Block height(Required)")

	rollbackCmd := flag.NewFlagSet("rollback", flag.ExitOnError)
	rollbackBlockHeight := rollbackCmd.Uint64("blockheight", 0, "Rollback block height(Required)")
	rollbackBlockHash := rollbackCmd.String("blockhash", "", "Rollback block hash(Required)")

	// Parse the CLI
	switch cmd {
	case "version":
		err := versionCmd.Parse(os.Args[3:])
		if err != nil {
			versionCmd.PrintDefaults()
			os.Exit(1)
		}
	case "init":
		err := initCmd.Parse(os.Args[3:])
		if err != nil {
			initCmd.PrintDefaults()
			os.Exit(1)
		}
	case "query":
		err := queryCmd.Parse(os.Args[3:])
		if err != nil {
			queryCmd.PrintDefaults()
			os.Exit(1)
		}
	case "claim":
		err := claimCmd.Parse(os.Args[3:])
		if err != nil {
			claimCmd.PrintDefaults()
			os.Exit(1)
		}
	case "commitclaim":
		err := commitClaimCmd.Parse(os.Args[3:])
		if err != nil {
			commitClaimCmd.PrintDefaults()
			os.Exit(1)
		}
	case "commitblock":
		err := commitBlockCmd.Parse(os.Args[3:])
		if err != nil {
			commitBlockCmd.PrintDefaults()
			os.Exit(1)
		}
	case "calculate":
		err := calculateCmd.Parse(os.Args[3:])
		if err != nil {
			calculateCmd.PrintDefaults()
			os.Exit(1)
		}
	case "monitor":
		err := monitorCmd.Parse(os.Args[3:])
		if err != nil {
			monitorCmd.PrintDefaults()
			os.Exit(1)
		}
	case "query_calculate_status":
		err := queryCalculateStatusCmd.Parse(os.Args[3:])
		if err != nil {
			queryCalculateStatusCmd.PrintDefaults()
			os.Exit(1)
		}
	case "query_calculate_result":
		err := queryCRCmd.Parse(os.Args[3:])
		if err != nil {
			queryCRCmd.PrintDefaults()
			os.Exit(1)
		}
	case "rollback":
		err := rollbackCmd.Parse(os.Args[3:])
		if err != nil {
			rollbackCmd.PrintDefaults()
			os.Exit(1)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	// Connect to server
	net := "unix"
	conn, err := ipc.Dial(net, address)
	if err != nil {
		fmt.Printf("Failed to dial %s:%s err=%+v\n", net, address, err)
		os.Exit(1)
	}
	defer conn.Close()

	// flush READY message
	for true {
		var m core.ResponseVersion
		msg, _, _ := conn.Receive(m)
		if msg == core.MsgReady {
			break
		}
	}

	// Send message to server

	if versionCmd.Parsed() {
		// send VERSION message
		cli.version(conn)
	}

	if initCmd.Parsed() {
		// send INIT message
		cli.init(conn, *initBlockHeight)
	}

	if claimCmd.Parsed() {
		if *claimAddress == "" {
			claimCmd.PrintDefaults()
			os.Exit(1)
		}
		// send CLAIM message
		cli.claim(conn, *claimAddress, *claimBlockHeight, *claimBlockHash, *claimTXIndex, *claimTXHash,
			*claimNoCommit, *claimNoCommitBlock)
	}

	if commitClaimCmd.Parsed() {
		if *commitClaimAddress == "" {
			claimCmd.PrintDefaults()
			os.Exit(1)
		}
		// send COMMIT_CLAIM message
		cli.commitClaim(conn, *commitClaimFail, *commitClaimAddress, *commitClaimBlockHeight, *commitClaimBlockHash,
			*commitClaimTXIndex, *commitClaimTXHash)
	}

	if commitBlockCmd.Parsed() {
		// send COMMIT_BLOCK message
		cli.commitBlock(conn, *commitBlockFail, *commitBlockBlockHeight, *commitBlockBlockHash)
	}

	if queryCmd.Parsed() {
		if *queryAddress == "" {
			queryCmd.PrintDefaults()
			os.Exit(1)
		}
		// send QUERY message
		cli.query(conn, *queryAddress)
	}

	if calculateCmd.Parsed() {
		if *calculateIISSData == "" {
			calculateCmd.PrintDefaults()
			os.Exit(1)
		}
		start := time.Now()

		// send CALCULATE message
		cli.calculate(conn, *calculateIISSData, *calculateBlockHeight)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if queryCalculateStatusCmd.Parsed() {
		cli.queryCalculateStatus(conn)
	}

	if queryCRCmd.Parsed() {
		cli.queryCalculateResult(conn, *queryCRBlockHeight)
	}

	if monitorCmd.Parsed() {
		cli.monitor(conn, *monitorConfig, *monitorURL)
	}

	if rollbackCmd.Parsed() {
		if *rollbackBlockHeight == 0 || *rollbackBlockHash == "" {
			rollbackCmd.PrintDefaults()
			os.Exit(1)
		}
		cli.rollback(conn, *rollbackBlockHeight, *rollbackBlockHash)
	}
}
