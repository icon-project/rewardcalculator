package main

import (
	"flag"
	"os"
)

const (
	BlockHeightUsage = "Block height to query"
	AddressUsage     = "Address to query"
	HelpMsgUsage     = "Print help message"
)

type Options struct {
	address     string
	blockHeight uint64
	help        bool
}

func initCalcDebugResultOptions(flagSet *flag.FlagSet) *Options {
	options := new(Options)
	flagSet.StringVar(&options.address, "address", "", AddressUsage)
	flagSet.StringVar(&options.address, "a", "", AddressUsage)
	flagSet.Uint64Var(&options.blockHeight, "blockheight", 0, BlockHeightUsage)
	flagSet.Uint64Var(&options.blockHeight, "b", 0, BlockHeightUsage)
	flagSet.BoolVar(&options.help, "help", false, HelpMsgUsage)
	flagSet.BoolVar(&options.help, "h", false, HelpMsgUsage)
	return options
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
