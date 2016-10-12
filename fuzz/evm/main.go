package evm

import "github.com/ethereum/go-ethereum/core/vm/runtime"

func Fuzz(data []byte) int {
	ret, _, err := runtime.Execute(data, data, &runtime.Config{GasLimit: 5000000})
	if err != nil && ret != nil {
		panic("ret != nil on error")
	}
	return 1
}
