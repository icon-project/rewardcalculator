package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
)

const (
	msgVERSION   uint = 0
	msgClaim          = 1
	msgQuery          = 2
	msgCalculate      = 3
	msgCommitBlock    = 4
)

type ResponseQuery struct {
	Address common.Address
	IScore  common.HexInt
	BlockHeight uint64
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

	c.SetHandler(msgVERSION, handler)
	c.SetHandler(msgClaim, handler)
	c.SetHandler(msgQuery, handler)
	c.SetHandler(msgCalculate, handler)
	c.SetHandler(msgCommitBlock, handler)

	return handler, nil
}

func (mh *msgHandler) HandleMessage(c ipc.Connection, msg uint, id uint32, data []byte) error {
	switch msg {
	case msgVERSION:
		go mh.version(c, id, data)
	case msgClaim:
		go mh.claim(c, id, data)
	case msgQuery:
		go mh.query(c, id, data)
	case msgCalculate:
		go mh.calculate(c, id, data)
	case msgCommitBlock:
		go mh.commitBlock(c, id, data)
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
	return nil
}

func (mh *msgHandler) version(c ipc.Connection, id uint32, data []byte) error {
	mh.mgr.ctx.Print()

	return c.Send(msgVERSION, id, Version)
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
		if ia.BlockHeight == claim.BlockHeight {
			// already claimed in current period
			return &resp
		}
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
	}

	// set calculated I-Score to response
	resp.IScore.Set(&ia.IScore.Int)

	return &resp
}
