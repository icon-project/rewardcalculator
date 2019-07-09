package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/icon-project/rewardcalculator/core"
	"github.com/natefinch/lumberjack"
)

var (
	version = "unknown"
	build   = "unknown"
)

func main() {
	var cfg core.RcConfig
	var generate bool

	flag.StringVar(&cfg.IISSDataDir, "iissdata", "./iissdata", "IISS Data directory")
	flag.StringVar(&cfg.DBDir, "db", ".iscoredb", "I-Score database directory")
	flag.StringVar(&cfg.IpcNet, "ipc-net", "unix", "IPC channel network type")
	flag.StringVar(&cfg.IpcAddr, "ipc-addr", "/tmp/icon-rc.sock", "IPC channel address")
	flag.StringVar(&cfg.FileName, "config", "rc_config.json", "Reward Calculator configuration file")
	flag.BoolVar(&cfg.ClientMode, "client", false, "Connect to ICON Service")
	flag.BoolVar(&cfg.Monitor, "monitor", false, "Open monitoring channel")
	flag.IntVar(&cfg.DBCount, "db-count", 2, "The number of Account DB (MAX:256)")
	flag.StringVar(&cfg.LogFile, "log-file", "icon_rc.log", "Log file name")
	flag.IntVar(&cfg.LogMaxSize, "log-max-size", 10, "MAX size of log file in megabytes")
	flag.IntVar(&cfg.LogMaxBackups, "log-max-backups", 10, "MAX number of old log files")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(&lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogMaxSize,
		MaxBackups: cfg.LogMaxBackups,
		LocalTime:  true,
	})

	log.Printf("Version : %s", version)
	log.Printf("Build   : %s", build)

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

	if cfg.DBCount > core.MaxDBCount {
		fmt.Printf("Too large -db-count %d. MAX: %d\n", cfg.DBCount, core.MaxDBCount)
	}

	rcm, err := core.InitManager(&cfg)
	if err != nil {
		log.Panicf("Failed to start RewardCalculator manager %+v", err)
	}

	forever := make(chan bool)

	go rcm.Loop()

	fmt.Printf("[*] To exit press CTRL+C\n")
	<-forever
}
