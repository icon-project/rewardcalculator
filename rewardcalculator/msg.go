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

type VersionMessage struct {
	Success     bool
	BlockHeight uint64
}

type QueryMessage struct {
	Address     common.Address
}

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
	rc := &msgHandler{
		mgr: m,
		conn: c,
	}

	c.SetHandler(msgVERSION, rc)
	c.SetHandler(msgClaim, rc)
	c.SetHandler(msgQuery, rc)
	c.SetHandler(msgCalculate, rc)
	c.SetHandler(msgCommitBlock, rc)

	return rc, nil
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
	var req VersionMessage
	req.Success = true
	req.BlockHeight = mh.mgr.ctx.db.info.BlockHeight

	mh.mgr.ctx.Print()

	return c.Send(msgVERSION, id, &req)
}

func (mh *msgHandler) query(c ipc.Connection, id uint32, data []byte) error {
	var addr common.Address
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}

	var claim *Claim = nil
	var ia *IScoreAccount = nil
	ctx := mh.mgr.ctx
	isDB := ctx.db

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
		return c.Send(msgQuery, id, &resp)
	}

	if claim != nil {
		if ia.BlockHeight == claim.BlockHeight {
			// already claimed in current period
			return c.Send(msgQuery, id, &resp)
		}
		// subtract claimed I-Score
		ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
	}

	// set calculated I-Score to response
	resp.IScore = ia.IScore

	return c.Send(msgQuery, id, &resp)
}
