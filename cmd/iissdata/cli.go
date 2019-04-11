package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
)

const (
	DBDir     = "/Users/eunsoopark/test/rc_test"
	DBType    = "goleveldb"
	DBName    = "test"
)

type CLI struct{
	DB db.Database
}

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [db_name] [command]\n", os.Args[0])
	fmt.Printf("\t db_name     DB name\n")
	fmt.Printf("\t command     Command\n")
	fmt.Printf("\n [command]\n")
	fmt.Printf("\t read                                 Read the IISS data DB\n")
	fmt.Printf("\t delete                               Delete an IISS data DB\n")
	fmt.Printf("\t header VERSION BLOCKHEIGHT           Set VERSION and block height in header\n")
	fmt.Printf("\t gv BLOCKHEIGHT INCENTIVE RERWARD     Set governance variable\n")
	fmt.Printf("\t prep BLOCKHEIGHT GENERATOR VALIDATOR DELETE Add P-Rep statistics at block height \n")
	fmt.Printf("\t    GENERATOR                         Address of block generator\n")
	fmt.Printf("\t    VALIDATOR                         Addresses of block validators which seperated by ','\n")
	fmt.Printf("\t    DELETE                            Delete P-Rep statistics\n")
	fmt.Printf("\t tx ADDR BLOCKHEIGHT TYPE DATA        Directory where the IISS data DB is located\n")
	fmt.Printf("\t    DATA                              DATA should be one of followings\n")
	fmt.Printf("\t        DELEGATE_ADDR DELEGATE        delegation address and delegation amount\n")
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
	prepCmd := flag.NewFlagSet("prep", flag.ExitOnError)
	txCmd := flag.NewFlagSet("tx", flag.ExitOnError)

	headerVersion := headerCmd.Uint64("version", 1, "Version of IISS data")
	headerBlockHeight := headerCmd.Uint64("blockheight", 1, "Block height of IISS data")
	gvBlockHeight := gvCmd.Uint64("blockheight", 0, "Block height of Governance variable")
	gvIncentive := gvCmd.Uint64("incentive", 1, "P-Rep incentive in %")
	gvReward := gvCmd.Uint64("reward", 1, "P-Rep reward in %")
	prepBlockHeight := prepCmd.Uint64("blockheight", 0, "Block height of P-Rep statistics")
	prepGenerator := prepCmd.String("generator", "", "Address of block generator")
	prepValidator := prepCmd.String("validator", "", "Addresses of block validator")
	prepDelete := prepCmd.Bool("delete", false, "Delete P-Rep Statistics")
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
		cli.printUsage()
		os.Exit(1)
	}

	// Run the command
	if readCmd.Parsed() {
		// read
		cli.read(DBDir, dbName)
		return
	}

	if deleteCmd.Parsed() {
		path := filepath.Join(DBDir, dbName)
		os.RemoveAll(path)
		fmt.Printf("Delete %s\n", path)
		return
	}

	cli.DB = db.Open(DBDir, DBType, dbName)
	defer cli.DB.Close()

	if headerCmd.Parsed() {
		cli.header(*headerVersion, *headerBlockHeight)
		return
	}

	if gvCmd.Parsed() {
		cli.governanceVariable(*gvBlockHeight, *gvIncentive, *gvReward)
		return
	}

	if prepCmd.Parsed() {
		cli.prep(*prepBlockHeight, *prepGenerator, *prepValidator, *prepDelete)
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
