package main

import (
	"io/ioutil"
	"os"

	"github.com/ethereum/eth-go"
	"github.com/ethereum/go-ethereum/ethereum/repl"
	"github.com/ethereum/go-ethereum/javascript"
	"github.com/ethereum/go-ethereum/utils"
)

func InitJsConsole(ethereum *eth.Ethereum) {
	repl := ethrepl.NewJSRepl(ethereum)
	go repl.Start()
	utils.RegisterInterrupt(func(os.Signal) {
		repl.Stop()
	})
}

func ExecJsFile(ethereum *eth.Ethereum, InputFile string) {
	file, err := os.Open(InputFile)
	if err != nil {
		logger.Fatalln(err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Fatalln(err)
	}
	re := javascript.NewJSRE(ethereum)
	utils.RegisterInterrupt(func(os.Signal) {
		re.Stop()
	})
	re.Run(string(content))
}
