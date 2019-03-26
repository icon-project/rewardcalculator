package main

import (
	"encoding/json"
	"flag"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
	"os"
)

type RcConfig struct {
	IISSDataPath string `json:"IISSData"`
	DBDir        string `json:"IScoreDB"`
	IpcAddr      string `json:"IPCAddress"`
	ClientMode   bool   `json:"ClientMode"`
	Worker       int    `json:"Worker"`
	fileName     string
	test         uint
}

func (cfg *RcConfig) Print() {
	b, err := json.Marshal(cfg)
	if err != nil {
		log.Printf("Can't covert configuration to json")
		return
	}


	log.Printf("Running config %s\n", string(b))
}

func main() {
	var cfg RcConfig
	var generate bool

	flag.StringVar(&cfg.IISSDataPath, "iissdata", "./iissdata", "IISS Data directory")
	flag.StringVar(&cfg.DBDir, "db", ".iscoredb", "I-Score database directory")
	flag.StringVar(&cfg.IpcAddr, "ipc", "/tmp/icon-rc.sock", "IPC channel")
	flag.StringVar(&cfg.fileName, "config", "rc_config.json", "Reward Calculator configuration file")
	flag.BoolVar(&cfg.ClientMode, "client", false, "Generate configuration file")
	cfg.Worker = *flag.Int("worker", 2, "The number of I-Score calculation Worker")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.Parse()
	cfg.Print()

	if generate {
		if len(cfg.fileName) == 0 {
			cfg.fileName = "rc_config.json"
		}
		f, err := os.OpenFile(cfg.fileName,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", cfg.fileName, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	rcm, err := rewardcalculator.InitManager(cfg.ClientMode, "unix", cfg.IpcAddr, cfg.IISSDataPath, cfg.DBDir, cfg.Worker)
	if err != nil {
		log.Panicf("Failed to start RewardCalculator manager %+v", err)
	}

	forever := make(chan bool)

	go rcm.Loop()

	log.Println("[*] To exit press CTRL+C")
	<-forever
}
