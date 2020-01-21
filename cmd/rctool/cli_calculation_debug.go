package main

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/core"
)

func (cli *CLI) calculateDebug(input []string) error {
	errMsg := fmt.Errorf("invalid input")
	if len(input) == 0 {
		printUsage()
		return errMsg
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
	default:
		goto INVALID
	}
	return err

INVALID:
	printUsage()
	return errMsg
}

func printUsage() {
	fmt.Printf("Commands\n")
	fmt.Printf("enable\n")
	fmt.Printf("disable\n")
	fmt.Printf("add <Address>\n")
	fmt.Printf("delete <address>\n")
	fmt.Printf("output <new outputPath>\n")
}

func (cli *CLI) enableCalcDebug() error {
	var req core.RequestCalcDebugFlag
	req.Cmd = core.CalcDebugOn
	var resp core.ResponseCalcDebugFlag

	err := cli.conn.SendAndReceive(core.MsgCalcDebugFlag, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("calculate_dbug enable command got response: \n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) disableCalcDebug() error {
	var req core.RequestCalcDebugFlag
	req.Cmd = core.CalcDebugOff
	var resp core.ResponseCalcDebugFlag

	err := cli.conn.SendAndReceive(core.MsgCalcDebugFlag, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("calculate_dbug disable command got response: \n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) addCalcDebuggingAddress(address string) error {
	var req core.RequestCalcDebugAddress
	req.Cmd = core.AddDebuggingAddress
	req.Address = *common.NewAddressFromString(address)
	var resp core.ResponseCalcDebugAddress

	err := cli.conn.SendAndReceive(core.MsgCalcDebugAddress, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("calculate_dbug add command got response: \n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) deleteCalcDebuggingAddress(address string) error {
	var req core.RequestCalcDebugAddress
	req.Cmd = core.DeleteDebuggingAddress
	req.Address = *common.NewAddressFromString(address)
	var resp core.ResponseCalcDebugAddress

	err := cli.conn.SendAndReceive(core.MsgCalcDebugAddress, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("calculate_dbug Delete command got response: \n%s\n", Display(resp))
	}

	return err
}

func (cli *CLI) changeCalcDebugResultPath(path string) error {
	var req core.RequestCalcResultOutput
	req.Path = path
	var resp core.ResponseCalcResultOutput

	err := cli.conn.SendAndReceive(core.MsgCalcDebugOutput, cli.id, req, &resp)
	if err == nil {
		fmt.Printf("calculate_dbug disable command got response: \n%s\n", Display(resp))
	}

	return err
}
