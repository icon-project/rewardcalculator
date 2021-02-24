package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/icon-project/rewardcalculator/common/db"
	"github.com/icon-project/rewardcalculator/core"
)

type testOption struct {
	rootPath string
	cmd      *exec.Cmd
	db       db.Database
	ipc      *core.RCIPC
}

func initTest() *testOption {
	var err error
	testDir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}

	opts := new(testOption)
	opts.rootPath = testDir

	dbPath := filepath.Join(opts.rootPath, ".iscoredb")
	address := filepath.Join(opts.rootPath, "icon_rc.sock")
	cmd := exec.Command("icon_rc", "-db", dbPath, "-ipc-addr", address)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	opts.cmd = cmd

	opts.ipc, err = core.InitRCIPC("unix", address)
	if err != nil {
		panic(err)
	}

	log.Printf("Initialize test %v", opts)

	return opts
}

func finalizeTest(opts *testOption) {
	log.Printf("Finalize test %v", opts)

	core.FiniRCIPC(opts.ipc)

	err := opts.cmd.Process.Kill()
	if err != nil {
		log.Printf("%v finished with error: %v", opts.cmd, err)
	}
	err = os.RemoveAll(opts.rootPath)
	if err != nil {
		log.Printf("Failed to remove DB. %v", err)
	}
}

type TestScenario struct {
	Name  string `json:"name"`
	IISS  []iiss `json:"iiss"`
	Tests []ipc  `json:"tests,omitempty"`
}

func (ts *TestScenario) load(file string) error {
	// read configuration file
	f, _ := os.Open(file)
	defer f.Close()

	original, _ := ioutil.ReadAll(f)
	bs, err := ts.translate(original)
	if err != nil {
		log.Printf("Failed to translate test scenario. err=%+v\n", err)
		return err
	}

	// unmarshal scenario data
	if err := json.Unmarshal(bs, ts); err != nil {
		log.Printf("Failed to unmarshal test scenario. err=%+v\n", err)
		return err
	}

	return nil
}

func (ts *TestScenario) translate(bs []byte) ([]byte, error) {
	var allData map[string]*json.RawMessage
	json.Unmarshal(bs, &allData)

	// load variables
	var variables map[string]interface{}
	if err := json.Unmarshal(*allData["variable"], &variables); err != nil {
		log.Printf("Failed to unmarshal variable data. err=%+v\n", err)
		return nil, err
	}

	// get variable Info.
	type replaceValue struct {
		index int
		new   interface{}
	}
	replace := make([]replaceValue, 0)
	fields := bytes.Fields(bs)
	for i, data := range fields {
		if bytes.HasPrefix(data, []byte("\"$")) {
			varName := getVariableName(data)
			var v replaceValue
			var ok bool
			v.index = i
			v.new, ok = variables[varName]
			if ok {
				replace = append(replace, v)
			}
		}
	}

	// replace variable with value
	for _, data := range replace {
		comma := ""
		if bytes.HasSuffix(fields[data.index], []byte(",")) {
			comma = ","
		}

		var newValue []byte
		switch v := data.new.(type) {
		case string:
			newValue = []byte("\"" + v + "\"" + comma)
		case float64:
			newValue = []byte(fmt.Sprintf("%d%s", uint64(v), comma))
		}
		fields[data.index] = newValue
	}

	return bytes.Join(fields, nil), nil
}

func getVariableName(data []byte) string {
	// remove prefix "$
	data = data[2:]
	// remove suffix " or ",
	if bytes.HasSuffix(data, []byte(",")) {
		data = data[:len(data)-2]
	} else {
		data = data[:len(data)-1]
	}
	return string(data)
}

func (ts *TestScenario) run(t *testing.T, opts *testOption) {
	t.Run(ts.Name, func(t *testing.T) {
		for _, value := range ts.IISS {
			err := value.run(opts)
			if err != nil {
				t.Error(err)
			}
		}
	})

	for _, tt := range ts.Tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := tt.run(opts)
			if err != nil {
				t.Error(err)
			}
		})
	}
}
