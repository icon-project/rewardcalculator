package main

import (
	"encoding/json"
	"flag"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
	"os"
)

type RcConfig struct {
	IISSDataPath	string	`json:"IISSData"`
	DBDir           string  `json:"IScoreDB"`
	worker          *int     `json:"Worker"`
	fileName        string
	test			uint
}

func main() {
	var cfg RcConfig
	var generate bool

	flag.StringVar(&cfg.IISSDataPath, "iissdata", "./", "IISS Data path")
	flag.StringVar(&cfg.DBDir, "iscore_db", ".db", "I-Score database directory")
	flag.StringVar(&cfg.fileName, "config", "rc_config.json", "Reward Calculator configuration file")
	cfg.worker = flag.Int("worker", 2, "The number of I-Score calculation worker")
	flag.BoolVar(&generate, "gen", false, "Generate configuration file")
	flag.Parse()

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

	rcm, err := rewardcalculator.InitManager("unix", "/tmp/icon-rc.sock",
		cfg.IISSDataPath, cfg.DBDir, *cfg.worker)
	if err != nil {
		log.Panicf("Failed to start RewardCalculator manager %+v", err)
	}

	forever := make(chan bool)

	go rcm.Loop()

	log.Println("[*] To exit press CTRL+C")
	<-forever
}
