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

func (rc *rewardCalculate) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case msgVERSION:
		go rc.version(c, data)
	case msgClaim:
		go rc.claim(c, data)
	case msgQuery:
		go rc.query(c, data)
	case msgCalculate:
		return rc.calculate(c, data)
	case msgCommitBlock:
		go rc.commitBlock(c, data)
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
	return nil
}

func (rc *rewardCalculate) version(c ipc.Connection, data []byte) error {
	var req VersionMessage
	req.Success = true
	req.BlockHeight = rc.mgr.gOpts.db.info.BlockHeight

	rc.mgr.gOpts.Print()

	return rc.conn.Send(msgVERSION, &req)
}

func (rc *rewardCalculate) query(c ipc.Connection, data []byte) error {
	var addr common.Address
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}

	ia := new(IScoreAccount)
	claim := new(Claim)
	opts := rc.mgr.gOpts
	isDB := opts.db

	// read from claim DB
	cDB := isDB.GetClaimDB()
	bucket, err := cDB.GetBucket(db.PrefixIScore)
	bs, err := bucket.Get(addr.Bytes())
	if bs != nil {
		claim.SetBytes(bs)
	} else {
		claim = nil
	}

	// read from account query DB
	aDB := isDB.GetQueryDB(addr)
	bucket, err = aDB.GetBucket(db.PrefixIScore)
	bs, err = bucket.Get(addr.Bytes())
	if err != nil {
		return err
	}
	if bs != nil {
		ia.SetBytes(bs)
		if nil != claim && ia.BlockHeight >= claim.BlockHeight {
			ia.IScore.Sub(&ia.IScore.Int, &claim.IScore.Int)
		}
	}

	// make response
	var resp ResponseQuery
	resp.Address = addr
	resp.BlockHeight = ia.BlockHeight
	resp.IScore = ia.IScore

	return c.Send(msgQuery, &resp)
}
