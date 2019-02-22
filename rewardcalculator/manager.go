package rewardcalculator

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/ipc"
	"log"
)

type Manager interface {
	Loop() error
	Close() error
}

type manager struct {
	server ipc.Server
	db     db.Database
	//lock   sync.Mutex		//TODO need?
}

func (m *manager) Loop() error {
	return m.server.Loop()
}

func (m *manager) Close() error {
	if err := m.server.Close(); err != nil {
		log.Printf("Failed to close IPC server err=%+v", err)
		return err
	}
	// TODO stop all RewardCalculate instance
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

func InitManager(net string, addr string, datapath string) (*manager, error) {
	// IPC
	srv := ipc.NewServer()
	err := srv.Listen(net, addr)
	if err != nil {
		return nil, err
	}

	m := new(manager)
	srv.SetHandler(m)
	m.server = srv

	// DB
	m.db = InitDB(datapath, string(db.GoLevelDBBackend), "IScore")

	return m, nil
}
