package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
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

	// final configuration
	config *Config

	configFile []string

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
	for _, configFile := range c.configFile {
		cfg, err := readConfigFile(configFile)
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
	c.config = config

	// start the logger
	setupLogger(config.LogLevel)

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

	log.Info("Heimdall setup", "url", ethCfg.HeimdallURL)

	// debug tracing is enabled by default
	stack.RegisterAPIs(tracers.APIs(backend.APIBackend))

	// graphql is started from another place
	if config.JsonRPC.Graphql.Enabled {
		if err := graphql.New(stack, backend.APIBackend, config.JsonRPC.Cors, config.JsonRPC.Modules); err != nil {
			c.UI.Error(fmt.Sprintf("Failed to register the GraphQL service: %v", err))
			return 1
		}
	}

	// register ethash service
	if config.Ethstats != "" {
		if err := ethstats.New(stack, backend.APIBackend, backend.Engine(), config.Ethstats); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	// setup account manager (only keystore)
	var borKeystore *keystore.KeyStore
	{
		keydir := stack.KeyStoreDir()
		n, p := keystore.StandardScryptN, keystore.StandardScryptP
		if config.Accounts.UseLightweightKDF {
			n, p = keystore.LightScryptN, keystore.LightScryptP
		}
		borKeystore = keystore.NewKeyStore(keydir, n, p)
		stack.AccountManager().AddBackend(borKeystore)
	}

	// unlock accounts if necessary
	if len(config.Accounts.Unlock) != 0 {
		if err := c.unlockAccounts(borKeystore); err != nil {
			c.UI.Error(fmt.Sprintf("failed to unlock: %v", err))
			return 1
		}
	}

	// sealing (if enabled)
	if config.Sealer.Enabled {
		if err := backend.StartMining(1); err != nil {
			c.UI.Error(err.Error())
			return 1
		}
	}

	if err := c.setupTelemetry(config.Telemetry); err != nil {
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

func (c *Command) unlockAccounts(borKeystore *keystore.KeyStore) error {
	// If insecure account unlocking is not allowed if node's APIs are exposed to external.
	if !c.node.Config().InsecureUnlockAllowed && c.node.Config().ExtRPCEnabled() {
		return fmt.Errorf("account unlock with HTTP access is forbidden")
	}

	// read passwords from file if possible
	passwords := []string{}
	if c.config.Accounts.PasswordFile != "" {
		var err error
		if passwords, err = readMultilineFile(c.config.Accounts.PasswordFile); err != nil {
			return err
		}
	}
	decodePassword := func(addr common.Address, index int) (string, error) {
		if len(passwords) > 0 {
			if index < len(passwords) {
				return passwords[index], nil
			}
			return passwords[len(passwords)-1], nil
		}
		// ask for the password
		return c.UI.AskSecret(fmt.Sprintf("Please give a password to unlock '%s'", addr.String()))
	}

	for index, addrStr := range c.config.Accounts.Unlock {
		if !common.IsHexAddress(addrStr) {
			return fmt.Errorf("unlock value '%s' is not an address", addrStr)
		}
		acct := accounts.Account{Address: common.HexToAddress(addrStr)}

		password, err := decodePassword(acct.Address, index)
		if err != nil {
			return err
		}
		if err := borKeystore.Unlock(acct, password); err != nil {
			return err
		}
		log.Info("Unlocked account", "address", acct.Address.Hex())
	}
	return nil
}

func (c *Command) setupTelemetry(config *TelemetryConfig) error {
	metrics.Enabled = config.Enabled
	metrics.EnabledExpensive = config.Expensive

	if !metrics.Enabled {
		// metrics are disabled, do not set up any sink
		return nil
	}

	log.Info("Enabling metrics collection")

	// influxdb
	if v1Enabled, v2Enabled := (config.InfluxDB.V1Enabled), (config.InfluxDB.V2Enabled); v1Enabled || v2Enabled {
		if v1Enabled && v2Enabled {
			return fmt.Errorf("both influx v1 and influx v2 cannot be enabled")
		}

		cfg := config.InfluxDB
		tags := cfg.Tags
		endpoint := cfg.Endpoint

		if v1Enabled {
			log.Info("Enabling metrics export to InfluxDB (v1)")
			go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, cfg.Database, cfg.Username, cfg.Password, "geth.", tags)
		}
		if v2Enabled {
			log.Info("Enabling metrics export to InfluxDB (v2)")
			go influxdb.InfluxDBV2WithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, cfg.Token, cfg.Bucket, cfg.Organization, "geth.", tags)
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
		c.node.Wait()
		close(gracefulCh)
	}()

	for i := 10; i > 0; i-- {
		select {
		case <-signalCh:
			log.Warn("Already shutting down, interrupt more force stop.", "times", i-1)
		case <-gracefulCh:
			return 0
		}
	}
	return 1
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

func readMultilineFile(path string) ([]string, error) {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines, nil
}
