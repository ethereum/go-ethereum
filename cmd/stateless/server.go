package main

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
	"syscall"
)

func server(ctx *cli.Context) error {
	chainConfig := loadChainConfig(ctx.String(ChainConfigFlag.Name))
	closeCh, _, err := utils.RunLocalServer(chainConfig, 8080)
	if err != nil {
		return err
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	select {
	case <-sigc:
		closeCh <- struct{}{}
	}

	return nil
}
