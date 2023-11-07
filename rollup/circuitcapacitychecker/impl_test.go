package circuitcapacitychecker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type traceContainer struct {
	Jsonrpc string
	Id      int
	Result  *types.BlockTrace
}

func BenchmarkApplyBlock(b *testing.B) {
	ccc := NewCircuitCapacityChecker(false)

	dir, err := os.ReadDir("block-traces")
	if err != nil {
		b.Fatal(err)
	}

	for _, bte := range dir {
		next := filepath.Join("block-traces", bte.Name())
		data, err := os.ReadFile(next)
		if err != nil {
			b.Fatal(err)
		}

		var container traceContainer
		err = json.Unmarshal(data, &container)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(bte.Name(), func(b *testing.B) {
			ccc.Reset()
			_, err := ccc.ApplyBlock(container.Result)
			if err != nil {
				b.Fatal(err)
			}
		})
	}
}
