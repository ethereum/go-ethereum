package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/signer/core"
	"github.com/ethereum/go-ethereum/signer/fourbyte"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "<hexdata>")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, `
Parses the given ABI data and tries to interpret it from the fourbyte database.`)
	}
}

func parse(data []byte) {
	db, err := fourbyte.New()
	if err != nil {
		die(err)
	}
	messages := core.ValidationMessages{}
	db.ValidateCallData(nil, data, &messages)
	for _, m := range messages.Messages {
		fmt.Printf("%v: %v\n", m.Typ, m.Message)
	}
}

// Example
// ./abidump a9059cbb000000000000000000000000ea0e2dc7d65a50e77fc7e84bff3fd2a9e781ff5c0000000000000000000000000000000000000000000000015af1d78b58c40000
func main() {
	flag.Parse()

	switch {
	case flag.NArg() == 1:
		hexdata := flag.Arg(0)
		data, err := hex.DecodeString(strings.TrimPrefix(hexdata, "0x"))
		if err != nil {
			die(err)
		}
		parse(data)
	default:
		fmt.Fprintln(os.Stderr, "Error: one argument needed")
		flag.Usage()
		os.Exit(2)
	}
}

func die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
