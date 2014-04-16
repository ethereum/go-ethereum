package utils

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/mutan"
	"strings"
)

// General compile function
func Compile(script string) ([]byte, error) {
	asm, errors := mutan.Compile(strings.NewReader(script), false)
	if len(errors) > 0 {
		var errs string
		for _, er := range errors {
			if er != nil {
				errs += er.Error()
			}
		}
		return nil, fmt.Errorf("%v", errs)
	}

	return ethutil.Assemble(asm...), nil
}
