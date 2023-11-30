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
	"google.golang.org/grpc/credentials/insecure"
)

const (
	emptyPlaceHolder = "<none>"
)

type MarkDownCommand interface {
	MarkDown
	cli.Command
}

type MarkDownCommandFactory func() (MarkDownCommand, error)

func Run(args []string) int {
	commands := Commands()

	mappedCommands := make(map[string]cli.CommandFactory)

	for k, v := range commands {
		// Declare a new v to limit the scope of v to inside the block, so the anonymous function below
		// can get the "current" value of v, instead of the value of last v in the loop.
		// See this post: https://stackoverflow.com/questions/10116507/go-transfer-var-into-anonymous-function for more explanation
		v := v
		mappedCommands[k] = func() (cli.Command, error) {
			cmd, err := v()
			return cmd.(cli.Command), err
		}
	}

	cli := &cli.CLI{
		Name:     "bor",
		Args:     args,
		Commands: mappedCommands,
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

func Commands() map[string]MarkDownCommandFactory {
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

	return map[string]MarkDownCommandFactory{
		"server": func() (MarkDownCommand, error) {
			return &server.Command{
				UI: ui,
			}, nil
		},
		"version": func() (MarkDownCommand, error) {
			return &VersionCommand{
				UI: ui,
			}, nil
		},
		"dumpconfig": func() (MarkDownCommand, error) {
			return &DumpconfigCommand{
				Meta2: meta2,
			}, nil
		},
		"debug": func() (MarkDownCommand, error) {
			return &DebugCommand{
				UI: ui,
			}, nil
		},
		"debug pprof": func() (MarkDownCommand, error) {
			return &DebugPprofCommand{
				Meta2: meta2,
			}, nil
		},
		"debug block": func() (MarkDownCommand, error) {
			return &DebugBlockCommand{
				Meta2: meta2,
			}, nil
		},
		"chain": func() (MarkDownCommand, error) {
			return &ChainCommand{
				UI: ui,
			}, nil
		},
		"chain watch": func() (MarkDownCommand, error) {
			return &ChainWatchCommand{
				Meta2: meta2,
			}, nil
		},
		"chain sethead": func() (MarkDownCommand, error) {
			return &ChainSetHeadCommand{
				Meta2: meta2,
			}, nil
		},
		"account": func() (MarkDownCommand, error) {
			return &Account{
				UI: ui,
			}, nil
		},
		"account new": func() (MarkDownCommand, error) {
			return &AccountNewCommand{
				Meta: meta,
			}, nil
		},
		"account import": func() (MarkDownCommand, error) {
			return &AccountImportCommand{
				Meta: meta,
			}, nil
		},
		"account list": func() (MarkDownCommand, error) {
			return &AccountListCommand{
				Meta: meta,
			}, nil
		},
		"peers": func() (MarkDownCommand, error) {
			return &PeersCommand{
				UI: ui,
			}, nil
		},
		"peers add": func() (MarkDownCommand, error) {
			return &PeersAddCommand{
				Meta2: meta2,
			}, nil
		},
		"peers remove": func() (MarkDownCommand, error) {
			return &PeersRemoveCommand{
				Meta2: meta2,
			}, nil
		},
		"peers list": func() (MarkDownCommand, error) {
			return &PeersListCommand{
				Meta2: meta2,
			}, nil
		},
		"peers status": func() (MarkDownCommand, error) {
			return &PeersStatusCommand{
				Meta2: meta2,
			}, nil
		},
		"status": func() (MarkDownCommand, error) {
			return &StatusCommand{
				Meta2: meta2,
			}, nil
		},
		"fingerprint": func() (MarkDownCommand, error) {
			return &FingerprintCommand{
				UI: ui,
			}, nil
		},
		"attach": func() (MarkDownCommand, error) {
			return &AttachCommand{
				UI:    ui,
				Meta:  meta,
				Meta2: meta2,
			}, nil
		},
		"bootnode": func() (MarkDownCommand, error) {
			return &BootnodeCommand{
				UI: ui,
			}, nil
		},
		"removedb": func() (MarkDownCommand, error) {
			return &RemoveDBCommand{
				Meta2: meta2,
			}, nil
		},
		"snapshot": func() (MarkDownCommand, error) {
			return &SnapshotCommand{
				UI: ui,
			}, nil
		},
		"snapshot prune-state": func() (MarkDownCommand, error) {
			return &PruneStateCommand{
				Meta: meta,
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
	conn, err := grpc.Dial(m.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		Usage: "Path of the data directory to store keys",
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
	columnConf.Empty = emptyPlaceHolder

	return columnize.Format(in, columnConf)
}

func formatKV(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = emptyPlaceHolder
	columnConf.Glue = " = "

	return columnize.Format(in, columnConf)
}
