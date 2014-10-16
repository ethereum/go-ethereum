package helper

import (
	"log"
	"os"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
)

var Logger ethlog.LogSystem
var Log = ethlog.NewLogger("TEST")

func init() {
	Logger = ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.InfoLevel)
	ethlog.AddLogSystem(Logger)

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
}
