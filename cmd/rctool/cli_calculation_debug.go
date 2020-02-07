package main

import (
	"flag"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) calculateDebug(input []string) error {
	calcDebugUsage := fmt.Errorf("\nCommands\n" +
		"\t enable \t enable calculation debug\n" +
		"\t disable \t disable calculation debug\n" +
		"\t add <Address> \t add calculation debugging address\n" +
		"\t delete <Address> \t delete calculation debugging address\n" +
		"\t output <outputPath> \t change calculation debugging output path\n" +
		"\t list \t print calculation debugging addresses\n" +
		"\t result \t print calculation debugging result. give -h option to check options")

	if len(input) == 0 {
		return calcDebugUsage
	}
	var err error
	resultFlagSet := flag.NewFlagSet("calcDebugResult", flag.ExitOnError)

	calcDebugResultOptions := initCalcDebugResultOptions(resultFlagSet)
	switch input[0] {
	case "enable":
		err = cli.enableCalcDebug()
	case "disable":
		err = cli.disableCalcDebug()
	case "add":
		if len(input) != 2 {
			goto INVALID
		}
		err = cli.addCalcDebuggingAddress(input[1])
	case "delete":
		if len(input) != 2 {
			goto INVALID
		}
		err = cli.deleteCalcDebuggingAddress(input[1])
	case "output":
		if len(input) != 2 {
			goto INVALID
		}
		err = cli.changeCalcDebugResultPath(input[1])
	case "list":
		err = cli.printCalcDebuggingAddresses()
	case "result":
		err = resultFlagSet.Parse(input[1:])
		validateInput(resultFlagSet, err, calcDebugResultOptions.help)
		err = cli.queryCalculationDebugResult(calcDebugResultOptions)
	default:
		goto INVALID
	}
	return err

INVALID:
	err = calcDebugUsage
	return err
}

func (cli *CLI) enableCalcDebug() error {
	var req core.DebugMessage
	req.Cmd = core.DebugCalcFlagOn

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}

func (cli *CLI) disableCalcDebug() error {
	var req core.DebugMessage
	req.Cmd = core.DebugCalcFlagOff

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}

func (cli *CLI) printCalcDebuggingAddresses() error {
	var req core.DebugMessage
	var resp core.ResponseCalcDebugAddressList
	req.Cmd = core.DebugCalcListAddresses
	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("Calculation debugging Addresses : \n%s\n", Display(resp.Addresses))
	}

	return err
}

func (cli *CLI) addCalcDebuggingAddress(address string) error {
	var req core.DebugMessage
	req.Cmd = core.DebugCalcAddAddress
	req.Address = *common.NewAddressFromString(address)

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}

func (cli *CLI) deleteCalcDebuggingAddress(address string) error {
	var req core.DebugMessage
	req.Cmd = core.DebugCalcDelAddress
	req.Address = *common.NewAddressFromString(address)

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}

func (cli *CLI) changeCalcDebugResultPath(path string) error {
	var req core.DebugMessage
	req.OutputPath = path

	return cli.conn.Send(core.MsgDebug, cli.id, req)
}

func (cli *CLI) queryCalculationDebugResult(options *Options) error {
	var req core.DebugMessage
	var resp core.ResponseQueryCalcDebugResult
	address := *common.NewAddressFromString(options.address)
	blockHeight := options.blockHeight

	req.Cmd = core.DebugCalcDebugResult
	req.Address = address
	req.BlockHeight = blockHeight

	err := cli.conn.SendAndReceive(core.MsgDebug, cli.id, req, &resp)
	if err == nil && resp.Results != nil {
		fmt.Printf("%s\n", Display(resp.Results))
	}

	return err
}
