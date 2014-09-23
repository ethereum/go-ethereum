package ethutil

import (
	"fmt"
	"strings"

	"github.com/obscuren/mutan"
	"github.com/obscuren/mutan/backends"
)

// General compile function
func Compile(script string, silent bool) (ret []byte, err error) {
	if len(script) > 2 {
		compiler := mutan.NewCompiler(backend.NewEthereumBackend())
		compiler.Silent = silent
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

	return nil, nil
}
