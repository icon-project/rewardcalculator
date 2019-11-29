package ipc

import (
	"log"
	"net"
	"sync"

	"github.com/icon-project/rewardcalculator/common/codec"
	codec2 "github.com/ugorji/go/codec"
)

type MessageHandler interface {
	HandleMessage(c Connection, msg uint, id uint32, data []byte) error
}

type Connection interface {
	Send(msg uint, id uint32, data interface{}) error
	SendAndReceive(msg uint, id uint32, data interface{}, buf interface{}) error
	Receive(buf interface{}) (uint, uint32, error)
	SetHandler(msg uint, handler MessageHandler)
	HandleMessage() error
	Close() error
}

type ConnectionHandler interface {
	OnConnect(c Connection) error
	OnClose(c Connection) error
}

type connection struct {
	lock    sync.Mutex
	conn    net.Conn
	handler map[uint]MessageHandler
}

type messageToSend struct {
	Msg  uint
	Id   uint32
	Data interface{}
}

func connectionFromConn(conn net.Conn) *connection {
	c := &connection{
		conn:    conn,
		handler: map[uint]MessageHandler{},
	}
	return c
}

func (c *connection) Send(msg uint, id uint32, data interface{}) error {
	var m = messageToSend{
		Msg:  msg,
		Id:   id,
		Data: data,
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	return codec.MP.Marshal(c.conn, m)
}

type messageToReceive struct {
	Msg  uint
	Id   uint32
	Data codec2.Raw
}

func (c *connection) Receive(buffer interface{}) (uint, uint32, error) {
	var m messageToReceive
	if err := codec.MP.Unmarshal(c.conn, &m); err != nil {
		return m.Msg, m.Id, err
	}
	if _, err := codec.MP.UnmarshalFromBytes(m.Data, buffer); err != nil {
		return m.Msg, m.Id, err
	}

	return m.Msg, m.Id, nil
}

func (c *connection) SendAndReceive(msg uint, id uint32, data interface{}, buffer interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var m = messageToSend{
		Msg:  msg,
		Id:   id,
		Data: data,
	}

	err := codec.MP.Marshal(c.conn, m)
	if err != nil {
		return err
	}

	var m2 messageToReceive
	if err := codec.MP.Unmarshal(c.conn, &m2); err != nil {
		return err
	}
	if buffer != nil {
		if _, err := codec.MP.UnmarshalFromBytes(m2.Data, buffer); err != nil {
			return err
		}
	}
	return nil
}

func (c *connection) HandleMessage() error {
	var m messageToReceive
	if err := codec.MP.Unmarshal(c.conn, &m); err != nil {
		return err
	}
	c.lock.Lock()

	handler := c.handler[m.Msg]
	c.lock.Unlock()

	if handler == nil {
		log.Printf("Unknown message msg=%d\n", m.Msg)
		return nil
	}

	return handler.HandleMessage(c, m.Msg, m.Id, m.Data)
}

func (c *connection) SetHandler(msg uint, handler MessageHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.handler[msg] = handler
}

func (c *connection) Close() error {
	return c.conn.Close()
}

func Dial(network, address string) (Connection, error) {
	if conn, err := net.Dial(network, address); err != nil {
		return nil, err
	} else {
		return connectionFromConn(conn), nil
	}
}
