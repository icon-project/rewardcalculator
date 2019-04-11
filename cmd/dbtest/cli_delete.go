package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func (cli *CLI) delete(dbName string) {
	path := filepath.Join(DBDir, dbName)
	os.RemoveAll(path)
	fmt.Printf("Delete %s\n", path)
}
