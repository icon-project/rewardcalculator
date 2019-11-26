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
	DBTypeCalcResult = "calcResult"
	DBTypeHeader     = "header"
	DBTypeGV         = "gvInfo"
	DBTypeBPInfo     = "bp"
	DBTypePRep       = "prep"
	DBTypeTX         = "tx"

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
	fmt.Printf("Usage: %s [db_name](DB to query)\n", os.Args[0])
	fmt.Printf("\t db_name     DB Name (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)\n",
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
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 3 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	// Initialize the CLI

	dbName := os.Args[1]

	manageCmd := flag.NewFlagSet("manage", flag.ExitOnError)
	accountCmd := flag.NewFlagSet("account", flag.ExitOnError)
	claimCmd := flag.NewFlagSet("claim", flag.ExitOnError)
	preCommitCmd := flag.NewFlagSet("precommit", flag.ExitOnError)
	calcResultCmd := flag.NewFlagSet("calcresult", flag.ExitOnError)

	headerCmd := flag.NewFlagSet("header", flag.ExitOnError)
	gvCmd := flag.NewFlagSet("gv", flag.ExitOnError)
	bpCmd := flag.NewFlagSet("bp", flag.ExitOnError)
	prepCmd := flag.NewFlagSet("prep", flag.ExitOnError)
	txCmd := flag.NewFlagSet("tx", flag.ExitOnError)

	manageDBPath := manageCmd.String("path", "", "path of DB")
	manageHelp := manageCmd.Bool("h", false, "print help message")

	accountDBPath := accountCmd.String("path", "", "path of DB")
	rcDBPath := accountCmd.String("rpath", "", "path of rcDB root")
	addressForAccountDB := accountCmd.String("address", "", "address to query")
	accountHelp := accountCmd.Bool("h", false, "print help message")

	claimDBPath := claimCmd.String("path", "", "path of DB")
	addressClaimDB := claimCmd.String("address", "", "address to query")
	claimHelp := claimCmd.Bool("h", false, "print help message")

	preCommitDBPath := preCommitCmd.String("path", "", "path of DB")
	addressPreCommitDB := preCommitCmd.String("address", "", "address to query")
	blockHeightPreCommitDB := preCommitCmd.Uint64("height", 0, "blockHeight to query")
	precommitHelp := preCommitCmd.Bool("h", false, "print help message")

	calcResultDBPath := calcResultCmd.String("path", "", "path of DB")
	blockHeightCalcResultDB := calcResultCmd.Uint64("height", 0, "blockHeight to query")
	calcResultHelp := calcResultCmd.Bool("h", false, "print help message")

	headerDBPath := headerCmd.String("path", "", "path of DB")
	headerHelp := headerCmd.Bool("h", false, "print help message")

	gvDBPath := gvCmd.String("path", "", "path of DB")
	blockHeightGvDB := gvCmd.Uint64("height", 0, "blockHeight to query")
	gvHelp := gvCmd.Bool("h", false, "print help message")

	bpDBPath := bpCmd.String("path", "", "path of DB")
	blockHeightBpDB := bpCmd.Uint64("height", 0, "blockHeight to query")
	bpHelp := bpCmd.Bool("h", false, "print help message")

	prepDBPath := prepCmd.String("path", "", "path of DB")
	blockHeightPrepDB := prepCmd.Uint64("height", 0, "blockHeight to query")
	prepHelp := prepCmd.Bool("h", false, "print help message")

	txDBPath := txCmd.String("path", "", "path of DB")
	indexTxDB := txCmd.String("index", "", "index to query")
	txHelp := txCmd.Bool("h", false, "print help message")

	switch dbName {
	case DBTypeManagement:
		err := manageCmd.Parse(os.Args[2:])
		validateInput(manageCmd, err, *manageHelp)
		ManagerDB{dbPath: *manageDBPath}.query()
	case DBTypeAccount:
		err := accountCmd.Parse(os.Args[2:])
		validateInput(accountCmd, err, *accountHelp)
		AccountDB{}.query(*accountDBPath, *rcDBPath, *addressForAccountDB)
	case DBTypeClaim:
		err := claimCmd.Parse(os.Args[2:])
		validateInput(claimCmd, err, *claimHelp)
		ClaimDB{dbPath: *claimDBPath}.query(*addressClaimDB)
	case DBTypePreCommit:
		err := preCommitCmd.Parse(os.Args[2:])
		validateInput(preCommitCmd, err, *precommitHelp)
		PreCommitDB{dbPath: *preCommitDBPath}.query(*addressPreCommitDB, *blockHeightPreCommitDB)
	case DBTypeCalcResult:
		err := calcResultCmd.Parse(os.Args[2:])
		validateInput(calcResultCmd, err, *calcResultHelp)
		CalcResultDB{dbPath: *calcResultDBPath}.query(*blockHeightCalcResultDB)
	case DBTypeHeader:
		err := headerCmd.Parse(os.Args[2:])
		validateInput(headerCmd, err, *headerHelp)
		HeaderDB{dbPath: *headerDBPath}.query()
	case DBTypeGV:
		err := gvCmd.Parse(os.Args[2:])
		validateInput(gvCmd, err, *gvHelp)
		GvDB{dbPath: *gvDBPath}.query(*blockHeightGvDB)
	case DBTypeBPInfo:
		err := bpCmd.Parse(os.Args[2:])
		validateInput(bpCmd, err, *bpHelp)
		BpDB{dbPath: *bpDBPath}.query(*blockHeightBpDB)
	case DBTypePRep:
		err := prepCmd.Parse(os.Args[2:])
		validateInput(prepCmd, err, *prepHelp)
		PrepDB{dbPath: *prepDBPath}.query(*blockHeightPrepDB)
	case DBTypeTX:
		err := txCmd.Parse(os.Args[2:])
		validateInput(txCmd, err, *txHelp)
		TxDB{dbPath: *txDBPath}.query(*indexTxDB)
	default:
		cli.printUsage()
		os.Exit(1)
	}
}
