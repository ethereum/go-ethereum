package utils

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/mutan"
	"strings"
)

// General compile function
func Compile(script string) ([]byte, error) {
	byteCode, errors := mutan.Compile(strings.NewReader(script), false)
	if len(errors) > 0 {
		var errs string
		for _, er := range errors {
			if er != nil {
				errs += er.Error()
			}
		}
		return nil, fmt.Errorf("%v", errs)
	}

	return byteCode, nil
}

func CompileScript(script string) ([]byte, []byte, error) {
	// Preprocess
	mainInput, initInput := ethutil.PreProcess(script)
	// Compile main script
	mainScript, err := Compile(mainInput)
	if err != nil {
		return nil, nil, err
	}

	// Compile init script
	initScript, err := Compile(initInput)
	if err != nil {
		return nil, nil, err
	}

	return mainScript, initScript, nil
}
