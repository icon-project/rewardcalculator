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
	msgHELLO uint = 0
	msgClaim      = 1
	msgQuery      = 2
	msgCalculate  = 3
)

type helloMessage struct {
	Success     bool
	BlockHeight uint
}

type responseIScore struct {
	address common.Address
	iScore  common.HexInt
}

type caclulateMessage struct {
	path        string
	blockHeight uint64
}

type rewardCalculate struct {
	lock sync.Mutex
	mgr  *manager
	conn ipc.Connection

	commitBlock uint
}

func (rc *rewardCalculate) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	log.Printf("Get message: (%d), %X", msg, data)
	switch msg {
	case msgHELLO:
		var buf []byte
		_, err := codec.MP.UnmarshalFromBytes(data, &buf)
		if err != nil {
			log.Printf("Fail to unmarshal bytes:% X", data)
			return nil
		}

		var m helloMessage
		m.Success = true
		m.BlockHeight = rc.commitBlock

		return rc.conn.Send(msg, &m)
	case msgClaim:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		var response responseIScore
		response.address = addr
		// TODO implement with goroutine
		//response.iScore.Set(.frame.ctx.GetBalance(&addr))

		return rc.conn.Send(msgClaim, &response)
	case msgQuery:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		var response responseIScore
		response.address = addr
		// TODO implement with goroutine
		//response.iScore.Set(.frame.ctx.GetBalance(&addr))

		return rc.conn.Send(msgQuery, &response)
	case msgCalculate:
		var msg caclulateMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &msg); err != nil {
			return err
		}
		// TODO implement with goroutine
		//response.iScore.Set(.frame.ctx.GetBalance(&msg))

		return nil

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

	c.SetHandler(msgHELLO, rc)
	c.SetHandler(msgClaim, rc)
	c.SetHandler(msgQuery, rc)
	c.SetHandler(msgCalculate, rc)
	// TODO add message handlers

	return rc, nil
}
