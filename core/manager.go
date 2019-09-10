package core

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
)

const (
	DebugAddress        = "/tmp/.icon-rc-monitor.sock"
)

type RcConfig struct {
	IISSDataDir   string `json:"IISSData"`
	DBDir         string `json:"IScoreDB"`
	IpcNet        string `json:"IPCNet"`
	IpcAddr       string `json:"IPCAddress"`
	ClientMode    bool   `json:"ClientMode"`
	DBCount       int    `json:"DBCount"`
	Monitor       bool   `json:"Monitor"`
	LogFile       string `json:"LogFile"`
	LogMaxSize    int    `json:"LogMaxSize"`
	LogMaxBackups int    `json:"LogMaxBackups"`
	FileName      string
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
	clientMode  bool
	monitorMode bool
	server      ipc.Server
	conn        ipc.Connection

	ctx        *Context
}

func (m *manager) Loop() error {
	if m.clientMode {
		for {
			err := m.conn.HandleMessage()
			if err != nil {
				log.Printf("Failed to handle message err=%+v", err)
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
		monitor := new(manager)
		monitor.ctx = m.ctx
		monitor.monitorMode = true

		srv := ipc.NewServer()
		err = srv.Listen("unix", DebugAddress)
		if err != nil {
			return nil, err
		}
		srv.SetHandler(monitor)
		monitor.server = srv

		go monitor.Loop()
	}

	return m, err
}

func reloadIISSData(ctx *Context, dir string) {
	respSlice := make([]*CalculateDone, 0)
	for _, iissData := range findIISSData(dir) {
		var req CalculateRequest
		req.Path = filepath.Join(dir, iissData.Name())
		req.BlockHeight = 0

		log.Printf("Reload IISS Data. %s", req.Path)
		success, blockHeight, stats, stateHash := DoCalculate(ctx, &req, nil, 0)

		// remove IISS data DB
		os.RemoveAll(req.Path)

		if success == false {
			break
		}

		// save result
		resp := new(CalculateDone)
		resp.BlockHeight = blockHeight
		resp.Success = success
		if stats != nil {
			resp.IScore.Set(&stats.Beta3.Int)
		} else {
			resp.IScore.SetUint64(0)
		}
		resp.StateHash = stateHash

		respSlice = append(respSlice, resp)
	}

	ctx.reloadIISS = respSlice
}

func sendReloadIISSDataResult(ctx *Context, c ipc.Connection) error {
	var err error = nil

	// send IISS data reload result
	for _, resp := range ctx.reloadIISS {
		err = c.Send(MsgCalculate, 0, *resp)
		if err != nil {
			log.Printf("Failed to send IISS data reload result. (%+v)", resp)
			break
		} else {
			log.Printf("Send IISS data reload result. (%+v)", resp)
		}
	}

	return err
}
