package core

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
	"log"
)

const (
	msgVERSION     uint = 0
	msgClaim            = 1
	msgQuery            = 2
	msgCalculate        = 3
	msgCommitBlock      = 4
	MsgDebug            = 100
)

type msgHandler struct {
	mgr  *manager
	conn ipc.Connection
}

func newConnection(m *manager, c ipc.Connection) (*msgHandler, error) {
	handler := &msgHandler{
		mgr: m,
		conn: c,
	}

	c.SetHandler(msgVERSION, handler)
	c.SetHandler(msgQuery, handler)
	if m.monitorMode == true {
		c.SetHandler(MsgDebug, handler)
	} else {
		c.SetHandler(msgClaim, handler)
		c.SetHandler(msgCalculate, handler)
		c.SetHandler(msgCommitBlock, handler)
	}

	// send IISS data reload result
	err := sendReloadIISSDataResult(m.ctx, c)
	if err != nil {
		log.Printf("Failed to send IISSData reload result")
		return nil, err
	}

	// send VERSION message to peer
	err = handler.version(c, 0)
	if err != nil {
		log.Printf("Failed to send VERSION messag")
	}

	return handler, err
}

func (mh *msgHandler) HandleMessage(c ipc.Connection, msg uint, id uint32, data []byte) error {
	switch msg {
	case msgVERSION:
		go mh.version(c, id)
	case msgClaim:
		go mh.claim(c, id, data)
	case msgQuery:
		go mh.query(c, id, data)
	case msgCalculate:
		go mh.calculate(c, id, data)
	case msgCommitBlock:
		go mh.commitBlock(c, id, data)
	case MsgDebug:
		go mh.debug(c, id, data)
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
	resp := ResponseVersion{
		Version: Version,
		BlockHeight: mh.mgr.ctx.DB.info.BlockHeight,
	}

	return c.Send(msgVERSION, 0, resp)
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

	resp := DoQuery(mh.mgr.ctx, addr)
	return c.Send(msgQuery, id, &resp)
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
		ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
	}

	// set calculated I-Score to response
	resp.IScore.Set(&ia.IScore.Int)

	return &resp
}
