package helper

import (
	"log"
	"os"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
)

func init() {
	ethlog.AddLogSystem(ethlog.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlog.LogLevel(4)))

	ethutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
}
