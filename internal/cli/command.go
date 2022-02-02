package cli

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mitchellh/cli"
	"github.com/ryanuber/columnize"
	"google.golang.org/grpc"
)

func Run(args []string) int {
	commands := commands()

	cli := &cli.CLI{
		Name:     "bor",
		Args:     args,
		Commands: commands,
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}
	return exitCode
}

func commands() map[string]cli.CommandFactory {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	meta2 := &Meta2{
		UI: ui,
	}
	meta := &Meta{
		UI: ui,
	}
	return map[string]cli.CommandFactory{
		"server": func() (cli.Command, error) {
			return &server.Command{
				UI: ui,
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				UI: ui,
			}, nil
		},
		"debug": func() (cli.Command, error) {
			return &DebugCommand{
				Meta2: meta2,
			}, nil
		},
		"chain": func() (cli.Command, error) {
			return &ChainCommand{
				UI: ui,
			}, nil
		},
		"chain watch": func() (cli.Command, error) {
			return &ChainWatchCommand{
				Meta2: meta2,
			}, nil
		},
		"chain sethead": func() (cli.Command, error) {
			return &ChainSetHeadCommand{
				Meta2: meta2,
			}, nil
		},
		"account": func() (cli.Command, error) {
			return &Account{
				UI: ui,
			}, nil
		},
		"account new": func() (cli.Command, error) {
			return &AccountNewCommand{
				Meta: meta,
			}, nil
		},
		"account import": func() (cli.Command, error) {
			return &AccountImportCommand{
				Meta: meta,
			}, nil
		},
		"account list": func() (cli.Command, error) {
			return &AccountListCommand{
				Meta: meta,
			}, nil
		},
		"peers": func() (cli.Command, error) {
			return &PeersCommand{
				UI: ui,
			}, nil
		},
		"peers add": func() (cli.Command, error) {
			return &PeersAddCommand{
				Meta2: meta2,
			}, nil
		},
		"peers remove": func() (cli.Command, error) {
			return &PeersRemoveCommand{
				Meta2: meta2,
			}, nil
		},
		"peers list": func() (cli.Command, error) {
			return &PeersListCommand{
				Meta2: meta2,
			}, nil
		},
		"peers status": func() (cli.Command, error) {
			return &PeersStatusCommand{
				Meta2: meta2,
			}, nil
		},
		"status": func() (cli.Command, error) {
			return &StatusCommand{
				Meta2: meta2,
			}, nil
		},
	}
}

type Meta2 struct {
	UI cli.Ui

	addr string
}

func (m *Meta2) NewFlagSet(n string) *flagset.Flagset {
	f := flagset.NewFlagSet(n)

	f.StringFlag(&flagset.StringFlag{
		Name:    "address",
		Value:   &m.addr,
		Usage:   "Address of the grpc endpoint",
		Default: "127.0.0.1:3131",
	})
	return f
}

func (m *Meta2) Conn() (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(m.addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	return conn, nil
}

func (m *Meta2) BorConn() (proto.BorClient, error) {
	conn, err := m.Conn()
	if err != nil {
		return nil, err
	}
	return proto.NewBorClient(conn), nil
}

// Meta is a helper utility for the commands
type Meta struct {
	UI cli.Ui

	dataDir     string
	keyStoreDir string
}

func (m *Meta) NewFlagSet(n string) *flagset.Flagset {
	f := flagset.NewFlagSet(n)

	f.StringFlag(&flagset.StringFlag{
		Name:  "datadir",
		Value: &m.dataDir,
		Usage: "Path of the data directory to store information",
	})
	f.StringFlag(&flagset.StringFlag{
		Name:  "keystore",
		Value: &m.keyStoreDir,
		Usage: "Path of the data directory to store information",
	})

	return f
}

func (m *Meta) AskPassword() (string, error) {
	return m.UI.AskSecret("Your new account is locked with a password. Please give a password. Do not forget this password")
}

func (m *Meta) GetKeystore() (*keystore.KeyStore, error) {
	cfg := node.DefaultConfig
	cfg.DataDir = m.dataDir
	cfg.KeyStoreDir = m.keyStoreDir

	stack, err := node.New(&cfg)
	if err != nil {
		return nil, err
	}

	keydir := stack.KeyStoreDir()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP

	keys := keystore.NewKeyStore(keydir, scryptN, scryptP)
	return keys, nil
}

func formatList(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	return columnize.Format(in, columnConf)
}

func formatKV(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	columnConf.Glue = " = "
	return columnize.Format(in, columnConf)
}
