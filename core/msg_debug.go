package core

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"log"

	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const (
	DebugStatistics    uint64 = 0
	DebugDBInfo        uint64 = 1
	DebugPRep          uint64 = 2
	DebugPRepCandidate uint64 = 3
	DebugGV            uint64 = 4

	DebugLogCTX uint64 = 100

	DebugCalc              uint64 = 200
	DebugCalcFlagOn               = DebugCalc
	DebugCalcFlagOff              = DebugCalc + 1
	DebugCalcAddAddress           = DebugCalc + 2
	DebugCalcDelAddress           = DebugCalc + 3
	DebugCalcListAddresses        = DebugCalc + 4
	DebugCalcOutputPath           = DebugCalc + 5
)

type DebugMessage struct {
	Cmd uint64
	MessageData
}

type MessageData struct {
	Address    common.Address
	OutputPath string
}

func (mh *msgHandler) debug(c ipc.Connection, id uint32, data []byte) error {
	var req DebugMessage
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		log.Printf("Failed to deserialize DEBUG message. err=%+v", err)
		return err
	}
	log.Printf("\t DEBUG request: %s", MsgDataToString(req))

	ctx := mh.mgr.ctx

	switch req.Cmd {
	case DebugStatistics:
		return handleStats(c, id, ctx)
	case DebugDBInfo:
		return handleDBInfo(c, id, ctx)
	case DebugPRep:
		return handlePRep(c, id, ctx)
	case DebugPRepCandidate:
		return handlePRepCandidate(c, id, ctx)
	case DebugGV:
		return handleGV(c, id, ctx)
	case DebugLogCTX:
		ctx.Print()
	case DebugCalcFlagOn:
		return handleCalcDebugFlagOn(c, id, ctx)
	case DebugCalcFlagOff:
		return handleCalcDebugFlagOff(c, id, ctx)
	case DebugCalcAddAddress:
		return handleCalcDebugAddAddress(c, id, ctx, req.Address)
	case DebugCalcDelAddress:
		return handleCalcDebugDeleteAddress(c, id, ctx, req.Address)
	case DebugCalcListAddresses:
		return handleCalcDebugAddresses(c, id, ctx)
	case DebugCalcOutputPath:
		return handleCalcResultOutput(c, id, ctx, req.OutputPath)
	}

	return fmt.Errorf("unknown debug message %d", req.Cmd)
}

type ResponseDebugStats struct {
	DebugMessage
	BlockHeight uint64
	Stats       Statistics
}

func handleStats(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseDebugStats
	resp.Cmd = DebugStatistics
	resp.BlockHeight = ctx.DB.getCalcDoneBH()
	if ctx.stats != nil {
		resp.Stats = *ctx.stats
	}

	return c.Send(MsgDebug, id, &resp)
}

type ResponseDebugDBInfo struct {
	DebugMessage
	DBInfo DBInfo
}

func handleDBInfo(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseDebugDBInfo
	resp.Cmd = DebugDBInfo
	if ctx.DB.info != nil {
		resp.DBInfo = *ctx.DB.info
	}

	return c.Send(MsgDebug, id, &resp)
}

type ResponseDebugPRep struct {
	DebugMessage
	PReps []PRep
}

func handlePRep(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseDebugPRep
	resp.Cmd = DebugPRep
	resp.PReps = make([]PRep, len(ctx.PRep))
	for i, p := range ctx.PRep {
		resp.PReps[i] = *p
	}

	return c.Send(MsgDebug, id, &resp)
}

type ResponseDebugPRepCandidate struct {
	DebugMessage
	PRepCandidates []PRepCandidate
}

func handlePRepCandidate(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseDebugPRepCandidate
	resp.Cmd = DebugPRepCandidate
	resp.PRepCandidates = make([]PRepCandidate, len(ctx.PRepCandidates))
	i := 0
	for _, p := range ctx.PRepCandidates {
		resp.PRepCandidates[i] = *p
		i++
	}

	return c.Send(MsgDebug, id, &resp)
}

type ResponseDebugGV struct {
	DebugMessage
	GV []GovernanceVariable
}

func handleGV(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseDebugGV
	resp.Cmd = DebugGV
	if len(ctx.GV) > 0 {
		resp.GV = make([]GovernanceVariable, len(ctx.GV))
		for i, p := range ctx.GV {
			resp.GV[i] = *p
		}
	} else {
		resp.GV = nil
	}

	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugFlagOn(c ipc.Connection, id uint32, ctx *Context) error {
	var resp DebugMessage
	resp.Cmd = DebugCalcFlagOn
	ctx.calcDebugConf.Flag = true
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugFlagOff(c ipc.Connection, id uint32, ctx *Context) error {
	var resp DebugMessage
	resp.Cmd = DebugCalcFlagOff
	ctx.calcDebugConf.Flag = false
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugAddAddress(c ipc.Connection, id uint32, ctx *Context, address common.Address) error {
	var resp DebugMessage
	resp.Cmd = DebugCalcAddAddress
	resp.Address = address
	addDebuggingAddress(ctx, address)
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugDeleteAddress(c ipc.Connection, id uint32, ctx *Context, address common.Address) error {
	var resp DebugMessage
	resp.Cmd = DebugCalcDelAddress
	resp.Address = address
	deleteDebuggingAddress(ctx, address)
	return c.Send(MsgDebug, id, &resp)
}

type ResponseCalcDebugAddressList struct {
	DebugMessage
	Addresses []*common.Address
}

func handleCalcDebugAddresses(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseCalcDebugAddressList
	resp.Cmd = DebugCalcListAddresses
	resp.Addresses = ctx.calcDebugConf.Addresses
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcResultOutput(c ipc.Connection, id uint32, ctx *Context, output string) error {
	var resp DebugMessage
	ctx.calcDebugConf.Output = output
	resp.OutputPath = output
	return c.Send(MsgDebug, id, &resp)
}
