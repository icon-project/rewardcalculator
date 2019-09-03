package core

import (
	"encoding/json"
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
	"log"
)

const (
	MsgVersion     uint = 0
	MsgClaim            = 1
	MsgQuery            = 2
	MsgCalculate        = 3
	MsgCommitBlock      = 4
	MsgCommitClaim      = 5
	MsgQueryCalculateStatus = 6

	MsgNotify           = 100
	MsgReady            = MsgNotify + 0
	MsgCalculateDone    = MsgNotify + 1

	MsgDebug            = 1000
)

func MsgToString(msg uint) string{
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
	case MsgReady:
		return "READY"
	case MsgCalculateDone:
		return "CALCULATE_DONE"
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
		mgr: m,
		conn: c,
	}

	c.SetHandler(MsgVersion, handler)
	c.SetHandler(MsgQuery, handler)
	c.SetHandler(MsgQueryCalculateStatus, handler)
	if m.monitorMode == true {
		c.SetHandler(MsgDebug, handler)
	} else {
		c.SetHandler(MsgClaim, handler)
		c.SetHandler(MsgCalculate, handler)
		c.SetHandler(MsgCommitBlock, handler)
		c.SetHandler(MsgCommitClaim, handler)
	}

	// send IISS data reload result
	err := sendReloadIISSDataResult(m.ctx, c)
	if err != nil {
		log.Printf("Failed to send IISSData reload result")
		return nil, err
	}

	// send READY message to peer
	err = sendVersion(c, MsgReady, 0, handler.mgr.ctx.DB.info.BlockHeight)
	if err != nil {
		log.Printf("Failed to send READY message")
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
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
	return nil
}

type ResponseVersion struct {
	Version uint64
	BlockHeight uint64
}

func (mh *msgHandler) version(c ipc.Connection, id uint32) error {
	return sendVersion(c, MsgVersion, id, mh.mgr.ctx.DB.info.BlockHeight)
}

func sendVersion(c ipc.Connection, msg uint, id uint32, blockHeight uint64) error {
	resp := ResponseVersion{
		Version: Version,
		BlockHeight: blockHeight,
	}

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(msg), id, MsgDataToString(resp))
	return c.Send(msg, id, resp)
}

type ResponseQuery struct {
	Address common.Address
	IScore  common.HexInt
	BlockHeight uint64
}

func (mh *msgHandler) query(c ipc.Connection, id uint32, data []byte) error {
	var addr common.Address
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}
	log.Printf("\t QUERY request: address: %s", addr.String())

	resp := DoQuery(mh.mgr.ctx, addr)

	log.Printf("Send message. (msg:%s, id:%d, data:%s)", MsgToString(MsgQuery), id, MsgDataToString(resp))
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
