package ethutil

import (
	"fmt"
	"github.com/obscuren/mutan"
	"github.com/obscuren/serpent-go"
	"strings"
)

// General compile function
func Compile(script string) (ret []byte, err error) {
	c := strings.Split(script, "\n")[0]

	if c == "#!serpent" {
		byteCode, err := serpent.Compile(script)
		if err != nil {
			return nil, err
		}

		return byteCode, nil
	} else {
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
}

func CompileScript(script string) ([]byte, []byte, error) {
	// Preprocess
	mainInput, initInput := mutan.PreParse(script)
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
