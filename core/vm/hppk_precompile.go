package vm

import "errors"

const (
	hppkGasBase    = 3000
	hppkGasPerByte = 12
)

type hppkPrecompile struct{}

func (*hppkPrecompile) Name() string { return "hppk" }

func (*hppkPrecompile) RequiredGas(input []byte) uint64 {
	return uint64(hppkGasBase + hppkGasPerByte*len(input))
}

func (*hppkPrecompile) Run(input []byte) ([]byte, error) {
	if len(input) < 4 {
		return nil, errors.New("hppk: input too short")
	}
	out := make([]byte, len(input))
	copy(out, input)
	return out, nil
}
