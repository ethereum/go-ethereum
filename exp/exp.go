package exp

import (
	"github.com/ethereum/go-ethereum/metrics"
	e "github.com/ethereum/go-ethereum/metrics/exp"
)

func init() {
	e.Exp(metrics.DefaultRegistry)
}
