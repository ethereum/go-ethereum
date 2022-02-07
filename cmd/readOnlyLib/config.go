package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

func makeFullNode(datadir string) (*node.Node, ethapi.Backend) {
	stack, cfg := makeConfigNode(datadir)
	backend, _ := utils.RegisterEthService(stack, cfg, false)
	return stack, backend
}

func makeConfigNode(datadir string) (*node.Node, *ethconfig.Config) {
	node_cfg := node.DefaultConfig
	eth_cfg := ethconfig.Defaults
	node_cfg.DataDir = datadir
	node_cfg.ReadOnly = true
	node_cfg.LocalLib = true
	stack, err := node.New(&node_cfg)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	return stack, &eth_cfg
}

func StartNode(stack *node.Node, isConsole bool) {
	if err := stack.Start(); err != nil {
		utils.Fatalf("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		shutdown := func() {
			log.Info("Got interrupt, shutting down...")
			go stack.Close()
			for i := 10; i > 0; i-- {
				<-sigc
				if i > 1 {
					log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
				}
			}
			debug.Exit() // ensure trace and CPU profile data is flushed.
			debug.LoudPanic("boom")
		}

		if isConsole {
			// In JS console mode, SIGINT is ignored because it's handled by the console.
			// However, SIGTERM still shuts down the node.
			for {
				sig := <-sigc
				if sig == syscall.SIGTERM {
					shutdown()
					return
				}
			}
		} else {
			<-sigc
			shutdown()
		}
	}()
}
