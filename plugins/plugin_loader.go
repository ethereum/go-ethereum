package plugins

import (
	"plugin"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	// "github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
	"flag"
	"io/ioutil"
	"strings"
	"path"
	"fmt"
	"reflect"
)


type Subcommand func(*cli.Context, []string) error
type TracerResult interface {
	vm.Tracer
	GetResult() (interface{}, error)
}


type PluginLoader struct{
	Plugins []plugin.Plugin
	Tracers map[string]func(StateDB)TracerResult
	StateHooks []interface{} // TODO: Set interface
	// RPCPlugins []APILoader
	Subcommands map[string]Subcommand
	Flags []*flag.FlagSet
	CreateConsensusEngine func(stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine
	UpdateBlockchainVMConfig func(*vm.Config)
	PreProcessBlockList []func(*types.Block)
	PreProcessTransactionList []func(*types.Transaction, *types.Block, int)
	BlockProcessingErrorList []func(*types.Transaction, *types.Block, error)
	PostProcessTransactionList []func(*types.Transaction, *types.Block, int, *types.Receipt)
	PostProcessBlockList []func(*types.Block)
}


var DefaultPluginLoader *PluginLoader


func NewPluginLoader(target string) (*PluginLoader, error) {
	pl := &PluginLoader{
		Plugins: []plugin.Plugin,
		// RPCPlugins: []APILoader{},
		Subcommands: make(map[string]Subcommand),
		Tracers: make(map[string]func(StateDB)TracerResult),
		Flags: []*flag.FlagSet{},
		// CreateConsensusEngine: ethconfig.CreateConsensusEngine,
		UpdateBlockchainVMConfig: func(cfg *vm.Config) {},
	}
	files, err := ioutil.ReadDir(target)
	if err != nil {
		log.Warn("Could not load plugins directory. Skipping.", "path", target)
		return pl, nil
	}
	setConsensus := false
	setUpdateBCVMCfg := false
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
		tr, err := plug.Lookup("Tracers")
		if err == nil {
			tracers, ok := tr.(*map[string]func(StateDB)TracerResult)
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
		ce, err := plug.Lookup("CreateConsensusEngine")
		if err == nil {
			cce, ok := ce.(func (stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine)
			if !ok {
				log.Warn("Could not cast plugin.CreateConsensusEngine to appropriate function", "file", fpath)
			} else {
				if setConsensus {
					log.Warn("CreateConsensusEngine redeclared", "file", fpath)
				}
				pl.CreateConsensusEngine = cce
				setConsensus = true
			}
		}
		vmcfgu, err := plug.Lookup("UpdateBlockchainVMConfig")
		if err == nil {
			vmcfgfn, ok := vmcfgu.(func(*vm.Config))
			if !ok {
				log.Warn("Could not cast plugin.UpdateBlockchainVMConfig to appropriate function", "file", fpath)
			} else {
				if setUpdateBCVMCfg {
					log.Warn("UpdateBlockchainVMConfig redeclared", "file", fpath)
				}
				pl.UpdateBlockchainVMConfig = vmcfgfn
				setUpdateBCVMCfg = true
			}
		}


		prepb, err := plug.Lookup("PreProcessBlock")
		if err == nil {
			prepbfn, ok := prepb.(func(*types.Block))
			if !ok {
				log.Warn("Could not cast plugin.PreProcessBlock to appropriate function", "file", fpath)
			} else {
				pl.PreProcessBlockList = append(pl.PreProcessBlockList, prepbfn)
			}
		}
		prept, err := plug.Lookup("PreProcessTransaction")
		if err == nil {
			preptfn, ok := prept.(func(*types.Transaction, *types.Block, int))
			if !ok {
				log.Warn("Could not cast plugin.PreProcessTransaction to appropriate function", "file", fpath)
			} else {
				pl.PreProcessTransactionList = append(pl.PreProcessTransactionList, preptfn)
			}
		}
		bpe, err := plug.Lookup("BlockProcessingError")
		if err == nil {
			bpefn, ok := bpe.(func(*types.Transaction, *types.Block, error))
			if !ok {
				log.Warn("Could not cast plugin.BlockProcessingError to appropriate function", "file", fpath)
			} else {
				pl.BlockProcessingErrorList = append(pl.BlockProcessingErrorList, bpefn)
			}
		}
		prept, err := plug.Lookup("PostProcessTransaction")
		if err == nil {
			preptfn, ok := prept.(func(*types.Transaction, *types.Block, int, *types.Receipt))
			if !ok {
				log.Warn("Could not cast plugin.PostProcessTransaction to appropriate function", "file", fpath)
			} else {
				pl.PostProcessTransactionList = append(pl.PostProcessTransactionList, preptfn)
			}
		}
		prepb, err := plug.Lookup("PostProcessBlock")
		if err == nil {
			prepbfn, ok := prepb.(func(*types.Block))
			if !ok {
				log.Warn("Could not cast plugin.PostProcessBlock to appropriate function", "file", fpath)
			} else {
				pl.PostProcessBlockList = append(pl.PostProcessBlockList, prepbfn)
			}
		}




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

func (pl *PluginLoader) GetTracer(s string) (func(StateDB)TracerResult, bool) {
	tr, ok := pl.Tracers[s]
	return tr, ok
}

func GetTracer(s string) (func(StateDB)TracerResult, bool) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting GetTracer, but default PluginLoader has not been initialized")
		return nil, false
	}
	return DefaultPluginLoader.GetTracer(s)
}

// func CreateConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, config *ethash.Config, notify []string, noverify bool, db ethdb.Database) consensus.Engine {
// 	if DefaultPluginLoader == nil {
// 		log.Warn("Attempting CreateConsensusEngine, but default PluginLoader has not been initialized")
// 		return ethconfig.CreateConsensusEngine(stack, chainConfig, config, notify, noverify, db)
// 	}
// 	return DefaultPluginLoader.CreateConsensusEngine(stack, chainConfig, config, notify, noverify, db)
// }

func UpdateBlockchainVMConfig(cfg *vm.Config) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting UpdateBlockchainVMConfig, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.UpdateBlockchainVMConfig(cfg)
}


func (pl *PluginLoader) PreProcessBlock(block *types.Block) {
	for _, fn := range pl.PreProcessBlockList {
		fn(block)
	}
}
func PreProcessBlock(block *types.Block) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting PreProcessBlock, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.PreProcessBlock(block)
}
func (pl *PluginLoader) PreProcessTransaction(tx *types.Transaction, block *types.Block, i int) {
	for _, fn := range pl.PreProcessTransactionList {
		fn(tx, block, i)
	}
}
func PreProcessTransaction(tx *types.Transaction, block *types.Block, i int) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting PreProcessTransaction, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.PreProcessTransaction(tx, block, i)
}
func (pl *PluginLoader) BlockProcessingError(tx *types.Transaction, block *types.Block, err error) {
	for _, fn := range pl.BlockProcessingErrorList {
		fn(tx, block, err)
	}
}
func BlockProcessingError(tx *types.Transaction, block *types.Block, err error) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting BlockProcessingError, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.BlockProcessingError(tx, block, err)
}
func (pl *PluginLoader) PostProcessTransaction(tx *types.Transaction, block *types.Block, i int, receipt *types.Receipt) {
	for _, fn := range pl.PostProcessTransactionList {
		fn(tx, block, i, receipt)
	}
}
func PostProcessTransaction(tx *types.Transaction, block *types.Block, i int, receipt *types.Receipt) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting PostProcessTransaction, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.PostProcessTransaction(tx, block, i, receipt)
}
func (pl *PluginLoader) PostProcessBlock(block *types.Block) {
	for _, fn := range pl.PostProcessBlockList {
		fn(block)
	}
}
func PostProcessBlock(block *types.Block) {
	if DefaultPluginLoader == nil {
		log.Warn("Attempting PostProcessBlock, but default PluginLoader has not been initialized")
		return
	}
	DefaultPluginLoader.PostProcessBlock(block)
}
