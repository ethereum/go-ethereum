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
	if len(script) > 2 {
		line := strings.Split(script, "\n")[0]

		if len(line) > 1 && line[0:2] == "#!" {
			switch line {
			case "#!serpent":
				byteCode, err := serpent.Compile(script)
				if err != nil {
					return nil, err
				}

				return byteCode, nil
			}
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

	return nil, nil
}
