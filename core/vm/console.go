package vm

import (
	_ "embed"
	"encoding/hex"
	"fmt"
	"regexp"

	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
)

//go:embed console.sol
var console string

var logOverloads = map[string]*abi.Type{}

func init() {
	rxp := regexp.MustCompile("abi.encodeWithSignature\\(\"log(.*)\"")
	matches := rxp.FindAllStringSubmatch(console, -1)

	for _, match := range matches {
		signature := match[1]

		// parse the type of the console call. Note that 'uint'
		// objects are defined without bytes (i.e. 256).
		typ, err := abi.NewType("tuple" + signature)
		if err != nil {
			panic(fmt.Errorf("BUG: Failed to parse %s", signature))
		}

		// signature of the call. Use the version without the bytes in 'uint'.
		sig := ethgo.Keccak256([]byte("log" + match[1]))[:4]
		logOverloads[hex.EncodeToString(sig)] = typ
	}
}

func decodeConsole(input []byte) (val []string) {
	sig := hex.EncodeToString(input[:4])
	logSig, ok := logOverloads[sig]
	if !ok {
		return
	}
	input = input[4:]
	raw, err := logSig.Decode(input)
	if err != nil {
		return
	}
	val = []string{}
	for _, v := range raw.(map[string]interface{}) {
		val = append(val, fmt.Sprint(v))
	}
	return
}

// consolePrecompile is a debug precompile contract that simulates the `console.sol` functionality
type consolePrecompile struct{}

func (c *consolePrecompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (c *consolePrecompile) Run(input []byte) ([]byte, error) {
	val := decodeConsole(input)
	fmt.Printf("Console: %v\n", val)

	return nil, nil
}
