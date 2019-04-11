package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/icon-project/rewardcalculator/rewardcalculator"
)

func (cli *CLI) read(dbDir string, dbName string) {
	path := filepath.Join(dbDir, dbName)
	fmt.Printf("Start read IISS data DB. Name: %s\n", path)

	if _ , err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("There is no DB %s\n", path)
			return
		}
	}

	rewardcalculator.LoadIISSData(path, true)
}
