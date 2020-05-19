package tests

import (
	"testing"
)

func Test_rollback(t *testing.T) {
	var scenario TestScenario
	scenarioFile := "./json/rollback_scenario.json"

	opts := initTest()
	defer finalizeTest(opts)

	err := scenario.load(scenarioFile)
	if err != nil {
		t.Errorf("Failed to load %s. %v", scenarioFile, err)
		return
	}

	scenario.run(t, opts)
}
