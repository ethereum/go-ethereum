package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/graph-gophers/graphql-go/errors"
)

var filesToSkip = []string{
	"hive.json",
}

type simulationResult struct {
	Id             int    `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ClientVersions struct {
		GoEthereum string `json:"go-ethereum"`
	} `json:"clientVersions"`
	TestCases map[string]*testCase `json:"testCases"`
}

func (sim *simulationResult) check() error {
	for _, testCase := range sim.TestCases {
		if err := testCase.check(); err != nil {
			return errors.Errorf("simulation %v - %v (%v) failed with an error: %v", sim.Id, sim.Name, sim.Description, err)
		}
	}

	return nil
}

type testCase struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	SummaryResult struct {
		Pass bool `json:"pass"`
	} `json:"summaryResult"`
}

func (t *testCase) check() error {
	if !t.SummaryResult.Pass {
		return errors.Errorf("test case %v (%v) failed", t.Name, t.Description)
	}

	return nil
}

func parseSimulationResults(pathToResults string) ([]*simulationResult, error) {
	dirEntries, err := os.ReadDir(pathToResults)
	if err != nil {
		return nil, fmt.Errorf("can't read directory: %v", err)
	}

	simResults := make([]*simulationResult, 0)
	for _, dirEntry := range dirEntries {
		if skipFile(dirEntry) {
			continue
		}

		pathToFile := filepath.Join(pathToResults, dirEntry.Name())
		simResultsInJson, err := os.ReadFile(pathToFile)
		if err != nil {
			return nil, fmt.Errorf("can't read file: %v", err)
		}

		var simResult simulationResult
		if err := json.Unmarshal(simResultsInJson, &simResult); err != nil {
			return nil, fmt.Errorf("can't unmarshal simulation results: %v", err)
		}

		simResults = append(simResults, &simResult)
	}

	return simResults, nil
}

// skipFile returns true if file should be skipped (excluded from parsing and processing)
// skipFile returns true in such cases:
// - entry is a directory (not a file)
// - file doesn't have json extension
// - filename is explicitly marked to be skipped
func skipFile(dirEntry os.DirEntry) bool {
	if dirEntry.IsDir() {
		return true
	}

	ext := filepath.Ext(dirEntry.Name())
	if ext != ".json" {
		return true
	}

	for _, fileToSkip := range filesToSkip {
		if dirEntry.Name() == fileToSkip {
			return true
		}
	}

	return false
}

func main() {
	pathToResultsHelpString := `path to simulation results files of hive framework, by default should be in /path/to/hive/repo/workspace/logs`
	pathToResults := flag.String("path_to_results", "", pathToResultsHelpString)
	flag.Parse()

	simResults, err := parseSimulationResults(*pathToResults)
	if err != nil {
		log.Fatalf("can't parse simulation results: %v", err)
	}

	for _, simResult := range simResults {
		if err := simResult.check(); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("simulation %v - %v (%v) passed successfully\n", simResult.Id, simResult.Name, simResult.Description)
	}

	fmt.Printf("All %v simulations passed successfully\n", len(simResults))
}
