package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"gopkg.in/urfave/cli.v1"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
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

	// HeimdallgRPCAddressFlag flag for heimdall gRPC address
	HeimdallgRPCAddressFlag = cli.StringFlag{
		Name:  "bor.heimdallgRPC",
		Usage: "Address of Heimdall gRPC service",
		Value: "",
	}

	// RunHeimdallFlag flag for running heimdall internally from bor
	RunHeimdallFlag = cli.BoolFlag{
		Name:  "bor.runheimdall",
		Usage: "Run Heimdall service as a child process",
	}

	RunHeimdallArgsFlag = cli.StringFlag{
		Name:  "bor.runheimdallargs",
		Usage: "Arguments to pass to Heimdall service",
		Value: "",
	}

	// UseHeimdallApp flag for using internall heimdall app to fetch data
	UseHeimdallAppFlag = cli.BoolFlag{
		Name:  "bor.useheimdallapp",
		Usage: "Use child heimdall process to fetch data, Only works when bor.runheimdall is true",
	}

	// BorFlags all bor related flags
	BorFlags = []cli.Flag{
		HeimdallURLFlag,
		WithoutHeimdallFlag,
		HeimdallgRPCAddressFlag,
		RunHeimdallFlag,
		RunHeimdallArgsFlag,
		UseHeimdallAppFlag,
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
	cfg.HeimdallgRPCAddress = ctx.GlobalString(HeimdallgRPCAddressFlag.Name)
	cfg.RunHeimdall = ctx.GlobalBool(RunHeimdallFlag.Name)
	cfg.RunHeimdallArgs = ctx.GlobalString(RunHeimdallArgsFlag.Name)
	cfg.UseHeimdallApp = ctx.GlobalBool(UseHeimdallAppFlag.Name)
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
