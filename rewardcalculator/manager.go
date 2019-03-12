package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
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

	IISSDataPath  string
}

func (m *manager) Loop() error {
	return m.server.Loop()
}

func (m *manager) Close() error {
	if err := m.server.Close(); err != nil {
		log.Printf("Failed to close IPC server err=%+v", err)
		return err
	}
	// TODO stop all rewardCalculate instance
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

func InitManager(net string, addr string, IISSDataPath string, dbPath string, worker int) (*manager, error) {
	// IPC Server
	srv := ipc.NewServer()
	err := srv.Listen(net, addr)
	if err != nil {
		return nil, err
	}

	m := new(manager)
	srv.SetHandler(m)
	m.server = srv

	// IISS datapath
	m.IISSDataPath = IISSDataPath

	// DB
	m.db, err = InitIscoreDB(dbPath, string(db.GoLevelDBBackend), "IScore", worker)


	return m, err
}
