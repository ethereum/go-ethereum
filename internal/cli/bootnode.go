package cli

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/cmd/bootnode"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/prometheus"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"

	"github.com/mitchellh/cli"
)

type BootnodeCommand struct {
	UI cli.Ui

	listenAddr     string
	enableMetrics  bool
	prometheusAddr string
	v5             bool
	verbosity      int
	logLevel       string
	nat            string
	nodeKey        string
	saveKey        string
	dryRun         bool
}

// Help implements the cli.Command interface
func (b *BootnodeCommand) Help() string {
	return `Usage: bor bootnode`
}

// MarkDown implements cli.MarkDown interface
func (c *BootnodeCommand) MarkDown() string {
	items := []string{
		"# Bootnode",
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

func (b *BootnodeCommand) Flags() *flagset.Flagset {
	flags := flagset.NewFlagSet("bootnode")

	flags.StringFlag(&flagset.StringFlag{
		Name:    "listen-addr",
		Default: "0.0.0.0:30303",
		Usage:   "listening address of bootnode (<ip>:<port>)",
		Value:   &b.listenAddr,
	})
	flags.BoolFlag(&flagset.BoolFlag{
		Name:    "metrics",
		Usage:   "Enable metrics collection and reporting",
		Value:   &b.enableMetrics,
		Default: true,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:    "prometheus-addr",
		Default: "127.0.0.1:7071",
		Usage:   "listening address of bootnode (<ip>:<port>)",
		Value:   &b.prometheusAddr,
	})
	flags.BoolFlag(&flagset.BoolFlag{
		Name:    "v5",
		Default: false,
		Usage:   "Enable UDP v5",
		Value:   &b.v5,
	})
	flags.IntFlag(&flagset.IntFlag{
		Name:    "verbosity",
		Default: 3,
		Usage:   "Logging verbosity (5=trace|4=debug|3=info|2=warn|1=error|0=crit)",
		Value:   &b.verbosity,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:    "log-level",
		Default: "info",
		Usage:   "log level (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead",
		Value:   &b.logLevel,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:    "nat",
		Default: "none",
		Usage:   "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value:   &b.nat,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:    "node-key",
		Default: "",
		Usage:   "file or hex node key",
		Value:   &b.nodeKey,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:    "save-key",
		Default: "",
		Usage:   "path to save the ecdsa private key",
		Value:   &b.saveKey,
	})
	flags.BoolFlag(&flagset.BoolFlag{
		Name:    "dry-run",
		Default: false,
		Usage:   "validates parameters and prints bootnode configurations, but does not start bootnode",
		Value:   &b.dryRun,
	})

	return flags
}

// Synopsis implements the cli.Command interface
func (b *BootnodeCommand) Synopsis() string {
	return "Start a bootnode"
}

// Run implements the cli.Command interface
// nolint: gocognit
func (b *BootnodeCommand) Run(args []string) int {
	flags := b.Flags()
	if err := flags.Parse(args); err != nil {
		b.UI.Error(err.Error())
		return 1
	}

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))

	var logInfo string

	if b.verbosity != 0 && b.logLevel != "" {
		b.UI.Warn(fmt.Sprintf("Both verbosity and log-level provided, using verbosity: %v", b.verbosity))
		logInfo = server.VerbosityIntToString(b.verbosity)
	} else if b.verbosity != 0 {
		logInfo = server.VerbosityIntToString(b.verbosity)
	} else {
		logInfo = b.logLevel
	}

	lvl, err := log.LvlFromString(strings.ToLower(logInfo))
	if err == nil {
		glogger.Verbosity(lvl)
	} else {
		glogger.Verbosity(log.LvlInfo)
	}

	log.Root().SetHandler(glogger)

	natm, err := nat.Parse(b.nat)
	if err != nil {
		b.UI.Error(fmt.Sprintf("failed to parse nat: %v", err))
		return 1
	}

	// create a one time key
	var nodeKey *ecdsa.PrivateKey
	// nolint: nestif
	if b.nodeKey != "" {
		// try to read the key either from file or command line
		if _, err := os.Stat(b.nodeKey); errors.Is(err, os.ErrNotExist) {
			if nodeKey, err = crypto.HexToECDSA(b.nodeKey); err != nil {
				b.UI.Error(fmt.Sprintf("failed to parse hex address: %v", err))
				return 1
			}
		} else {
			if nodeKey, err = crypto.LoadECDSA(b.nodeKey); err != nil {
				b.UI.Error(fmt.Sprintf("failed to load node key: %v", err))
				return 1
			}
		}
	} else {
		// generate a new temporal key
		if nodeKey, err = crypto.GenerateKey(); err != nil {
			b.UI.Error(fmt.Sprintf("could not generate key: %v", err))
			return 1
		}

		if b.saveKey != "" {
			path := b.saveKey

			// save the private key
			if err = crypto.SaveECDSA(filepath.Join(path, "priv.key"), nodeKey); err != nil {
				b.UI.Error(fmt.Sprintf("failed to write node priv key: %v", err))
				return 1
			}
			// save the public key
			pubRaw := fmt.Sprintf("%x", crypto.FromECDSAPub(&nodeKey.PublicKey)[1:])
			if err := os.WriteFile(filepath.Join(path, "pub.key"), []byte(pubRaw), 0600); err != nil {
				b.UI.Error(fmt.Sprintf("failed to write node pub key: %v", err))
				return 1
			}
		}
	}

	addr, err := net.ResolveUDPAddr("udp", b.listenAddr)
	if err != nil {
		b.UI.Error(fmt.Sprintf("could not resolve udp addr '%s': %v", b.listenAddr, err))
		return 1
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		b.UI.Error(fmt.Sprintf("failed to listen udp addr '%s': %v", b.listenAddr, err))
		return 1
	}
	defer conn.Close()

	db, _ := enode.OpenDB("")
	ln := enode.NewLocalNode(db, nodeKey)

	listenerAddr := conn.LocalAddr().(*net.UDPAddr)
	if natm != nil {
		natAddr := bootnode.DoPortMapping(natm, ln, listenerAddr)
		if natAddr != nil {
			listenerAddr = natAddr
		}
	}

	bootnode.PrintNotice(&nodeKey.PublicKey, *listenerAddr)

	cfg := discover.Config{
		PrivateKey: nodeKey,
		Log:        log.Root(),
	}

	if b.v5 {
		if _, err := discover.ListenV5(conn, ln, cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	} else {
		if _, err := discover.ListenUDP(conn, ln, cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	if b.enableMetrics {
		prometheusMux := http.NewServeMux()

		prometheusMux.Handle("/debug/metrics/prometheus", prometheus.Handler(metrics.DefaultRegistry))

		promServer := &http.Server{
			Addr:              b.prometheusAddr,
			Handler:           prometheusMux,
			ReadHeaderTimeout: 30 * time.Second,
		}

		go func() {
			if err := promServer.ListenAndServe(); err != nil {
				log.Error("Failure in running Prometheus server", "err", err)
			}
		}()

		log.Info("Enabling metrics export to prometheus", "path", fmt.Sprintf("http://%s/debug/metrics/prometheus", b.prometheusAddr))
	}

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-signalCh

	b.UI.Output(fmt.Sprintf("Caught signal: %v", sig))
	b.UI.Output("Gracefully shutting down agent...")

	return 0
}
