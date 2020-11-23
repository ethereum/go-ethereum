package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
)

var (
	//
	// Bor Specific flags
	//

	// HeimdallURLFlag flag for heimdall url
	HeimdallURLFlag = cli.StringFlag{
		Name:  "bor.heimdall",
		Usage: "URL of Heimdall service",
		Value: "http://localhost:1317",
	}

	// WithoutHeimdallFlag no heimdall (for testing purpose)
	WithoutHeimdallFlag = cli.BoolFlag{
		Name:  "bor.withoutheimdall",
		Usage: "Run without Heimdall service (for testing purpose)",
	}

	// BorFlags all bor related flags
	BorFlags = []cli.Flag{
		HeimdallURLFlag,
		WithoutHeimdallFlag,
	}
)

func getGenesis(genesisPath string) (*core.Genesis, error) {
	log.Info("Reading genesis at ", "file", genesisPath)
	file, err := os.Open(genesisPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		return nil, err
	}
	return genesis, nil
}

// SetBorConfig sets bor config
func SetBorConfig(ctx *cli.Context, cfg *eth.Config) {
	cfg.HeimdallURL = ctx.GlobalString(HeimdallURLFlag.Name)
	cfg.WithoutHeimdall = ctx.GlobalBool(WithoutHeimdallFlag.Name)
}

// CreateBorEthereum Creates bor ethereum object from eth.Config
func CreateBorEthereum(cfg *eth.Config) *eth.Ethereum {
	workspace, err := ioutil.TempDir("", "bor-command-node-")
	if err != nil {
		Fatalf("Failed to create temporary keystore: %v", err)
	}

	// Create a networkless protocol stack and start an Ethereum service within
	stack, err := node.New(&node.Config{DataDir: workspace, UseLightweightKDF: true, Name: "bor-command-node"})
	if err != nil {
		Fatalf("Failed to create node: %v", err)
	}
	ethereum, err := eth.New(stack, cfg)
	if err != nil {
		Fatalf("Failed to register Ethereum protocol: %v", err)
	}

	// Start the node and assemble the JavaScript console around it
	if err = stack.Start(); err != nil {
		Fatalf("Failed to start stack: %v", err)
	}
	_, err = stack.Attach()
	if err != nil {
		Fatalf("Failed to attach to node: %v", err)
	}

	return ethereum
}
