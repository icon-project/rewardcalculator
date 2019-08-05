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

const (
	msgVERSION   uint = 0
	msgClaim          = 1
	msgQuery          = 2
	msgCalculate      = 3
	msgCommitBlock    = 4
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
	fmt.Printf("Usage: %s [ADDRESS] [COMMAND]\n", os.Args[0])
	fmt.Printf("ADDRESS         Unix domain socket path\n")
	fmt.Printf("COMMAND\n")
	fmt.Printf("\t version                            Send a VERSION message\n")
	fmt.Printf("\t query ACCOUNT                      Send a QUERY message to query I-Score\n")
	fmt.Printf("\t       ACCOUNT                      Account adddress(Required)\n")
	fmt.Printf("\t claim ACCOUNT                      Send a CLAIM message to claim I-Score\n")
	fmt.Printf("\t       ACCOUNT                      Account adddress(Required)\n")
	fmt.Printf("\t calculate IISSDATA BLOCKHEIGHT     Send a CALCULATE message to update I-Score DB\n")
	fmt.Printf("\t       IISSDATA                     IISS data DB path(Required)\n")
	fmt.Printf("\t       BLOCKHEIGHT                  Block height to calculate. Set 0 if you want current block+1\n")
	fmt.Printf("\t monitor CONFIG                     Monitor account in configuration file\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	address := os.Args[1]
	cmd := os.Args[2]

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	queryAddress := queryCmd.String("address", "", "Account address(Required)")

	claimCmd := flag.NewFlagSet("claim", flag.ExitOnError)
	claimAddress := claimCmd.String("address", "", "Account address(Required)")
	claimBlockHeight := claimCmd.Uint64("blockheight", 0, "Block height(Required)")

	calculateCmd := flag.NewFlagSet("calculate", flag.ExitOnError)
	calculateIISSData := calculateCmd.String("iissdata", "", "IISS data DB path(Required)")
	calculateBlockHeight := calculateCmd.Uint64("blockheight", 0, "Block height to calculate. Set 0 if you want current block+1")

	monitorCmd := flag.NewFlagSet("monitor", flag.ExitOnError)
	monitorConfig := monitorCmd.String("config", "./monitor.json", "Monitoring configuration file path")
	monitorURL := monitorCmd.String("url", "http://localhost:9091", "Push URL")

	// Parse the CLI
	switch cmd {
	case "version":
		err := versionCmd.Parse(os.Args[3:])
		if err != nil {
			versionCmd.PrintDefaults()
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

	// flush VERSION message
	var m core.ResponseVersion
	conn.Receive(m)

	// Send message to server

	if versionCmd.Parsed() {
		// send calculate message
		cli.version(conn)
	}

	if claimCmd.Parsed() {
		if *claimAddress == "" || *claimBlockHeight == 0 {
			claimCmd.PrintDefaults()
			os.Exit(1)
		}
		start := time.Now()

		// send claim message
		cli.claim(conn, *claimAddress, *claimBlockHeight)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if queryCmd.Parsed() {
		if *queryAddress == "" {
			queryCmd.PrintDefaults()
			os.Exit(1)
		}
		start := time.Now()

		// send query message
		cli.query(conn, *queryAddress)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if calculateCmd.Parsed() {
		if *calculateIISSData == "" {
			calculateCmd.PrintDefaults()
			os.Exit(1)
		}
		start := time.Now()

		// send calculate message
		cli.calculate(conn, *calculateIISSData, *calculateBlockHeight)

		end := time.Now()
		diff := end.Sub(start)
		fmt.Printf("Duration : %v\n", diff)
	}

	if monitorCmd.Parsed() {
		cli.monitor(conn, *monitorConfig, *monitorURL)
	}
}
