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

func (mh *msgHandler) handleCalcDebugFlag(c ipc.Connection, id uint32, data []byte) error {
	var req RequestCalcDebugFlag
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	ctx := mh.mgr.ctx

	switch req.Cmd {
	case CalcDebugOn:
		ctx.calcDebugConf.Flag = true
	case CalcDebugOff:
		ctx.calcDebugConf.Flag = false
	default:
		return fmt.Errorf("invalid command")
	}

	return c.Send(MsgCalcDebugFlag, id, req)
}

type RequestCalcDebugAddress struct {
	Cmd     uint64
	Address common.Address
}

const (
	AddDebuggingAddress    uint64 = 0
	DeleteDebuggingAddress uint64 = 1
)

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
	} else {
		return fmt.Errorf("invalid command")
	}
	return c.Send(MsgCalcDebugAddress, id, req)
}

type ResponseCalcDebugAddressList struct {
	Addresses []*common.Address
}

func (mh *msgHandler) handleCalcDebugAddresses(c ipc.Connection, id uint32) error {
	var resp ResponseCalcDebugAddressList
	ctx := mh.mgr.ctx
	resp.Addresses = ctx.calcDebugConf.Addresses
	return c.Send(MsgCalcDebugAddresses, id, &resp)
}

type RequestCalcResultOutput struct {
	Path string
}

func (mh *msgHandler) handleCalcResultOutput(c ipc.Connection, id uint32, data []byte) error {
	var req RequestCalcResultOutput
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		return err
	}
	ctx := mh.mgr.ctx
	ctx.calcDebugConf.Output = req.Path
	return c.Send(MsgCalcDebugOutput, id, req)
}
