package main

import (
	"flag"
	"fmt"
	"github.com/ethereum/eth-go"
	"github.com/ethereum/ethchain-go"
	"github.com/ethereum/ethutil-go"
	_ "github.com/ethereum/ethwire-go"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path"
	"runtime"
	"time"
)

const Debug = true

var StartConsole bool
var StartMining bool

func Init() {
	flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
	flag.BoolVar(&StartMining, "m", false, "start dagger mining")

	flag.Parse()
}

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
	runtime.GOMAXPROCS(runtime.NumCPU())
	Init()

	ethchain.InitFees()
	ethutil.ReadConfig()

	log.Printf("Starting Ethereum v%s\n", ethutil.Config.Ver)

	// Instantiated a eth stack
	ethereum, err := eth.New()
	if err != nil {
		log.Println(err)
		return
	}

	if StartConsole {
		err := os.Mkdir(ethutil.Config.ExecPath, os.ModePerm)
		// Error is OK if the error is ErrExist
		if err != nil && !os.IsExist(err) {
			log.Panic("Unable to create EXECPATH. Exiting")
		}

		// TODO The logger will eventually be a non blocking logger. Logging is a expensive task
		// Log to file only
		file, err := os.OpenFile(path.Join(ethutil.Config.ExecPath, "debug.log"), os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			log.Panic("Unable to set proper logger", err)
		}

		ethutil.Config.Log = log.New(file, "", 0)

		console := NewConsole(ethereum)
		go console.Start()
	}

	RegisterInterupts(ethereum)

	ethereum.Start()

	if StartMining {
		blockTime := time.Duration(15)
		log.Printf("Dev Test Mining started. Blocks found each %d seconds\n", blockTime)

		// Fake block mining. It broadcasts a new block every 5 seconds
		go func() {
			for {

				time.Sleep(blockTime * time.Second)

				txs := ethereum.TxPool.Flush()

				block := ethchain.CreateBlock(
					ethereum.BlockManager.CurrentBlock.State().Root,
					ethereum.BlockManager.LastBlockHash,
					"123",
					big.NewInt(1),
					big.NewInt(1),
					"",
					txs)
				err := ethereum.BlockManager.ProcessBlockWithState(block, block.State())
				if err != nil {
					log.Println(err)
				} else {
					log.Println("\n+++++++ MINED BLK +++++++\n", block.String())
				}
			}
		}()
	}

	// Wait for shutdown
	ethereum.WaitForShutdown()
}
