package core

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
)

const (
	IPCVersion uint64 = 2

	MsgVersion              uint = 0
	MsgClaim                     = 1
	MsgQuery                     = 2
	MsgCalculate                 = 3
	MsgCommitBlock               = 4
	MsgCommitClaim               = 5
	MsgQueryCalculateStatus      = 6
	MsgQueryCalculateResult      = 7
	MsgRollBack                  = 8
	MsgINIT                      = 9

	MsgNotify        = 100
	MsgReady         = MsgNotify + 0
	MsgCalculateDone = MsgNotify + 1

	MsgDebug = 1000
)

func MsgToString(msg uint) string {
	switch msg {
	case MsgVersion:
		return "VERSION"
	case MsgClaim:
		return "CLAIM"
	case MsgQuery:
		return "QUERY"
	case MsgCalculate:
		return "CALCULATE"
	case MsgCommitBlock:
		return "COMMIT_BLOCK"
	case MsgCommitClaim:
		return "COMMIT_CLAIM"
	case MsgQueryCalculateStatus:
		return "QUERY_CALCULATE_STATUS"
	case MsgQueryCalculateResult:
		return "QUERY_CALCULATE_RESULT"
	case MsgReady:
		return "READY"
	case MsgCalculateDone:
		return "CALCULATE_DONE"
	case MsgRollBack:
		return "ROLLBACK"
	case MsgINIT:
		return "INIT"
	case MsgDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

func MsgDataToString(data interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		return "Can't covert Message to json"
	}
	return string(b)
}

type msgHandler struct {
	mgr  *manager
	conn ipc.Connection
}

func newConnection(m *manager, c ipc.Connection) (*msgHandler, error) {
	handler := &msgHandler{
		mgr:  m,
		conn: c,
	}

	c.SetHandler(MsgVersion, handler)
	c.SetHandler(MsgQuery, handler)
	c.SetHandler(MsgQueryCalculateStatus, handler)
	c.SetHandler(MsgQueryCalculateResult, handler)
	if m.monitorMode == true {
		c.SetHandler(MsgDebug, handler)
	} else {
		c.SetHandler(MsgClaim, handler)
		c.SetHandler(MsgCalculate, handler)
		c.SetHandler(MsgCommitBlock, handler)
		c.SetHandler(MsgCommitClaim, handler)
		c.SetHandler(MsgRollBack, handler)
		c.SetHandler(MsgINIT, handler)
	}

	// send READY message to peer
	cBI := handler.mgr.ctx.DB.getCurrentBlockInfo()
	err := sendVersion(c, MsgReady, 0, cBI.BlockHeight, cBI.BlockHash)
	if err != nil {
		log.Printf("Failed to send READY message")
	} else {
		log.Printf("Accept new connection and send READY message")
	}

	return handler, err
}

func (mh *msgHandler) HandleMessage(c ipc.Connection, msg uint, id uint32, data []byte) error {
	log.Printf("Get message. (msg:%s, id:%d)", MsgToString(msg), id)
	switch msg {
	case MsgVersion:
		go mh.version(c, id)
	case MsgClaim:
		go mh.claim(c, id, data)
	case MsgQuery:
		go mh.query(c, id, data)
	case MsgCalculate:
		go mh.calculate(c, id, data)
	case MsgCommitBlock:
		go mh.commitBlock(c, id, data)
	case MsgDebug:
		go mh.debug(c, id, data)
	case MsgCommitClaim:
		go mh.commitClaim(c, id, data)
	case MsgQueryCalculateStatus:
		go mh.queryCalculateStatus(c, id, data)
	case MsgQueryCalculateResult:
		go mh.queryCalculateResult(c, id, data)
	case MsgRollBack:
		// do not process other messages while process Rollback message
		return mh.rollback(c, id, data)
	case MsgINIT:
		go mh.init(c, id, data)
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
	return nil
}

type ResponseVersion struct {
	Version     uint64
	BlockHeight uint64
	BlockHash   [BlockHashSize]byte
}

func (rv *ResponseVersion) String() string {
	return fmt.Sprintf("Version: %d, BlockHeight: %d, BlockHash: %s",
		rv.Version, rv.BlockHeight, hex.EncodeToString(rv.BlockHash[:]))
}

func (mh *msgHandler) version(c ipc.Connection, id uint32) error {
	mh.mgr.IncreaseMessageTask()
	cBI := mh.mgr.ctx.DB.getCurrentBlockInfo()
	mh.mgr.DecreaseMessageTask()
	return sendVersion(c, MsgVersion, id, cBI.BlockHeight, cBI.BlockHash)
}

func sendVersion(c ipc.Connection, msg uint, id uint32, blockHeight uint64, blockHash [BlockHashSize]byte) error {
	resp := ResponseVersion{
		Version:     IPCVersion,
		BlockHeight: blockHeight,
		BlockHash:   blockHash,
	}

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(msg), id, resp.String())
	return c.Send(msg, id, resp)
}

type ResponseQuery struct {
	Address     common.Address
	IScore      common.HexInt
	BlockHeight uint64
}

func (rq *ResponseQuery) String() string {
	return fmt.Sprintf("Address: %s, IScore: %s, BlockHeight: %d",
		rq.Address.String(),
		rq.IScore.String(),
		rq.BlockHeight)
}

func (mh *msgHandler) query(c ipc.Connection, id uint32, data []byte) error {
	var addr common.Address
	mh.mgr.IncreaseMessageTask()
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}
	log.Printf("\t QUERY request: address: %s", addr.String())

	resp := DoQuery(mh.mgr.ctx, addr)

	mh.mgr.DecreaseMessageTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgQuery), id, resp.String())
	return c.Send(MsgQuery, id, &resp)
}

