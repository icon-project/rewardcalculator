package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
	"sync"
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

type preCommitMap struct {
	BlockHeight uint64
	BlockHash   []byte
	claimMap map[common.Address]*Claim
}

type rewardCalculate struct {
	mgr  *manager
	conn ipc.Connection

	claimLock sync.RWMutex
	preCommitMapList []*preCommitMap
}

func newConnection(m *manager, c ipc.Connection) (*rewardCalculate, error) {
	rc := &rewardCalculate{
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

func (rc *rewardCalculate) HandleMessage(c ipc.Connection, msg uint, id uint32, data []byte) error {
	switch msg {
	case msgVERSION:
		go rc.version(c, id, data)
	case msgClaim:
		go rc.claim(c, id, data)
	case msgQuery:
		go rc.query(c, id, data)
	case msgCalculate:
		go rc.calculate(c, id, data)
	case msgCommitBlock:
		go rc.commitBlock(c, id, data)
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
	return nil
}

func (rc *rewardCalculate) version(c ipc.Connection, id uint32, data []byte) error {
	var req VersionMessage
	req.Success = true
	req.BlockHeight = rc.mgr.gOpts.db.info.BlockHeight

	rc.mgr.gOpts.Print()

	return rc.conn.Send(msgVERSION, id, &req)
}

func (rc *rewardCalculate) query(c ipc.Connection, id uint32, data []byte) error {
	var addr common.Address
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}

	var claim *Claim = nil
	var ia *IScoreAccount = nil
	opts := rc.mgr.gOpts
	isDB := opts.db

	// make response
	var resp ResponseQuery
	resp.Address = addr

	// read from claim DB
	cDB := isDB.GetClaimDB()
	bucket, _ := cDB.GetBucket(db.PrefixIScore)
	bs, _ := bucket.Get(addr.Bytes())
	if bs != nil {
		claim, _ = NewClaimFromBytes(bs)
	}

	// read from account query DB
	aDB := isDB.GetQueryDB(addr)
	bucket, _ = aDB.GetBucket(db.PrefixIScore)
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
