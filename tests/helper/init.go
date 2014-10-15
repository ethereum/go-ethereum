package helper

import (
	"log"
	"os"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
)

var Logger ethlog.LogSystem

func init() {
	Logger = ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.LogLevel(4))
	ethlog.AddLogSystem(Logger)

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
}
