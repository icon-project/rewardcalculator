package main

import (
	"encoding/json"
	"flag"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
	"os"
)

var (
	version = "unknown"
	build   = "unknown"
)

func main() {
	var cfg rewardcalculator.RcConfig
	var generate bool

	log.Printf("Version : %s", version)
	log.Printf("Build   : %s", build)

	flag.StringVar(&cfg.IISSDataDir, "iissdata", "./iissdata", "IISS Data directory")
	flag.StringVar(&cfg.DBDir, "db", ".iscoredb", "I-Score database directory")
	flag.StringVar(&cfg.IpcNet, "ipc-net", "unix", "IPC channel network type")
	flag.StringVar(&cfg.IpcAddr, "ipc-addr", "/tmp/icon-rc.sock", "IPC channel address")
	flag.StringVar(&cfg.FileName, "config", "rc_config.json", "Reward Calculator configuration file")
	flag.BoolVar(&cfg.ClientMode, "client", false, "Connect to ICON Service")
	flag.BoolVar(&cfg.Monitor, "monitor", true, "Open monitoring channel")
	cfg.DBCount = *flag.Int("db-count", 2, "The number of Account DB (MAX:256")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.Parse()
	cfg.Print()

	if generate {
		if len(cfg.FileName) == 0 {
			cfg.FileName = "rc_config.json"
		}
		f, err := os.OpenFile(cfg.FileName,
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			log.Panicf("Fail to open file=%s err=%+v", cfg.FileName, err)
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(&cfg); err != nil {
			log.Panicf("Fail to generate JSON for %+v", cfg)
		}
		f.Close()
		os.Exit(0)
	}

	if cfg.DBCount > rewardcalculator.MaxDBCount {
		log.Printf("Too large -db-count %d. MAX: %d", cfg.DBCount, rewardcalculator.MaxDBCount)
	}

	rcm, err := rewardcalculator.InitManager(&cfg)
	if err != nil {
		log.Panicf("Failed to start RewardCalculator manager %+v", err)
	}

	forever := make(chan bool)

	go rcm.Loop()

	log.Println("[*] To exit press CTRL+C")
	<-forever
}
