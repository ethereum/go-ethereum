package helper

import (
	"log"
	"os"

	"github.com/ethereum/go-ethereum/ethlog"
	"github.com/ethereum/go-ethereum/ethutil"
)

var Logger ethlog.LogSystem
var Log = ethlog.NewLogger("TEST")

func init() {
	Logger = ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.InfoLevel)
	ethlog.AddLogSystem(Logger)

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
}
