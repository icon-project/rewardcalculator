package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	DBTypeManagemment	= "manage"
	DBTypeAccount       = "account"
	DBTypeClaim			= "claim"
	DBTypePreCommit		= "preCommit"
	DBTypeCalcResult	= "calculateResult"
)

type CLI struct{
	cmd *flag.FlagSet
}

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [db_name] [db_type] [[options]]\n", os.Args[0])
	fmt.Printf("\t db_name     DB name\n")
	fmt.Printf("\t db_type     DB type (%s, %s, %s, %s, %s)\n",
		DBTypeManagemment,
		DBTypeAccount,
		DBTypeClaim,
		DBTypePreCommit,
		DBTypeCalcResult,
	)
	fmt.Printf("[options]\n")
	cli.cmd.PrintDefaults()
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	// Initialize the CLI
	cli.cmd = flag.NewFlagSet("query", flag.ExitOnError)
	address := cli.cmd.String("address", "", "Address string")
	blockHeight := cli.cmd.Uint64("blockHeight", 0, "Block height")

	cli.validateArgs()

	dbName := os.Args[1]
	dbType := os.Args[2]
	if err := cli.cmd.Parse(os.Args[3:]); err != nil || cli.cmd.NArg() > 0 {
		cli.printUsage()
		os.Exit(1)
	}

	if cli.cmd.Parsed() {
		fmt.Printf("%s, %d\n", *address, *blockHeight)

		cli.query(dbName, dbType, *address, *blockHeight)
	}
}
