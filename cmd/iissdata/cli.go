package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

const (
	DBType = "goleveldb"
)

type CLI struct{
	DB db.Database
}

func (cli *CLI) printUsage() {
	fmt.Printf("Make IISS data DB")
	fmt.Printf("Usage: %s [DB] [COMMAND] [[options]]\n", os.Args[0])
	fmt.Printf("\t DB          DB name\n")
	fmt.Printf("\t COMMAND     Command\n")
	fmt.Printf("\t read               Read the IISS data DB\n")
	fmt.Printf("\t delete             Delete an IISS data DB\n")
	fmt.Printf("\t header             Set VERSION, revision and block height in header\n")
	fmt.Printf("\t gv                 Set governance variable\n")
	fmt.Printf("\t bp                 Add/delete Block produce Info.\n")
	fmt.Printf("\t prep               Add/delete P-Rep list\n")
	fmt.Printf("\t tx                 Directory where the IISS data DB is located\n")
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	dbName := os.Args[1]
	cmd := os.Args[2]

	// Initialize the CLI
	readCmd := flag.NewFlagSet("read", flag.ExitOnError)
	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	headerCmd := flag.NewFlagSet("header", flag.ExitOnError)
	gvCmd := flag.NewFlagSet("gv", flag.ExitOnError)
	bpCmd := flag.NewFlagSet("bp", flag.ExitOnError)
	prepCmd := flag.NewFlagSet("prep", flag.ExitOnError)
	txCmd := flag.NewFlagSet("tx", flag.ExitOnError)

	headerVersion := headerCmd.Uint64("version", core.IISSDataVersion, "Version of IISS data")
	headerRevision := headerCmd.Uint64("revision", core.IISSDataRevisionDefault, "Revision of ICON Service")
	headerBlockHeight := headerCmd.Uint64("blockheight", 1, "Block height of IISS data")

	gvBlockHeight := gvCmd.Uint64("blockheight", 0, "Block height of Governance variable")
	gvIncentive := gvCmd.Uint64("incentive", 1, "P-Rep incentive in %")
	gvReward := gvCmd.Uint64("reward", 1, "P-Rep reward in %")
	gvMainPRepCount := gvCmd.Uint64("mainprepcount", core.NumMainPRep, "Main P-Rep count")
	gvSubPRepCount := gvCmd.Uint64("subprepcount", core.NumSubPRep, "Sub P-Rep count")

	bpBlockHeight := bpCmd.Uint64("blockheight", 0, "Block height of Block produce Info.")
	bpGenerator := bpCmd.String("generator", "", "Address of block generator")
	bpValidator := bpCmd.String("validator", "", "Addresses of block validator")
	bpDelete := bpCmd.Bool("delete", false, "Delete P-Rep Statistics")

	prepBlockHeight := prepCmd.Uint64("blockheight", 0, "Block height of P-Rep list")
	prepList := prepCmd.String("preplist", "", "Addresses of Main/Sub P-Rep")
	prepDelegationList := prepCmd.String("delegationlist", "", "Delegation amount of Main/Sub P-Rep")
	prepDelete := prepCmd.Bool("delete", false, "Delete P-Rep list")

	txIndex := txCmd.Uint64("index", 0, "TX index")
	txAddress := txCmd.String("address", "", "TX owner address")
	txBlockHeight := txCmd.Uint64("blockheight", 0, "Block height of TX")
	txType := txCmd.Uint64("type", 0, "Type of TX. " +
		"(0:delegation, 1:P-Rep register, 2:P-Rep unregister")
	txDelegateAddress := txCmd.String("dg-address", "", "Delegation address")
	txDelegateAmount:= txCmd.Uint64("dg-amount", 10, "Delegation amount")

	// Parse the CLI
	switch cmd {
	case "read":
		err := readCmd.Parse(os.Args[3:])
		if err != nil {
			readCmd.Usage()
			os.Exit(1)
		}
	case "delete":
		err := deleteCmd.Parse(os.Args[3:])
		if err != nil {
			deleteCmd.PrintDefaults()
			os.Exit(1)
		}
	case "header":
		err := headerCmd.Parse(os.Args[3:])
		if err != nil {
			headerCmd.Usage()
			os.Exit(1)
		}
	case "gv":
		err := gvCmd.Parse(os.Args[3:])
		if err != nil {
			gvCmd.Usage()
			os.Exit(1)
		}
	case "bp":
		err := bpCmd.Parse(os.Args[3:])
		if err != nil {
			bpCmd.Usage()
			os.Exit(1)
		}
	case "prep":
		err := prepCmd.Parse(os.Args[3:])
		if err != nil {
			prepCmd.Usage()
			os.Exit(1)
		}
	case "tx":
		err := txCmd.Parse(os.Args[3:])
		if err != nil {
			txCmd.Usage()
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown command : %s\n", cmd)
		cli.printUsage()
		os.Exit(1)
	}

	if strings.HasSuffix(dbName, "/") {
		dbName = dbName[:len(dbName) - len("/")]
	}
	dbDir, dbName := filepath.Split(dbName)

	// Run the command
	if readCmd.Parsed() {
		// read
		cli.read(dbDir, dbName)
		return
	}

	if deleteCmd.Parsed() {
		path := filepath.Join(dbDir, dbName)
		os.RemoveAll(path)
		fmt.Printf("Delete %s\n", path)
		return
	}

	cli.DB = db.Open(dbDir, DBType, dbName)
	defer cli.DB.Close()

	if headerCmd.Parsed() {
		cli.header(*headerVersion, *headerBlockHeight, *headerRevision)
		return
	}

	if gvCmd.Parsed() {
		cli.governanceVariable(*gvBlockHeight, *gvIncentive, *gvReward, *gvMainPRepCount, *gvSubPRepCount)
		return
	}

	if bpCmd.Parsed() {
		cli.bp(*bpBlockHeight, *bpGenerator, *bpValidator, *bpDelete)
		return
	}

	if prepCmd.Parsed() {
		cli.prep(*prepBlockHeight, *prepList, *prepDelegationList, *prepDelete)
		return
	}

	if txCmd.Parsed() {
		if *txAddress == "" {
			txCmd.PrintDefaults()
			os.Exit(1)
		}
		cli.transaction(*txIndex, *txAddress, *txBlockHeight, *txType, *txDelegateAddress, *txDelegateAmount)
		return
	}
}
