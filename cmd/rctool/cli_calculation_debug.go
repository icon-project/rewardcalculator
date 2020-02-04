package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) calculateDebug(input []string) error {
	calcDebugUsage := fmt.Errorf("Commands\n" +
		"\t enable \t enable calculation debug\n" +
		"\t disable \t disable calculation debug\n" +
		"\t add <Address> \t add calculation debugging address\n" +
		"\t delete <Address> \t delete calculation debugging address\n" +
		"\t output <outputPath> \t change calculation debugging output path\n" +
		"\t list \t print calculation debugging addresses\n")

	if len(input) == 0 {
		return calcDebugUsage
	}
	var err error
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
	default:
		goto INVALID
	}
	return err

INVALID:
	err = calcDebugUsage
	return err
}

func (cli *CLI) enableCalcDebug() error {
	var req core.RequestCalcDebugFlag
	req.Cmd = core.CalcDebugOn

	return cli.conn.Send(core.MsgCalcDebugFlag, cli.id, req)
}

func (cli *CLI) disableCalcDebug() error {
	var req core.RequestCalcDebugFlag
	req.Cmd = core.CalcDebugOff

	return cli.conn.Send(core.MsgCalcDebugFlag, cli.id, req)
}

func (cli *CLI) printCalcDebuggingAddresses() error {
	var resp core.ResponseCalcDebugAddressList

	err := cli.conn.SendAndReceive(core.MsgCalcDebugAddresses, cli.id, nil, &resp)
	if err == nil {
		fmt.Printf("Calculation debugging Addresses : \n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) addCalcDebuggingAddress(address string) error {
	var req core.RequestCalcDebugAddress
	req.Cmd = core.AddDebuggingAddress
	req.Address = *common.NewAddressFromString(address)

	return cli.conn.Send(core.MsgCalcDebugAddress, cli.id, req)
}

func (cli *CLI) deleteCalcDebuggingAddress(address string) error {
	var req core.RequestCalcDebugAddress
	req.Cmd = core.DeleteDebuggingAddress
	req.Address = *common.NewAddressFromString(address)

	return cli.conn.Send(core.MsgCalcDebugAddress, cli.id, req)
}

func (cli *CLI) changeCalcDebugResultPath(path string) error {
	var req core.RequestCalcResultOutput
	req.Path = path

	return cli.conn.Send(core.MsgCalcDebugOutput, cli.id, req)
}
