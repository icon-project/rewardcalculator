package rewardcalculator

import (
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

type RewardCalculate struct {
	lock sync.Mutex
	mgr  *manager
	conn ipc.Connection

	commitBlock uint
}

func (rc *RewardCalculate) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
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
	// TODO ADD message
	case msgClaim:
		return nil
	case msgQuery:
		return nil
	case msgCalculate:
		return nil
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
}

func newConnection(m *manager, c ipc.Connection) (*RewardCalculate, error) {
	rc := &RewardCalculate{
		mgr: m,
		conn: c,
		commitBlock: 0,
	}

	c.SetHandler(msgHELLO, rc)
	// TODO add message handlers

	return rc, nil
}
