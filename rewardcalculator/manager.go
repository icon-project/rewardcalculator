package rewardcalculator

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const (
	Version uint16 = 1
	debugAddress   = "/tmp/icon-rc-debug.sock"
)

type RcConfig struct {
	IISSDataDir string `json:"IISSData"`
	DBDir       string `json:"IScoreDB"`
	IpcNet      string `json:"IPCNet"`
	IpcAddr     string `json:"IPCAddress"`
	ClientMode  bool   `json:"ClientMode"`
	DBCount     int    `json:"DBCount"`
	Monitor     bool   `json:"Monitor"`
	FileName    string
}

func (cfg *RcConfig) Print() {
	b, err := json.Marshal(cfg)
	if err != nil {
		log.Printf("Can't covert configuration to json")
		return
	}


	log.Printf("Running config %s\n", string(b))
}

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

	CloseIScoreDB(m.ctx.DB)
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

func InitManager(cfg *RcConfig) (*manager, error) {

	var err error
	m := new(manager)
	m.clientMode = cfg.ClientMode

	// Initialize DB and load context values
	m.ctx, err = NewContext(cfg.DBDir, string(db.GoLevelDBBackend), "IScore", cfg.DBCount)

	// find IISS data and calculate
	reloadIISSData(m.ctx, cfg.IISSDataDir)

	m.ctx.Print()

	// Initialize ipc channel
	if m.clientMode {
		// connect to server
		conn, err := ipc.Dial(cfg.IpcNet, cfg.IpcAddr)
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
		err = srv.Listen(cfg.IpcNet, cfg.IpcAddr)
		if err != nil {
			return nil, err
		}
		srv.SetHandler(m)
		m.server = srv
	}

	// Initialize debug channel
	if cfg.Monitor == true {
		debug := new(manager)
		debug.ctx = m.ctx

		srv := ipc.NewServer()
		err = srv.Listen("unix", debugAddress)
		if err != nil {
			return nil, err
		}
		srv.SetHandler(debug)
		debug.server = srv

		go debug.Loop()
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
