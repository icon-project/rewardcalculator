package core

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

type RequestCalcDebugFlag struct {
	Cmd uint64
}

const (
	CalcDebugOn  uint64 = 0
	CalcDebugOff uint64 = 1
)

type ResponseCalcDebugFlag struct {
	cmd  uint64
	flag bool
}

func (mh *msgHandler) handleCalcDebugFlag(c ipc.Connection, id uint32, data []byte) error {
	var req RequestCalcDebugFlag
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	ctx := mh.mgr.ctx

	switch req.Cmd {
	case CalcDebugOn:
		ctx.calculationDebugFlag = true
	case CalcDebugOff:
		ctx.calculationDebugFlag = false
	default:
		return fmt.Errorf("invalid command")
	}
	var resp ResponseCalcDebugFlag
	resp.cmd = req.Cmd
	resp.flag = ctx.calculationDebugFlag

	return c.Send(MsgCalcDebugFlag, id, &resp)
}

type RequestCalcDebugAddress struct {
	Cmd     uint64
	Address common.Address
}

const (
	AddDebuggingAddress    uint64 = 0
	DeleteDebuggingAddress uint64 = 1
)

type ResponseCalcDebugAddress struct {
	cmd       uint64
	addresses []*common.Address
}

func (mh *msgHandler) handleCalcDebugAddress(c ipc.Connection, id uint32, data []byte) error {
	var req RequestCalcDebugAddress
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	ctx := mh.mgr.ctx
	if req.Cmd == AddDebuggingAddress {
		addDebuggingAddress(ctx, req.Address)
	} else if req.Cmd == DeleteDebuggingAddress {
		deleteDebuggingAddress(ctx, req.Address)
	}
	var resp ResponseCalcDebugAddress
	resp.cmd = req.Cmd
	resp.addresses = ctx.debugCalculationAddresses
	return c.Send(MsgCalcDebugAddress, id, &resp)
}

type ResponseCalcDebugAddressList struct {
	addresses []*common.Address
}

func (mh *msgHandler) handleCalcDebugAddresses(c ipc.Connection, id uint32) error {
	var resp ResponseCalcDebugAddressList
	ctx := mh.mgr.ctx
	resp.addresses = ctx.debugCalculationAddresses
	return c.Send(MsgCalcDebugAddresses, id, &resp)
}

type RequestCalcResultOutput struct {
	Path string
}

type ResponseCalcResultOutput struct {
	path string
}

func (mh *msgHandler) handleCalcResultOutput(c ipc.Connection, id uint32, data []byte) error {
	var req RequestCalcResultOutput
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	var resp ResponseCalcResultOutput
	ctx := mh.mgr.ctx
	ctx.debuggingOutputPath = req.Path
	resp.path = ctx.debuggingOutputPath
	return c.Send(MsgCalcDebugOutput, id, &resp)
}
