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
	clientMode bool
	server     ipc.Server
	conn       ipc.Connection

	gOpts *GlobalOptions
	//lock   sync.Mutex		//TODO need?

	IISSDataPath  string
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
			return err
		}
	}

	// TODO stop all rewardCalculate instance
	CloseIScoreDB(m.gOpts.db)
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

func InitManager(clientMode bool, net string, addr string, IISSDataPath string, dbPath string, dbCount int) (*manager, error) {
	var err error
	m := new(manager)
	m.clientMode = clientMode
	if m.clientMode {
		// connect to server
		conn, err := ipc.Dial(net, addr)
		if err != nil {
			return nil, err
		}
		m.OnConnect(conn)
		m.conn = conn
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

	// set IISS Data path
	m.IISSDataPath = IISSDataPath

	// Initialize DB and load global options
	m.gOpts, err = InitIScoreDB(dbPath, string(db.GoLevelDBBackend), "IScore", dbCount)

	// TODO send VERSION message

	m.gOpts.Print()

	return m, err
}
