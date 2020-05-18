package core

import (
	"encoding/json"
	"fmt"
	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"log"
	"math"
	"path/filepath"
	"sync"
)

const (
	DebugAddress = "/tmp/.icon-rc-monitor.sock"
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
	CalcDebugConf string `json:"CalcDebugConf"`
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

	ctx       *Context
	waitGroup *sync.WaitGroup
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
	m.ctx.CancelCalculation.notifyExit()
	m.WaitMsgTasksDone()
	if m.clientMode {
		m.conn.Close()
	} else {
		if err := m.server.Close(); err != nil {
			log.Printf("Failed to close IPC server err=%+v", err)
		}
	}

	CloseIScoreDB(m.ctx.DB)
	log.Printf("Exit Reward Calculator")
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

func (m *manager) AddMsgTask() {
	m.waitGroup.Add(1)
}

func (m *manager) DoneMsgTask() {
	m.waitGroup.Done()
}

func (m *manager) WaitMsgTasksDone() {
	log.Printf("Wait until all goroutines accessing DB done")
	m.waitGroup.Wait()
}
func InitManager(cfg *RcConfig) (*manager, error) {

	var err error
	waitGroup := new(sync.WaitGroup)
	m := new(manager)
	m.clientMode = cfg.ClientMode
	m.waitGroup = waitGroup

	// Initialize DB and load context values
	m.ctx, err = NewContext(cfg.DBDir, string(db.GoLevelDBBackend), "IScore", cfg.DBCount, cfg.CalcDebugConf)
	if err != nil {
		return nil, err
	}

	m.ctx.Print()

	// find IISS data and reload
	go reloadIISSData(m.ctx, cfg.IISSDataDir)

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
		monitor.waitGroup = waitGroup

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

const reloadBlockHeight = math.MaxUint64
const reloadMsgID = math.MaxUint32

func reloadIISSData(ctx *Context, dir string) {
	if needIISSDataReload(ctx) {
		var req CalculateRequest
		req.Path = filepath.Join(dir, fmt.Sprintf(IISSDataDBFormat, ctx.DB.getCalculatingBH()))
		req.BlockHeight = reloadBlockHeight

		log.Printf("Reload IISS Data. %s", req.Path)
		err, _, _, _ := DoCalculate(ctx.CancelCalculation.GetChannel(), ctx, &req, nil, reloadMsgID)

		if err != nil {
			log.Printf("Failed to reload IISS Data. %s. %v", req.Path, err)
		} else {
			log.Printf("Succeeded to reload IISS Data. %s", req.Path)
			// cleanup IISS data DB
			cleanupIISSData(req.Path)
		}
	}
}

func needIISSDataReload(ctx *Context) bool {
	return ctx.DB.getCalcDoneBH() != ctx.DB.getCalculatingBH()
}

func isReloadRequest(blockHeight uint64, id uint32) bool {
	return blockHeight == reloadBlockHeight && id == reloadMsgID
}
