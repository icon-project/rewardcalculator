package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/codec"
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
	claimMap map[common.Address]uint64
}


type rewardCalculate struct {
	mgr  *manager
	conn ipc.Connection

	claimLock        sync.RWMutex
	preCommitMapList []*preCommitMap
	claimMap         map[common.Address]uint64	// value : block height
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
		var m VersionMessage
		m.Success = true
		m.BlockHeight = rc.mgr.gOpts.BlockHeight

		return rc.conn.Send(msg, &m)
	case msgClaim:
		go rc.claim(c, data)
		return nil
	case msgQuery:
		go rc.query(c, data)
		return nil
	case msgCalculate:
		return rc.calculate(c, data)
	case msgCommitBlock:
		return rc.commitBlock(c, data)
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
}

func (rc *rewardCalculate) query(c ipc.Connection, data []byte) error {
	var addr common.Address
	if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
		return err
	}
	rc.claimLock.RLock()
	blockHeight := rc.claimMap[addr]
	rc.claimLock.RUnlock()

	var resp ResponseQuery
	resp.Address = addr

	if blockHeight != 0 {
		// from claimMap
		resp.BlockHeight = blockHeight
		resp.IScore.SetUint64(0)
	} else {
		// from Account DB
		opts := rc.mgr.gOpts
		isDB := opts.db

		// read from account DB snapshot
		isDB.snapshotLock.RLock()
		snapshot := opts.GetAccountDBSnapshot(addr)
		bs, _ := snapshot.Get(addr.Bytes())
		isDB.snapshotLock.RUnlock()

		ia := new(IScoreAccount)
		if bs != nil {
			ia.SetBytes(bs)
		}

		resp.BlockHeight = ia.BlockHeight
		resp.IScore = ia.IScore
	}

	return c.Send(msgQuery, &resp)
}
