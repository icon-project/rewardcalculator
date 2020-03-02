package common

import (
	"flag"
	"os"
)

type Input struct {
	Path        string
	Address     string
	Height      uint64
	Data        string
	Help        bool
	AccountType string
	RcDBRoot    string
}

const (
	pathUsage        = "DB Path"
	BlockHeightUsage = "Block height to query"
	AddressUsage     = "Address to query"
	IISSDataUsage    = "Data type to query. One of header, gv(governance variables), bp(block produce info), prep and tx. Print all iiss related data if this option has not given"
	RCDBRootUsage    = "path of RC DB"
	HelpMsgUsage     = "Print help message"
)

func InitManageInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	ManageDataUsage := "Data type to query. One of di(db info), gv(governance variables), prep and pc(prep candidate). if this option has not given, Print all manage data"
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Data, "data", "", ManageDataUsage)
	flagSet.StringVar(&input.Data, "d", "", ManageDataUsage)
	flagSet.StringVar(&input.Address, "address", "", AddressUsage)
	flagSet.StringVar(&input.Address, "a", "", AddressUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitAccountInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	AccountQueryTypeUsage := "Type of account to query. `query` or `calculate`. if enter this option, must enter `dbroot` option."
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Address, "address", "", AddressUsage)
	flagSet.StringVar(&input.Address, "a", "", AddressUsage)
	flagSet.StringVar(&input.RcDBRoot, "dbroot", "", RCDBRootUsage)
	flagSet.StringVar(&input.RcDBRoot, "d", "", RCDBRootUsage)
	flagSet.StringVar(&input.AccountType, "type", "", AccountQueryTypeUsage)
	flagSet.StringVar(&input.AccountType, "t", "", AccountQueryTypeUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitClaimInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Address, "address", "", AddressUsage)
	flagSet.StringVar(&input.Address, "a", "", AddressUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitClaimBackupInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.Uint64Var(&input.Height, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.Height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitCalcResultInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.Uint64Var(&input.Height, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.Height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitPreCommitInput(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Address, "address", "", AddressUsage)
	flagSet.StringVar(&input.Address, "a", "", AddressUsage)
	flagSet.Uint64Var(&input.Height, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.Height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitIISS(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Data, "data", "", IISSDataUsage)
	flagSet.StringVar(&input.Data, "d", "", IISSDataUsage)
	flagSet.Uint64Var(&input.Height, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.Height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func InitCalcDebugResult(flagSet *flag.FlagSet) *Input {
	input := new(Input)
	flagSet.StringVar(&input.Path, "path", "", pathUsage)
	flagSet.StringVar(&input.Path, "p", "", pathUsage)
	flagSet.StringVar(&input.Address, "address", "", AddressUsage)
	flagSet.StringVar(&input.Address, "a", "", AddressUsage)
	flagSet.Uint64Var(&input.Height, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&input.Height, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&input.Help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&input.Help, "h", false, HelpMsgUsage)
	return input
}

func ValidateInput(flagSet *flag.FlagSet, err error, flag bool) {
	if err != nil {
		flagSet.PrintDefaults()
		os.Exit(1)
	}
	if flag {
		flagSet.PrintDefaults()
		os.Exit(0)
	}
}
