package rewardcalculator

import (
	"log"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const Version uint16 = 1

type Manager interface {
	Loop() error
	Close() error
}

type manager struct {
	clientMode bool
	server     ipc.Server
	conn       ipc.Connection

	ctx        *Context
}

func (m *manager) Loop() error {
	if m.clientMode {
		for {
			err := m.conn.HandleMessage()
			if err != nil {
				log.Printf("Fail to handle message err=%+v", err)
				m.Close()
				return err
			}
		}
	} else {
		return m.server.Loop()

	}
}

func (m *manager) Close() error {
	if m.clientMode {
		m.conn.Close()
	} else {
		if err := m.server.Close(); err != nil {
			log.Printf("Failed to close IPC server err=%+v", err)
		}
	}

	CloseIScoreDB(m.ctx.db)
	return nil
}

// ConnectionHandler.OnConnect
func (m *manager) OnConnect(c ipc.Connection) error {
	_, err := newConnection(m, c)
	return err
}

// ConnectionHandler.OnClose
func (m *manager) OnClose(c ipc.Connection) error {
	// TODO finalize connection
	// use sync.WaitGroup ?

	return nil
}

func InitManager(clientMode bool, net string, addr string, IISSDataDir string, dbPath string, dbCount int) (*manager, error) {
	var err error
	m := new(manager)
	m.clientMode = clientMode

	// Initialize DB and load context values
	m.ctx, err = NewContext(dbPath, string(db.GoLevelDBBackend), "IScore", dbCount)

	// find IISS data and calculate
	reloadIISSData(m.ctx, IISSDataDir)

	m.ctx.Print()

	// Initialize ipc channel
	if m.clientMode {
		// connect to server
		conn, err := ipc.Dial(net, addr)
		if err != nil {
			return nil, err
		}
		m.OnConnect(conn)
		m.conn = conn

		// send VERSION message to server
		if m.clientMode {
			m.conn.Send(msgVERSION, 0, Version)
		}
	} else {
		// IPC Server
		srv := ipc.NewServer()
		err = srv.Listen(net, addr)
		if err != nil {
			return nil, err
		}
		srv.SetHandler(m)
		m.server = srv
	}

	return m, err
}

func reloadIISSData(ctx *Context, dir string) {
	for _, iissdata := range findIISSData(dir) {
		var req CalculateRequest
		req.Path = filepath.Join(dir, iissdata.Name())
		req.BlockHeight = 0

		log.Printf("Restore IISS Data. %s", req.Path)
		DoCalculate(ctx, &req)
	}
}
