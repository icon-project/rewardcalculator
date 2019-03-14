package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common"
	"log"
	"sync"

	"github.com/icon-project/rewardcalculator/common/codec"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/pkg/errors"
)

const (
	msgVERSION   uint = 0
	msgClaim          = 1
	msgQuery          = 2
	msgCalculate      = 3
)

type VersionMessage struct {
	Success     bool
	BlockHeight uint
}

type ResponseIScore struct {
	Address common.Address
	IScore  common.HexInt
}

type CalculateRequest struct {
	Path        string
	BlockHeight uint64
}

type CalculateResponse struct {
	Success     bool
	BlockHeight uint64
	StateHash   []byte
}

type rewardCalculate struct {
	lock sync.Mutex
	mgr  *manager
	conn ipc.Connection

	commitBlock uint
}

func (rc *rewardCalculate) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	log.Printf("Get message: (%d), %+v", msg, data)
	switch msg {
	case msgVERSION:
		var m VersionMessage
		m.Success = true
		m.BlockHeight = rc.commitBlock

		return rc.conn.Send(msg, &m)
	case msgClaim:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		// TODO implement claim module with goroutine
		var resp ResponseIScore
		resp.Address = addr
		resp.IScore.SetUint64(123)

		return rc.conn.Send(msgClaim, &resp)
	case msgQuery:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		// TODO implement query module with goroutine
		var resp ResponseIScore
		resp.Address = addr
		resp.IScore.SetUint64(123)

		return rc.conn.Send(msgQuery, &resp)
	case msgCalculate:
		var req CalculateRequest
		if _, err := codec.MP.UnmarshalFromBytes(data, &req); err != nil {
			return err
		}
		// TODO implement calculate module with goroutine
		var resp CalculateResponse
		resp.Success = true
		resp.BlockHeight = req.BlockHeight
		resp.StateHash = []byte(req.Path)

		return rc.conn.Send(msgCalculate, &resp)

	// TODO ADD message
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
}

func newConnection(m *manager, c ipc.Connection) (*rewardCalculate, error) {
	rc := &rewardCalculate{
		mgr: m,
		conn: c,
		commitBlock: 0,
	}

	c.SetHandler(msgVERSION, rc)
	c.SetHandler(msgClaim, rc)
	c.SetHandler(msgQuery, rc)
	c.SetHandler(msgCalculate, rc)
	// TODO add message handlers

	return rc, nil
}
