package utils

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
)

var (
	//
	// Bor Specific flags
	//

	// HeimdallURLFlag flag for heimdall url
	HeimdallURLFlag = &cli.StringFlag{
		Name:  "bor.heimdall",
		Usage: "URL of Heimdall service",
		Value: "http://localhost:1317",
	}

	// WithoutHeimdallFlag no heimdall (for testing purpose)
	WithoutHeimdallFlag = &cli.BoolFlag{
		Name:  "bor.withoutheimdall",
		Usage: "Run without Heimdall service (for testing purpose)",
	}

	// HeimdallgRPCAddressFlag flag for heimdall gRPC address
	HeimdallgRPCAddressFlag = &cli.StringFlag{
		Name:  "bor.heimdallgRPC",
		Usage: "Address of Heimdall gRPC service",
		Value: "",
	}

	// RunHeimdallFlag flag for running heimdall internally from bor
	RunHeimdallFlag = &cli.BoolFlag{
		Name:  "bor.runheimdall",
		Usage: "Run Heimdall service as a child process",
	}

	RunHeimdallArgsFlag = &cli.StringFlag{
		Name:  "bor.runheimdallargs",
		Usage: "Arguments to pass to Heimdall service",
		Value: "",
	}

	// UseHeimdallApp flag for using internal heimdall app to fetch data
	UseHeimdallAppFlag = &cli.BoolFlag{
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

// SetBorConfig sets bor config
func SetBorConfig(ctx *cli.Context, cfg *eth.Config) {
	cfg.HeimdallURL = ctx.String(HeimdallURLFlag.Name)
	cfg.WithoutHeimdall = ctx.Bool(WithoutHeimdallFlag.Name)
	cfg.HeimdallgRPCAddress = ctx.String(HeimdallgRPCAddressFlag.Name)
	cfg.RunHeimdall = ctx.Bool(RunHeimdallFlag.Name)
	cfg.RunHeimdallArgs = ctx.String(RunHeimdallArgsFlag.Name)
	cfg.UseHeimdallApp = ctx.Bool(UseHeimdallAppFlag.Name)
}

// CreateBorEthereum Creates bor ethereum object from eth.Config
func CreateBorEthereum(cfg *ethconfig.Config) *eth.Ethereum {
	workspace, err := os.MkdirTemp("", "bor-command-node-")
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

	stack.Attach()

	return ethereum
}
