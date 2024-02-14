package main

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
	"syscall"
)

func server(ctx *cli.Context) error {
	var chainConfig *params.ChainConfig
	if chainConfigFlagVal := ctx.String(ChainConfigFlag.Name); chainConfigFlagVal != "" {
		chainConfig = loadChainConfig(ctx.String(ChainConfigFlag.Name))
	} else {
		// TODO: instead of assuming mainnet configuration in absence of chain config
		// val, accept known chain configurations via network preset flag.
		chainConfig = params.MainnetChainConfig
	}
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
