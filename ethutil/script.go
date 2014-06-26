package ethutil

import (
	"fmt"
	"github.com/obscuren/mutan"
	"github.com/obscuren/mutan/backends"
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
		compiler := mutan.NewCompiler(backend.NewEthereumBackend())
		byteCode, errors := compiler.Compile(strings.NewReader(script))
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
