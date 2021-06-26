package plugins

import (
	"plugin"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
	"flag"
	"io/ioutil"
	"strings"
	"path"
	"fmt"
	"reflect"
)


type Subcommand func(*cli.Context, []string) error


type PluginLoader struct{
	Plugins []*plugin.Plugin
	Subcommands map[string]Subcommand
	Flags []*flag.FlagSet
	LookupCache map[string][]interface{}
}

func (pl *PluginLoader) Lookup(name string, validate func(interface{}) bool) []interface{} {
	if v, ok := pl.LookupCache[name]; ok { return v }
	results := []interface{}{}
	for _, plugin := range pl.Plugins {
		if v, err := plugin.Lookup(name); err == nil {
			if validate(v) {
				results = append(results, v)
			}
		}
	}
	pl.LookupCache[name] = results
	return results
}

func Lookup(name string, validate func(interface{}) bool) []interface{} {
	if DefaultPluginLoader == nil {
		log.Warn("Lookup attempted, but PluginLoader is not initialized", "name", name)
		return []interface{}{}
	}
	return DefaultPluginLoader.Lookup(name, validate)
}


var DefaultPluginLoader *PluginLoader


func NewPluginLoader(target string) (*PluginLoader, error) {
	pl := &PluginLoader{
		Plugins: []*plugin.Plugin{},
		// RPCPlugins: []APILoader{},
		Subcommands: make(map[string]Subcommand),
		Flags: []*flag.FlagSet{},
		LookupCache: make(map[string][]interface{}),
		// CreateConsensusEngine: ethconfig.CreateConsensusEngine,
		// UpdateBlockchainVMConfig: func(cfg *vm.Config) {},
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
		pl.Plugins = append(pl.Plugins, plug)
	}
	return pl, nil
}

func Initialize(target string) (err error) {
	DefaultPluginLoader, err = NewPluginLoader(target)
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
	if DefaultPluginLoader == nil { return false, fmt.Errorf("Plugin loader not initialized") }
	return DefaultPluginLoader.RunSubcommand(ctx)
}

func (pl *PluginLoader) ParseFlags(args []string) bool {
	for _, flagset := range pl.Flags {
		flagset.Parse(args)
	}
	return len(pl.Flags) > 0
}

func ParseFlags(args []string) bool {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting to parse flags, but default PluginLoader has not been initialized")
		return false
	}
	return DefaultPluginLoader.ParseFlags(args)
}
