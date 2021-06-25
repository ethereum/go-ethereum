package plugins

import (
	"plugin"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/state"
	"gopkg.in/urfave/cli.v1"
	"flag"
	"io/ioutil"
	"strings"
	"path"
	"fmt"
	"reflect"
)


type APILoader func(*node.Node, Backend) []rpc.API
type Subcommand func(*cli.Context, []string) error
type TracerResult interface {
	vm.Tracer
	GetResult() (interface{}, error)
}


type PluginLoader struct{
	Tracers map[string]func(*state.StateDB)TracerResult
	StateHooks []interface{} // TODO: Set interface
	ChainEventHooks []interface{} // TODO: Set interface
	RPCPlugins []APILoader
	Subcommands map[string]Subcommand
	Flags []*flag.FlagSet
}

var defaultPluginLoader *PluginLoader


func NewPluginLoader(target string) (*PluginLoader, error) {
	pl := &PluginLoader{
		RPCPlugins: []APILoader{},
		Subcommands: make(map[string]Subcommand),
		Tracers: make(map[string]func(*state.StateDB)TracerResult),
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
			log.Debug("File inplugin directory is not '.so' file. Skipping.", "file", fpath)
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
		fn, err := plug.Lookup("GetAPIs")
		if err == nil {
			apiLoader, ok := fn.(func(*node.Node, Backend) []rpc.API)
			if !ok {
				log.Warn("Could not cast plugin.GetAPIs to APILoader", "file", fpath)
			} else {
				pl.RPCPlugins = append(pl.RPCPlugins, APILoader(apiLoader))
			}
		} else { log.Debug("Error retrieving GetAPIs for plugin", "file", fpath, "error", err.Error()) }

		sb, err := plug.Lookup("Subcommands")
		if err == nil {
			subcommands, ok := sb.(*map[string]func(*cli.Context, []string) error)
			if !ok {
				log.Warn("Could not cast plugin.Subcommands to `map[string]func(*cli.Context, []string) error`", "file", fpath, "type", reflect.TypeOf(sb))
			} else {
				for k, v := range *subcommands {
					if _, ok := pl.Subcommands[k]; ok {
						log.Warn("Subcommand redeclared", "file", fpath, "subcommand", k)
					}
					pl.Subcommands[k] = v
				}
			}
		}
		tr, err := plug.Lookup("Tracers")
		if err == nil {
			tracers, ok := tr.(*map[string]func(*state.StateDB)TracerResult)
			if !ok {
				log.Warn("Could not cast plugin.Tracers to `map[string]vm.Tracer`", "file", fpath)
			} else {
				for k, v := range *tracers {
					if _, ok := pl.Tracers[k]; ok {
						log.Warn("Tracer redeclared", "file", fpath, "tracer", k)
					}
					pl.Tracers[k] = v
				}
			}
		}
	}
	return pl, nil
}

func Initialize(target string) (err error) {
	defaultPluginLoader, err = NewPluginLoader(target)
	return err
}

func (pl *PluginLoader) RunSubcommand(ctx *cli.Context) (bool, error) {
	args := ctx.Args()
	if len(args) == 0 { return false, fmt.Errorf("No subcommand arguments")}
	subcommand, ok := pl.Subcommands[args[0]]
	if !ok { return false, fmt.Errorf("Subcommand %v does not exist", args[0])}
	return true, subcommand(ctx, args[1:])
}

func RunSubcommand(ctx *cli.Context) (bool, error) {
	if defaultPluginLoader == nil { return false, fmt.Errorf("Plugin loader not initialized") }
	return defaultPluginLoader.RunSubcommand(ctx)
}

func (pl *PluginLoader) ParseFlags(args []string) bool {
	for _, flagset := range pl.Flags {
		flagset.Parse(args)
	}
	return len(pl.Flags) > 0
}

func ParseFlags(args []string) bool {
	if defaultPluginLoader == nil {
		log.Warn("Attempting to parse flags, but default PluginLoader has not been initialized")
		return false
	}
	return defaultPluginLoader.ParseFlags(args)
}

func (pl *PluginLoader) GetAPIs(stack *node.Node, backend Backend) []rpc.API {
	apis := []rpc.API{}
	for _, apiLoader := range pl.RPCPlugins {
		apis = append(apis, apiLoader(stack, backend)...)
	}
	return apis
}

func GetAPIs(stack *node.Node, backend Backend) []rpc.API {
	if defaultPluginLoader == nil {
		log.Warn("Attempting GetAPIs, but default PluginLoader has not been initialized")
		return []rpc.API{}
	 }
	return defaultPluginLoader.GetAPIs(stack, backend)
}

func (pl *PluginLoader) GetTracer(s string) (func(*state.StateDB)TracerResult, bool) {
	tr, ok := pl.Tracers[s]
	return tr, ok
}

func GetTracer(s string) (func(*state.StateDB)TracerResult, bool) {
	if defaultPluginLoader == nil {
		log.Warn("Attempting GetTracer, but default PluginLoader has not been initialized")
		return nil, false
	}
	return defaultPluginLoader.GetTracer(s)
}
