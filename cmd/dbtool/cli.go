package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/icon-project/rewardcalculator/cmd/common"
	"os"
)

const (
	DBNameManagement      = "manage"
	DBNameAccount         = "account"
	DBNameClaim           = "claim"
	DBNameClaimBackup     = "claimBackup"
	DBNamePreCommit       = "preCommit"
	DBNameCalcResult      = "calcResult"
	DBNameIISS            = "iiss"
	DBNameCalcDebugResult = "calcDebug"

	DataTypeGV     = "gv"
	DataTypePRep   = "prep"
	DataTypeTX     = "tx"
	DataTypeHeader = "header"
	DataTypeBP     = "bp"
	DataTypeDI     = "di"
	DataTypePC     = "pc"

	AccountDBTypeQuery     = "query"
	AccountDBTypeCalculate = "calculate"
)

func printUsage() {
	fmt.Printf("Usage: %s [db_name] [[options]]\n", os.Args[0])
	fmt.Printf("\t db_name     DB Name (%s, %s, %s, %s, %s, %s, %s, %s)\n",
		DBNameManagement,
		DBNameAccount,
		DBNameClaim,
		DBNameClaimBackup,
		DBNamePreCommit,
		DBNameCalcResult,
		DBNameIISS,
		DBNameCalcDebugResult,
	)
}

func validateArgs() (err error) {
	if len(os.Args) == 1 {
		printUsage()
		os.Exit(0)
	} else if len(os.Args) == 2 {
		if (os.Args[1] == "-h") || (os.Args[1] == "-help") {
			printUsage()
			os.Exit(0)
		}
		return errors.New("invalid input")
	} else if len(os.Args) < 3 {
		printUsage()
		return errors.New("invalid input")
	}
	return nil
}

func Run() (err error) {
	// Initialize the CLI

	if err = validateArgs(); err != nil {
		return err
	}

	dbName := os.Args[1]

	manageFlagSet := flag.NewFlagSet(DBNameManagement, flag.ExitOnError)
	accountFlagSet := flag.NewFlagSet(DBNameAccount, flag.ExitOnError)
	claimFlagSet := flag.NewFlagSet(DBNameClaim, flag.ExitOnError)
	claimBackupFlagSet := flag.NewFlagSet(DBNameClaimBackup, flag.ExitOnError)
	preCommitFlagSet := flag.NewFlagSet(DBNamePreCommit, flag.ExitOnError)
	calcResultFlagSet := flag.NewFlagSet(DBNameCalcResult, flag.ExitOnError)
	iissFlagSet := flag.NewFlagSet(DBNameIISS, flag.ExitOnError)
	calcDebugFlagSet := flag.NewFlagSet(DBNameCalcDebugResult, flag.ExitOnError)

	manageInput := common.InitManageInput(manageFlagSet)
	accountInput := common.InitAccountInput(accountFlagSet)
	claimInput := common.InitClaimInput(claimFlagSet)
	claimBackupInput := common.InitClaimBackupInput(claimBackupFlagSet)
	preCommitInput := common.InitPreCommitInput(preCommitFlagSet)
	calcResultInput := common.InitCalcResultInput(calcResultFlagSet)
	iissInput := common.InitIISS(iissFlagSet)
	calcDebugInput := common.InitCalcDebugResult(calcDebugFlagSet)

	switch dbName {
	case DBNameManagement:
		err = manageFlagSet.Parse(os.Args[2:])
		common.ValidateInput(manageFlagSet, err, manageInput.Help)
		err = queryManagementDB(*manageInput)
	case DBNameAccount:
		err = accountFlagSet.Parse(os.Args[2:])
		common.ValidateInput(accountFlagSet, err, accountInput.Help)
		err = queryAccountDB(*accountInput)
	case DBNameClaim:
		err = claimFlagSet.Parse(os.Args[2:])
		common.ValidateInput(claimFlagSet, err, claimInput.Help)
		err = queryClaimDB(*claimInput)
	case DBNameClaimBackup:
		err = claimBackupFlagSet.Parse(os.Args[2:])
		common.ValidateInput(claimBackupFlagSet, err, claimBackupInput.Help)
		err = queryClaimBackupDB(*claimBackupInput)
	case DBNamePreCommit:
		err = preCommitFlagSet.Parse(os.Args[2:])
		common.ValidateInput(preCommitFlagSet, err, preCommitInput.Help)
		err = queryPreCommitDB(*preCommitInput)
	case DBNameCalcResult:
		err = calcResultFlagSet.Parse(os.Args[2:])
		common.ValidateInput(calcResultFlagSet, err, calcResultInput.Help)
		err = queryCalcResultDB(*calcResultInput)
	case DBNameIISS:
		err = iissFlagSet.Parse(os.Args[2:])
		common.ValidateInput(iissFlagSet, err, iissInput.Help)
		err = queryIISSDB(*iissInput)
	case DBNameCalcDebugResult:
		err = calcDebugFlagSet.Parse(os.Args[2:])
		common.ValidateInput(calcDebugFlagSet, err, calcDebugInput.Help)
		err = common.QueryCalcDebugDB(*calcDebugInput)
	default:
		printUsage()
		err = errors.New("invalid dbName")
	}
	return err
}
