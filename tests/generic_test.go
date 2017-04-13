package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	genericTestDir = filepath.Join(baseDir, "GeneralStateTests")
)

func TestGenericState(t *testing.T) {
	runDir(genericTestDir, t)
}

func runDir(path string, t *testing.T) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Error(err)
	}

	for _, file := range files {
		t.Run(file.Name(), func(t *testing.T) {
			fpath := filepath.Join(path, file.Name())
			if file.IsDir() {
				runDir(fpath, t)
			} else {
				if err := runTests(fpath, t); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func runTests(path string, t *testing.T) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading JSON file: %v", err)
	}

	test := make(map[string]StateTest)

	if err = json.Unmarshal(data, &test); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return fmt.Errorf("JSON unmarshal error: %v", err)
	}
	return nil
}

type hardfork string

const (
	EIP150     hardfork = "EIP150"
	EIP158              = "EIP158"
	Frontier            = "Frontier"
	Homestead           = "Homestead"
	Metropolis          = "Metropolis"
)

type StateTest struct {
	Aux       AuxiliaryData                  `json:"env"`
	PreState  map[common.Address]StateObject `json:"pre"`
	PostState map[hardfork][]PostState       `json:"post"`
}

type AuxiliaryData struct {
	Coinbase   common.Address `json:"currentCoinbase"`
	Difficulty hexutil.Big    `json:"currentDifficulty"`
	GasLimit   hexutil.Big    `json:"currentGasLimit"`
	Number     hexutil.Big    `json:"Number"`
	Timestamp  hexutil.Uint64 `json:"currentTimestamp"`
	ParentHash common.Hash    `json:"previousHash"`
}

type StateObject struct {
	Balance hexutil.Big                 `json:"balance"`
	Code    hexutil.Bytes               `json:"code"`
	Nonce   hexutil.Uint64              `json:"nonce"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

type PostState struct {
	Hash    common.Hash `json:"hash"`
	Indices struct {
		//Data  hexutil.Uint64 `json:"data"`
		//Gas   hexutil.Uint64 `json:"Gas"`
		//Value hexutil.Uint64 `json:"Value"`
	} `json:"indexes"`
}
