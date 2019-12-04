package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	DBNameManagement = "manage"
	DBNameAccount    = "account"
	DBNameClaim      = "claim"
	DBNamePreCommit  = "preCommit"
	DBNameCalcResult = "calcResult"
	DBNameIISS       = "iiss"

	DataTypeGV         = "gv"
	DataTypePRep       = "prep"
	DataTypeTX         = "tx"
	DataTypeHeader     = "header"
	DataTypeBP         = "bp"
	DataTypeDI         = "di"
	DataTypePC         = "pc"

	AccountTypeQuery = "query"
	AccountTypeCalculate = "calculate"
)

func printUsage() {
	fmt.Printf("Usage: %s [db_name](DB to query)\n", os.Args[0])
	fmt.Printf("\t db_name     DB Name (%s, %s, %s, %s, %s, %s)\n",
		DBNameManagement,
		DBNameAccount,
		DBNameClaim,
		DBNamePreCommit,
		DBNameCalcResult,
		DBNameIISS,
	)
}

func validateArgs() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}
}

func Run() {
	// Initialize the CLI

	dbName := os.Args[1]

	manageFlagSet := flag.NewFlagSet("manage", flag.ExitOnError)
	accountFlagSet := flag.NewFlagSet("account", flag.ExitOnError)
	claimFlagSet := flag.NewFlagSet("claim", flag.ExitOnError)
	preCommitFlagSet := flag.NewFlagSet("precommit", flag.ExitOnError)
	calcResultFlagSet := flag.NewFlagSet("calcresult", flag.ExitOnError)
	iissFlagSet := flag.NewFlagSet("iiss", flag.ExitOnError)

	manageInput := initManageInput(manageFlagSet)
	accountInput := initAccountInput(accountFlagSet)
	claimInput := initClaimInput(claimFlagSet)
	preCommitInput := initPreCommitInput(preCommitFlagSet)
	calcResultInput := initCalcResultInput(calcResultFlagSet)
	iissInput := initIISS(iissFlagSet)

	validateArgs()

	switch dbName {
	case DBNameManagement:
		err := manageFlagSet.Parse(os.Args[2:])
		validateInput(manageFlagSet, err, manageInput.help)
		queryManagementDB(*manageInput)
	case DBNameAccount:
		err := accountFlagSet.Parse(os.Args[2:])
		validateInput(accountFlagSet, err, accountInput.help)
		queryAccountDB(*accountInput)
	case DBNameClaim:
		err := claimFlagSet.Parse(os.Args[2:])
		validateInput(claimFlagSet, err, claimInput.help)
		queryClaimDB(*claimInput)
	case DBNamePreCommit:
		err := preCommitFlagSet.Parse(os.Args[2:])
		validateInput(preCommitFlagSet, err, preCommitInput.help)
		queryPreCommitDB(*preCommitInput)
	case DBNameCalcResult:
		err := calcResultFlagSet.Parse(os.Args[2:])
		validateInput(calcResultFlagSet, err, calcResultInput.help)
		queryCalcResultDB(*calcResultInput)
	case DBNameIISS:
		err := iissFlagSet.Parse(os.Args[2:])
		validateInput(iissFlagSet, err, iissInput.help)
		queryIISSDB(*iissInput)
	default:
		printUsage()
		os.Exit(1)
	}
}
