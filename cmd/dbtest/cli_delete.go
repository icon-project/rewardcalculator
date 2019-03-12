package main

import (
	"fmt"
	"os"
)

func (cli *CLI) delete(dbName string) {
	path := DBDir + "/" + dbName
	os.RemoveAll(path)
	fmt.Printf("Delete %s\n", path)
}
