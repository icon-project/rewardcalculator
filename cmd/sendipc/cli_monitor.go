package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/icon-project/rewardcalculator/common/ipc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type monitorTarget struct {
	Name    string         `json:"name"`
	Address common.Address `json:"address"`
	IScore  common.HexInt  `json:"IScore,omitempty"`
}

type monitorConfig struct {
	Interval time.Duration  `json:"interval,omitempty"`
	Targets []monitorTarget `json:"targets"`
}


func (cli *CLI) monitor(conn ipc.Connection, configFile string, url string) {
	var config monitorConfig

	// read configuration file
	f, _ := os.Open(configFile)
	bs, _ := ioutil.ReadAll(f)
	if err := json.Unmarshal(bs, &config); err != nil {
		fmt.Printf("Can't unmarshal config file. err=%+v\n", err)
		return
	}

	fmt.Printf("Monitoring config: %s\n", Display(config))

	log.Println("[*] To exit press CTRL+C")

	for true {
		cli.readAndPush(conn, config.Targets, url)
		time.Sleep(config.Interval * time.Second)
	}
}

func (cli *CLI) readAndPush(conn ipc.Connection, targets []monitorTarget, url string) {
	pusher := push.New(url, "icon_rc")

	for _, target := range targets {
		// read IScore from RC
		resp := cli.query(conn, target.Address.String())
		target.IScore.Set(&resp.IScore.Int)

		// set metric
		temp := prometheus.NewGauge(prometheus.GaugeOpts{Name: target.Name})
		temp.Set(float64(target.IScore.Int.Uint64()))
		pusher.Collector(temp)
	}

	if err := pusher.Push(); err != nil {
		log.Printf("Can't push to %s, %+v", url, err)
	}
}
