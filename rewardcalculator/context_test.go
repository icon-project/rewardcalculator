package rewardcalculator

import (
	"github.com/icon-project/rewardcalculator/common/db"
	"io/ioutil"
	"os"
)

var testDir string

func initTest() *Context{
	var err error
	testDir, err = ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDir)

	ctx, _ := NewContext(testDir, string(db.GoLevelDBBackend), "test", 1)

	return ctx
}

func finalizeTest() {
	defer os.RemoveAll(testDir)
}
