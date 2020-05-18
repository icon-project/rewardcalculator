package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/db"
	"log"

	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const (
	DebugStatistics      uint64 = 0
	DebugDBInfo          uint64 = 1
	DebugPRep            uint64 = 2
	DebugPRepCandidate   uint64 = 3
	DebugGV              uint64 = 4
	DebugCalcDebugResult uint64 = 5

	DebugLogCTX uint64 = 100

	DebugCalc              uint64 = 200
	DebugCalcFlagOn               = DebugCalc
	DebugCalcFlagOff              = DebugCalc + 1
	DebugCalcAddAddress           = DebugCalc + 2
	DebugCalcDelAddress           = DebugCalc + 3
	DebugCalcListAddresses        = DebugCalc + 4
)

type DebugMessage struct {
	Cmd uint64
	MessageData
}

type MessageData struct {
	Address     common.Address
	OutputPath  string
	BlockHeight uint64
}

func (mh *msgHandler) debug(c ipc.Connection, id uint32, data []byte) error {
	var req DebugMessage
	var result error
	mh.mgr.AddMsgTask()
	if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
		log.Printf("Failed to deserialize DEBUG message. err=%+v", err)
		return err
	}
	log.Printf("\t DEBUG request: %s", MsgDataToString(req))

	ctx := mh.mgr.ctx

	switch req.Cmd {
	case DebugStatistics:
		result = handleStats(c, id, ctx)
	case DebugDBInfo:
		result = handleDBInfo(c, id, ctx)
	case DebugPRep:
		result = handlePRep(c, id, ctx)
	case DebugPRepCandidate:
		result = handlePRepCandidate(c, id, ctx)
	case DebugGV:
		result = handleGV(c, id, ctx)
	case DebugLogCTX:
		ctx.Print()
		result = nil
	case DebugCalcFlagOn:
		result = handleCalcDebugFlagOn(c, id, ctx)
	case DebugCalcFlagOff:
		result = handleCalcDebugFlagOff(c, id, ctx)
	case DebugCalcAddAddress:
		result = handleCalcDebugAddAddress(c, id, ctx, req.Address)
	case DebugCalcDelAddress:
		result = handleCalcDebugDeleteAddress(c, id, ctx, req.Address)
	case DebugCalcListAddresses:
		result = handleCalcDebugAddresses(c, id, ctx)
	case DebugCalcDebugResult:
		result = handleQueryCalcDebugResult(c, id, ctx, req.Address, req.BlockHeight)
	default:
		result = fmt.Errorf("unknown debug message %d", req.Cmd)
	}

	mh.mgr.DoneMsgTask()
	return result
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

type ResponseCalcDebug struct {
	Success bool
	MessageData
}

func handleCalcDebugFlagOn(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseCalcDebug
	if ctx.DB.isCalculating() {
		resp.Success = false
	} else {
		resp.Success = true
	}
	ctx.calcDebug.conf.Flag = true
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugFlagOff(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseCalcDebug
	if ctx.DB.isCalculating() {
		resp.Success = false
	} else {
		resp.Success = true
	}
	ctx.calcDebug.conf.Flag = false
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugAddAddress(c ipc.Connection, id uint32, ctx *Context, address common.Address) error {
	var resp ResponseCalcDebug
	if ctx.DB.isCalculating() {
		resp.Success = false
	} else {
		resp.Success = true
	}
	resp.Address = address
	AddDebuggingAddress(ctx, address)
	return c.Send(MsgDebug, id, &resp)
}

func handleCalcDebugDeleteAddress(c ipc.Connection, id uint32, ctx *Context, address common.Address) error {
	var resp ResponseCalcDebug
	if ctx.DB.isCalculating() {
		resp.Success = false
	} else {
		resp.Success = true
	}
	resp.Address = address
	DeleteDebuggingAddress(ctx, address)
	return c.Send(MsgDebug, id, &resp)
}

type ResponseCalcDebugAddressList struct {
	DebugMessage
	Addresses []*common.Address
}

func handleCalcDebugAddresses(c ipc.Connection, id uint32, ctx *Context) error {
	var resp ResponseCalcDebugAddressList
	resp.Cmd = DebugCalcListAddresses
	resp.Addresses = ctx.calcDebug.conf.Addresses
	return c.Send(MsgDebug, id, &resp)
}

type ResponseQueryCalcDebugResult struct {
	Results []*CalcDebugResult
}

func handleQueryCalcDebugResult(c ipc.Connection, id uint32, ctx *Context,
	address common.Address, blockHeight uint64) error {

	var resp ResponseQueryCalcDebugResult

	calcDebugDB := db.Open(ctx.DB.info.DBRoot, string(db.GoLevelDBBackend), "calculation_debug")
	defer calcDebugDB.Close()
	CalcDebugKeys, err := GetCalcDebugResultKeys(calcDebugDB, blockHeight)
	if err != nil {
		return c.Send(MsgDebug, id, &resp)
	}
	bucket, err := calcDebugDB.GetBucket(db.PrefixClaim)
	if err != nil {
		return c.Send(MsgDebug, id, &resp)
	}

	nilAddress := new(common.Address)
	for _, key := range CalcDebugKeys {
		value, err := bucket.Get(key)
		if err != nil {
			return c.Send(MsgDebug, id, new(ResponseQueryCalcDebugResult))
		}
		if value == nil {
			continue
		}
		dr, err := NewCalcDebugResult(key, value)
		if err != nil {
			return err
		} else {
			for _, calcResult := range dr.Results {
				if address.Equal(nilAddress) {
					resp.Results = append(resp.Results, dr)
				} else if address.Equal(calcResult.Address) {
					resp.Results = append(resp.Results, dr)
				}
			}
		}
	}
	return c.Send(MsgDebug, id, &resp)
}

func GetCalcDebugResultKeys(qdb db.Database, blockHeight uint64) ([][]byte, error) {
	iter, err := qdb.GetIterator()
	if err != nil {
		return nil, err
	}

	cDebugResultKeys := make([][]byte, 0)
	iter.New(nil, nil)
	keyExist := false
	blockHeightBytesValue := common.Uint64ToBytes(blockHeight)
	for iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		if bytes.Equal(key[BlockHeightSize-len(blockHeightBytesValue):BlockHeightSize], blockHeightBytesValue) {
			keyExist = true
			cDebugResultKeys = append(cDebugResultKeys, key)
		}
	}
	iter.Release()

	if keyExist == false {
		return nil, errors.New("calcDebugResult key does not exist")
	}
	err = iter.Error()
	if err != nil {
		return nil, err
	}

	return cDebugResultKeys, err
}

func NewCalcDebugResult(key []byte, value []byte) (*CalcDebugResult, error) {
	dr := new(CalcDebugResult)

	err := dr.SetBytes(value)
	if err != nil {
		return nil, err

	}
	dr.BlockHeight = common.BytesToUint64(key[:BlockHeightSize])
	blockHash := make([]byte, BlockHashSize)
	copy(blockHash, key[BlockHeightSize:BlockHeightSize+BlockHashSize])
	dr.BlockHash = "0x" + hex.EncodeToString(blockHash)
	return dr, nil
}
