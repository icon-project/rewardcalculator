package main

import (
	"flag"
	"github.com/icon-project/rewardcalculator/rewardcalculator"
	"log"
)

type RcConfig struct {
	IISSDataPath	string	`json:"IISSData"`
	test			uint	`json:"test"`
}

func main() {
	var cfg RcConfig

	flag.StringVar(&cfg.IISSDataPath, "datapath", "./", "IISS Data path")
	flag.UintVar(&cfg.test, "test", 0, "Test")
	flag.Parse()

	rcm, err := rewardcalculator.InitManager("unix", "/tmp/icon-rc.sock", cfg.IISSDataPath)
	if err != nil {
		log.Panicln("FAIL to start RewardCalculator manager")
	}

	forever := make(chan bool)

	go rcm.Loop()

	log.Println("[*] To exit press CTRL+C")
	<-forever
}