func DoQuery(ctx *Context, addr common.Address) *ResponseQuery {
	var claim *Claim = nil
	var ia *IScoreAccount = nil
	isDB := ctx.DB

	// make response
	var resp ResponseQuery
	resp.Address = addr

	// read from claim DB
	cDB := isDB.getClaimDB()
	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(addr.Bytes())
	if bs != nil {
		claim, _ = NewClaimFromBytes(bs)
	}

	// read from Query DB
	qDB := isDB.getQueryDB(addr)
	bucket, _ = qDB.GetBucket(db.PrefixIScore)
	bs, _ = bucket.Get(addr.Bytes())
	if bs != nil {
		ia, _ = NewIScoreAccountFromBytes(bs)
		resp.BlockHeight = ia.BlockHeight
	} else {
		// No Info. about account
		return &resp
	}

	if claim != nil {
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.Data.IScore.Int)
	}

	// set calculated I-Score to response
	resp.IScore.Set(&ia.IScore.Int)

	return &resp
}

type ResponseInit struct {
	Success     bool
	BlockHeight uint64
}

func (resp *ResponseInit) String() string {
	return fmt.Sprintf("Success: %s, BlockHeight: %d", strconv.FormatBool(resp.Success), resp.BlockHeight)
}

func (mh *msgHandler) init(c ipc.Connection, id uint32, data []byte) error {
	var blockHeight uint64
	mh.mgr.IncreaseMessageTask()
	if _, err := codec.MP.UnmarshalFromBytes(data, &blockHeight); err != nil {
		return err
	}
	log.Printf("\t %s request: block height : %d", MsgToString(MsgINIT), blockHeight)

	resp := ResponseInit{true, blockHeight}
	err := DoInit(mh.mgr.ctx, blockHeight)
	if err != nil {
		log.Printf("Failed to INIT. %v", err)
		resp.Success = false
	}

	mh.mgr.DecreaseMessageTask()
	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgINIT), id, resp.String())
	return c.Send(MsgINIT, id, &resp)
}

func DoInit(ctx *Context, blockHeight uint64) error {
	currentBlockHeight := ctx.DB.getCurrentBlockInfo().BlockHeight
	if blockHeight > currentBlockHeight+1 {
		return fmt.Errorf("too high block height %d > %d", blockHeight, currentBlockHeight+1)
	}
	return initPreCommit(ctx.DB.getPreCommitDB(), blockHeight)
}
