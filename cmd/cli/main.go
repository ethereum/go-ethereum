package main

import (
	"os"

	"github.com/ethereum/go-ethereum/internal/cli"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	params.UpdateBorInfo()
	os.Exit(cli.Run(os.Args[1:]))
}
