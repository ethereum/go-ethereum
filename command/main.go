package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/command/flagset"
	"github.com/ethereum/go-ethereum/command/server"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mitchellh/cli"
	"github.com/ryanuber/columnize"
)

func main() {
	os.Exit(Run(os.Args[1:]))
}

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
	}
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
