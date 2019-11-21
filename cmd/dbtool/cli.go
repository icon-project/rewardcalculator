package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	DBTypeManagement = "manage"
	DBTypeAccount    = "account"
	DBTypeClaim      = "claim"
	DBTypePreCommit  = "preCommit"
	DBTypeCalcResult = "calculateResult"
	DBTypeHeader     = "header"
	DBTypeGV         = "governanceInfo"
	DBTypeBPInfo     = "blockProduce"
	DBTypePRep       = "prep"
	DBTypeTX         = "tx"

	ClaimPath      = "claim"
	PreCommitPath  = "PreCommit"
	CalcResultPath = "calculation_result"

	GVPrefixLen               = 2
	PRepCandidatePrefixLen    = 2
	BlockProduceInfoPrefixLen = 2
	PRepPrefixLen             = 2
	TransactionPrefixLen      = 2
)

type CLI struct {
	cmd *flag.FlagSet
}

func (cli *CLI) printUsage() {
	fmt.Printf("Usage: %s [db_name] [db_type] [[options]]\n", os.Args[0])
	fmt.Printf("\t db_name     DB name\n")
	fmt.Printf("\t db_type     DB type (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)\n",
		DBTypeManagement,
		DBTypeAccount,
		DBTypeClaim,
		DBTypePreCommit,
		DBTypeCalcResult,
		DBTypeHeader,
		DBTypeGV,
		DBTypeBPInfo,
		DBTypePRep,
		DBTypeTX,
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
	index := cli.cmd.Int64("index", -1, "option for tx command")

	cli.validateArgs()

	dbName := os.Args[1]
	dbType := os.Args[2]
	if err := cli.cmd.Parse(os.Args[3:]); err != nil || cli.cmd.NArg() > 0 {
		cli.printUsage()
		os.Exit(1)
	}

	if cli.cmd.Parsed() {
		fmt.Printf("%s, %d %d\n", *address, *blockHeight, *index)

		cli.query(dbName, dbType, *address, *blockHeight, *index)
	}
}
