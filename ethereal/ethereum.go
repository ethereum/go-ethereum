package main

import (
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/go-ethereum/ethereal/ui"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/go-qml/qml"
	"log"
	"os"
	"os/signal"
	"runtime"
)

const Debug = true

// Register interrupt handlers so we can stop the ethereum
func RegisterInterupts(s *eth.Ethereum) {
	// Buffered chan of one is enough
	c := make(chan os.Signal, 1)
	// Notify about interrupts for now
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("Shutting down (%v) ... \n", sig)

			s.Stop()
		}
	}()
}

func main() {
	Init()

	qml.Init(nil)

	runtime.GOMAXPROCS(runtime.NumCPU())

	ethchain.InitFees()
	ethutil.ReadConfig(DataDir)
	ethutil.Config.Seed = UseSeed

	// Instantiated a eth stack
	ethereum, err := eth.New(eth.CapDefault, UseUPnP)
	if err != nil {
		log.Println("eth start err:", err)
		return
	}
	ethereum.Port = OutboundPort

	if GenAddr {
		fmt.Println("This action overwrites your old private key. Are you sure? (y/n)")

		var r string
		fmt.Scanln(&r)
		for ; ; fmt.Scanln(&r) {
			if r == "n" || r == "y" {
				break
			} else {
				fmt.Printf("Yes or no?", r)
			}
		}

		if r == "y" {
			utils.CreateKeyPair(true)
		}
		os.Exit(0)
	} else {
		if len(ImportKey) > 0 {
			fmt.Println("This action overwrites your old private key. Are you sure? (y/n)")
			var r string
			fmt.Scanln(&r)
			for ; ; fmt.Scanln(&r) {
				if r == "n" || r == "y" {
					break
				} else {
					fmt.Printf("Yes or no?", r)
				}
			}

			if r == "y" {
				utils.ImportPrivateKey(ImportKey)
				os.Exit(0)
			}
		}
	}

	if ExportKey {
		key := ethutil.Config.Db.GetKeys()[0]
		fmt.Printf("%x\n", key.PrivateKey)
		os.Exit(0)
	}

	if ShowGenesis {
		fmt.Println(ethereum.BlockChain().Genesis())
		os.Exit(0)
	}

	log.Printf("Starting Ethereum GUI v%s\n", ethutil.Config.Ver)

	// Set the max peers
	ethereum.MaxPeers = MaxPeer

	gui := ethui.New(ethereum)
	gui.Start(AssetPath)
}
