package ethrepl

import (
	"bufio"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"io"
	"os"
	"path"
)

var logger = ethlog.NewLogger("REPL")

type Repl interface {
	Start()
	Stop()
}

type JSRepl struct {
	re *JSRE

	prompt string

	history *os.File

	running bool
}

func NewJSRepl(ethereum *eth.Ethereum) *JSRepl {
	hist, err := os.OpenFile(path.Join(ethutil.Config.ExecPath, "history"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}

	return &JSRepl{re: NewJSRE(ethereum), prompt: "> ", history: hist}
}

func (self *JSRepl) Start() {
	if !self.running {
		self.running = true
		logger.Infoln("init JS Console")
		reader := bufio.NewReader(self.history)
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("error reading history", err)
				break
			}

			addHistory(line[:len(line)-1])
		}
		self.read()
	}
}

func (self *JSRepl) Stop() {
	if self.running {
		self.running = false
		self.re.Stop()
		logger.Infoln("exit JS Console")
		self.history.Close()
	}
}

func (self *JSRepl) parseInput(code string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("[native] error", r)
		}
	}()

	value, err := self.re.Run(code)
	if err != nil {
		fmt.Println(err)
		return
	}

	self.PrintValue(value)
}
