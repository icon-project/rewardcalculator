package main

import (
	"flag"
	"os"
)

type Input struct {
	path        string
	address     string
	height      uint64
	data        string
	help        bool
	accountType string
	rcDBRoot    string
}

const (
	pathUsage        = "DB Path"
	BlockHeightUsage = "Block height to query"
	AddressUsage     = "Address to query"
	IISSDataUsage    = "Data type to query. One of header, gv(governance variables), bp(block produce info), prep and tx. Print all iiss related data if this option has not given"
	RCDBRootUsage    = "path of RC DB"
	HelpMsgUsage     = "Print help message"
)

func initManageInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	ManageDataUsage := "Data type to query. One of gv(governance variables), di(db info) and pc(prep candidate). if this option has not given, Print all manage data"
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.StringVar(&input.data, "data", "", ManageDataUsage)
	flagSet.StringVar(&input.data, "d", "", ManageDataUsage)
	flagSet.StringVar(&input.address, "address", "", AddressUsage)
	flagSet.StringVar(&input.address, "a", "", AddressUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func initAccountInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	AccountQueryTypeUsage := "Type of account to query. `query` or `calculate`. if enter this option, must enter `dbroot` option."
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.StringVar(&input.address, "address", "", AddressUsage)
	flagSet.StringVar(&input.address, "a", "", AddressUsage)
	flagSet.StringVar(&input.rcDBRoot, "dbroot", "", RCDBRootUsage)
	flagSet.StringVar(&input.rcDBRoot, "r", "", RCDBRootUsage)
	flagSet.StringVar(&input.accountType, "type", "", AccountQueryTypeUsage)
	flagSet.StringVar(&input.accountType, "t", "", AccountQueryTypeUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func initClaimInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.StringVar(&input.address, "address", "", AddressUsage)
	flagSet.StringVar(&input.address, "a", "", AddressUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func initCalcResultInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.Uint64Var(&input.height, "height", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func initPreCommitInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.StringVar(&input.address, "address", "", AddressUsage)
	flagSet.StringVar(&input.address, "a", "", AddressUsage)
	flagSet.Uint64Var(&input.height, "height", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func initIISS(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.path, "path", "", pathUsage)
	flagSet.StringVar(&input.path, "p", "", pathUsage)
	flagSet.StringVar(&input.data, "data", "", IISSDataUsage)
	flagSet.StringVar(&input.data, "d", "", IISSDataUsage)
	flagSet.Uint64Var(&input.height, "height", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.help, "h", false, HelpMsgUsage)
	return input
}

func validateInput(flagSet *flag.FlagSet, err error, flag bool) {
	if err != nil {
		flagSet.PrintDefaults()
		os.Exit(1)
	}
	if flag {
		flagSet.PrintDefaults()
		os.Exit(0)
	}
}
