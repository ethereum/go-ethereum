package main

import (
  "github.com/ethereum/eth-go"
  "github.com/ethereum/go-ethereum/utils"
  "os"
  "io/ioutil"
)

func InitJsConsole(ethereum *eth.Ethereum) {
  repl := NewJSRepl(ethereum)
  go repl.Start()
  utils.RegisterInterrupt(func(os.Signal) {
    repl.Stop()
    ethereum.Stop()
  })
}

func ExecJsFile (ethereum *eth.Ethereum, InputFile string) {
  file, err := os.Open(InputFile)
  if err != nil {
    logger.Fatalln(err)
  }
  content, err := ioutil.ReadAll(file)
  if err != nil {
    logger.Fatalln(err)
  }
  re := NewJSRE(ethereum)
  utils.RegisterInterrupt(func(os.Signal) {
    re.Stop()
  })
  re.Run(string(content))
}
