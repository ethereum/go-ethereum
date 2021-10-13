package server

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethstats"
	"github.com/ethereum/go-ethereum/graphql"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/cli"
)

// Command is the command to start the sever
type Command struct {
	UI cli.Ui

	// cli configuration
	cliConfig *Config

	configFile string

	// bor node
	node *node.Node
}

// Help implements the cli.Command interface
func (c *Command) Help() string {
	return `Usage: bor [options]
  
	Run the Bor server.
  ` + c.Flags().Help()
}

// Synopsis implements the cli.Command interface
func (c *Command) Synopsis() string {
	return "Run the Bor server"
}

// Run implements the cli.Command interface
func (c *Command) Run(args []string) int {
	flags := c.Flags()
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// read config file
	config := DefaultConfig()
	if c.configFile != "" {
		cfg, err := readConfigFile(c.configFile)
		if err != nil {
			c.UI.Error(err.Error())
			return 1
		}
		if err := config.Merge(cfg); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}
	if err := config.Merge(c.cliConfig); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// start the logger
	setupLogger(*config.LogLevel)

	// load the chain genesis
	if err := config.loadChain(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// create the node/stack
	nodeCfg, err := config.buildNode()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	stack, err := node.New(nodeCfg)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	c.node = stack

	// register the ethereum backend
	ethCfg, err := config.buildEth()
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	backend, err := eth.New(stack, ethCfg)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// debug tracing is enabled by default
	stack.RegisterAPIs(tracers.APIs(backend.APIBackend))

	// graphql is started from another place
	if *config.JsonRPC.Graphql.Enabled {
		if err := graphql.New(stack, backend.APIBackend, config.JsonRPC.Cors, config.JsonRPC.Modules); err != nil {
			c.UI.Error(fmt.Sprintf("Failed to register the GraphQL service: %v", err))
			return 1
		}
	}

	// register ethash service
	if config.EthStats != nil {
		if err := ethstats.New(stack, backend.APIBackend, backend.Engine(), *config.EthStats); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	// setup account manager (only keystore)
	{
		keydir := stack.KeyStoreDir()
		n, p := keystore.StandardScryptN, keystore.StandardScryptP
		if *config.UseLightweightKDF {
			n, p = keystore.LightScryptN, keystore.LightScryptP
		}
		stack.AccountManager().AddBackend(keystore.NewKeyStore(keydir, n, p))
	}

	// sealing (if enabled)
	if *config.Sealer.Enabled {
		if err := backend.StartMining(1); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	if err := c.setupMetrics(config.Metrics); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	// start the node
	if err := c.node.Start(); err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	return c.handleSignals()
}

func (c *Command) setupMetrics(config *MetricsConfig) error {
	metrics.Enabled = *config.Enabled
	metrics.EnabledExpensive = *config.Expensive

	if !metrics.Enabled {
		// metrics are disabled, do not set up any sink
		return nil
	}

	log.Info("Enabling metrics collection")

	// influxdb
	if v1Enabled, v2Enabled := (*config.InfluxDB.V1Enabled), (*config.InfluxDB.V2Enabled); v1Enabled || v2Enabled {
		if v1Enabled && v2Enabled {
			return fmt.Errorf("both influx v1 and influx v2 cannot be enabled")
		}

		cfg := config.InfluxDB
		tags := *cfg.Tags
		endpoint := *cfg.Endpoint

		if v1Enabled {
			log.Info("Enabling metrics export to InfluxDB (v1)")
			go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, *cfg.Database, *cfg.Username, *cfg.Password, "geth.", tags)
		}
		if v2Enabled {
			log.Info("Enabling metrics export to InfluxDB (v2)")
			go influxdb.InfluxDBV2WithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, *cfg.Token, *cfg.Bucket, *cfg.Organization, "geth.", tags)
		}
	}

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)

	return nil
}

func (c *Command) handleSignals() int {
	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalCh

	c.UI.Output(fmt.Sprintf("Caught signal: %v", sig))
	c.UI.Output("Gracefully shutting down agent...")

	gracefulCh := make(chan struct{})
	go func() {
		c.node.Close()
		close(gracefulCh)
	}()

	select {
	case <-signalCh:
		return 1
	case <-time.After(5 * time.Second):
		return 1
	case <-gracefulCh:
		return 0
	}
}

func setupLogger(logLevel string) {
	output := io.Writer(os.Stderr)
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream := log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger := log.NewGlogHandler(ostream)

	// logging
	lvl, err := log.LvlFromString(strings.ToLower(logLevel))
	if err == nil {
		glogger.Verbosity(lvl)
	} else {
		glogger.Verbosity(log.LvlInfo)
	}
	log.Root().SetHandler(glogger)
}
