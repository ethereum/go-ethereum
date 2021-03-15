package plugins

import (
	"plugin"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
	"flag"
	"io/ioutil"
	"strings"
	"path"
	"fmt"
)


type APILoader func(*node.Node, Backend) []rpc.API
type Subcommand func(*cli.Context, []string) error

type PluginType int

const (
	TracerPluginType PluginType = iota
	StateHookType
	ChainEventHookType
	RPCPluginType
	SubcommandType
)


type PluginLoader struct{
	TracerPlugins map[string]interface{} // TODO: Set interface
	StateHooks []interface{} // TODO: Set interface
	ChainEventHooks []interface{} // TODO: Set interface
	RPCPlugins []APILoader
	Subcommands map[string]Subcommand
	Flags []*flag.FlagSet
}


func NewPluginLoader(target string) (*PluginLoader, error) {
	pl := &PluginLoader{
		RPCPlugins: []APILoader{},
		Subcommands: make(map[string]Subcommand),
		Flags: []*flag.FlagSet{},
	}
	files, err := ioutil.ReadDir(target)
	if err != nil {
		log.Warn("Could not load plugins directory. Skipping.", "path", target)
		return pl, nil
	}
	for _, file := range files {
		fpath := path.Join(target, file.Name())
		if !strings.HasSuffix(file.Name(), ".so") {
			log.Warn("File inplugin directory is not '.so' file. Skipping.", "file", fpath)
			continue
		}
		plug, err := plugin.Open(fpath)
		if err != nil {
			log.Warn("File in plugin directory could not be loaded: %v", "file", fpath, "error", err.Error())
			continue
		}
		// Any type of plugin can potentially specify flags
		f, err := plug.Lookup("Flags")
		if err == nil {
			flagset, ok := f.(*flag.FlagSet)
			if !ok {
				log.Warn("Found plugin.Flags, but it its not a *FlagSet", "file", fpath)
			} else {
				pl.Flags = append(pl.Flags, flagset)
			}
		}
		t, err := plug.Lookup("Type")
		if err != nil {
			log.Warn("Could not load plugin Type", "file", fpath, "error", err.Error())
			continue
		}
		switch pt := t.(*PluginType); *pt {
		case RPCPluginType:
			fn, err := plug.Lookup("GetAPIs")
			if err != nil {
				log.Warn("Could not load GetAPIs from plugin", "file", fpath, "error", err.Error())
				continue
			}
			apiLoader, ok := fn.(func(*node.Node, Backend) []rpc.API)
			if !ok {
				log.Warn("Could not cast plugin.GetAPIs to APILoader", "file", fpath)
				continue
			}
			pl.RPCPlugins = append(pl.RPCPlugins, APILoader(apiLoader))
		case SubcommandType:
			n, err := plug.Lookup("Name")
			if err != nil {
				log.Warn("Could not load Name from subcommand plugin", "file", fpath, "error", err.Error())
				continue
			}
			name, ok := n.(*string)
			if !ok {
				log.Warn("Could not cast plugin.Name to string", "file", fpath)
				continue
			}
			fn, err := plug.Lookup("Main")
			if err != nil {
				log.Warn("Could not load Main from plugin", "file", fpath, "error", err.Error())
				continue
			}
			subcommand, ok := fn.(func(*cli.Context, []string) error)
			if !ok {
				log.Warn("Could not cast plugin.Main to Subcommand", "file", fpath)
				continue
			}
			if _, ok := pl.Subcommands[*name]; ok {
				return pl, fmt.Errorf("Multiple subcommand plugins with the same name: %v", *name)
			}
			pl.Subcommands[*name] = subcommand
		}
	}
	return pl, nil
}

func (pl *PluginLoader) RunSubcommand(ctx *cli.Context) (bool, error) {
	args := ctx.Args()
	if len(args) == 0 { return false, fmt.Errorf("No subcommand arguments")}
	subcommand, ok := pl.Subcommands[args[0]]
	if !ok { return false, fmt.Errorf("Subcommand %v does not exist", args[0])}
	return true, subcommand(ctx, args[1:])
}

func (pl *PluginLoader) ParseFlags(args []string) bool {
	for _, flagset := range pl.Flags {
		flagset.Parse(args)
	}
	return len(pl.Flags) > 0
}

func (pl *PluginLoader) GetAPIs(stack *node.Node, backend Backend) []rpc.API {
	apis := []rpc.API{}
	for _, apiLoader := range pl.RPCPlugins {
		apis = append(apis, apiLoader(stack, backend)...)
	}
	return apis
}
