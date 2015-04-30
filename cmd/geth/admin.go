package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/xeth"
	"github.com/robertkrimen/otto"
)

/*
node admin bindings
*/

func (js *jsre) adminBindings() {
	js.re.Set("admin", struct{}{})
	t, _ := js.re.Get("admin")
	admin := t.Object()
	admin.Set("addPeer", js.addPeer)
	admin.Set("startRPC", js.startRPC)
	admin.Set("stopRPC", js.stopRPC)
	admin.Set("nodeInfo", js.nodeInfo)
	admin.Set("peers", js.peers)
	admin.Set("newAccount", js.newAccount)
	admin.Set("unlock", js.unlock)
	admin.Set("import", js.importChain)
	admin.Set("export", js.exportChain)
	admin.Set("verbosity", js.verbosity)
	admin.Set("progress", js.downloadProgress)

	admin.Set("miner", struct{}{})
	t, _ = admin.Get("miner")
	miner := t.Object()
	miner.Set("start", js.startMining)
	miner.Set("stop", js.stopMining)
	miner.Set("hashrate", js.hashrate)
	miner.Set("setExtra", js.setExtra)

	admin.Set("debug", struct{}{})
	t, _ = admin.Get("debug")
	debug := t.Object()
	debug.Set("backtrace", js.backtrace)
	debug.Set("printBlock", js.printBlock)
	debug.Set("dumpBlock", js.dumpBlock)
	debug.Set("getBlockRlp", js.getBlockRlp)
	debug.Set("setHead", js.setHead)
	debug.Set("processBlock", js.debugBlock)
}

func (js *jsre) getBlock(call otto.FunctionCall) (*types.Block, error) {
	var block *types.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = js.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = js.ethereum.ChainManager().GetBlock(common.HexToHash(hash))
		} else {
			return nil, errors.New("invalid argument for dump. Either hex string or number")
		}
		return block, nil
	}

	return nil, errors.New("requires block number or block hash as argument")
}

func (js *jsre) debugBlock(call otto.FunctionCall) otto.Value {
	block, err := js.getBlock(call)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	old := vm.Debug
	vm.Debug = true
	_, err = js.ethereum.BlockProcessor().RetryProcess(block)
	if err != nil {
		glog.Infoln(err)
	}
	vm.Debug = old

	return otto.UndefinedValue()
}

func (js *jsre) setHead(call otto.FunctionCall) otto.Value {
	block, err := js.getBlock(call)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	js.ethereum.ChainManager().SetHead(block)
	return otto.UndefinedValue()
}

func (js *jsre) downloadProgress(call otto.FunctionCall) otto.Value {
	current, max := js.ethereum.Downloader().Stats()

	return js.re.ToVal(fmt.Sprintf("%d/%d", current, max))
}

func (js *jsre) getBlockRlp(call otto.FunctionCall) otto.Value {
	block, err := js.getBlock(call)
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	encoded, _ := rlp.EncodeToBytes(block)
	return js.re.ToVal(fmt.Sprintf("%x", encoded))
}

func (js *jsre) setExtra(call otto.FunctionCall) otto.Value {
	extra, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	if len(extra) > 1024 {
		fmt.Println("error: cannot exceed 1024 bytes")
		return otto.UndefinedValue()
	}

	js.ethereum.Miner().SetExtra([]byte(extra))
	return otto.UndefinedValue()
}

func (js *jsre) hashrate(otto.FunctionCall) otto.Value {
	return js.re.ToVal(js.ethereum.Miner().HashRate())
}

func (js *jsre) backtrace(call otto.FunctionCall) otto.Value {
	tracestr, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}
	glog.GetTraceLocation().Set(tracestr)

	return otto.UndefinedValue()
}

func (js *jsre) verbosity(call otto.FunctionCall) otto.Value {
	v, err := call.Argument(0).ToInteger()
	if err != nil {
		fmt.Println(err)
		return otto.UndefinedValue()
	}

	glog.SetV(int(v))
	return otto.UndefinedValue()
}

func (js *jsre) startMining(call otto.FunctionCall) otto.Value {
	_, err := call.Argument(0).ToInteger()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	// threads now ignored
	err = js.ethereum.StartMining()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) stopMining(call otto.FunctionCall) otto.Value {
	js.ethereum.StopMining()
	return otto.TrueValue()
}

func (js *jsre) startRPC(call otto.FunctionCall) otto.Value {
	addr, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	port, err := call.Argument(1).ToInteger()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}

	corsDomain := js.corsDomain
	if len(call.ArgumentList) > 2 {
		corsDomain, err = call.Argument(2).ToString()
		if err != nil {
			fmt.Println(err)
			return otto.FalseValue()
		}
	}

	config := rpc.RpcConfig{
		ListenAddress: addr,
		ListenPort:    uint(port),
		CorsDomain:    corsDomain,
	}

	xeth := xeth.New(js.ethereum, nil)
	err = rpc.Start(xeth, config)

	if err != nil {
		fmt.Printf(err.Error())
		return otto.FalseValue()
	}

	return otto.TrueValue()
}

func (js *jsre) stopRPC(call otto.FunctionCall) otto.Value {
	if rpc.Stop() == nil {
		return otto.TrueValue()
	}
	return otto.FalseValue()
}

func (js *jsre) addPeer(call otto.FunctionCall) otto.Value {
	nodeURL, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	err = js.ethereum.AddPeer(nodeURL)
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) unlock(call otto.FunctionCall) otto.Value {
	addr, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	seconds, err := call.Argument(2).ToInteger()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	arg := call.Argument(1)
	var passphrase string
	if arg.IsUndefined() {
		fmt.Println("Please enter a passphrase now.")
		passphrase, err = readPassword("Passphrase: ", true)
		if err != nil {
			utils.Fatalf("%v", err)
		}
	} else {
		passphrase, err = arg.ToString()
		if err != nil {
			fmt.Println(err)
			return otto.FalseValue()
		}
	}
	am := js.ethereum.AccountManager()
	err = am.TimedUnlock(common.FromHex(addr), passphrase, time.Duration(seconds)*time.Second)
	if err != nil {
		fmt.Printf("Unlock account failed '%v'\n", err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) newAccount(call otto.FunctionCall) otto.Value {
	arg := call.Argument(0)
	var passphrase string
	if arg.IsUndefined() {
		fmt.Println("The new account will be encrypted with a passphrase.")
		fmt.Println("Please enter a passphrase now.")
		auth, err := readPassword("Passphrase: ", true)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		confirm, err := readPassword("Repeat Passphrase: ", false)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		if auth != confirm {
			utils.Fatalf("Passphrases did not match.")
		}
		passphrase = auth
	} else {
		var err error
		passphrase, err = arg.ToString()
		if err != nil {
			fmt.Println(err)
			return otto.FalseValue()
		}
	}
	acct, err := js.ethereum.AccountManager().NewAccount(passphrase)
	if err != nil {
		fmt.Printf("Could not create the account: %v", err)
		return otto.UndefinedValue()
	}
	return js.re.ToVal("0x" + common.Bytes2Hex(acct.Address))
}

func (js *jsre) nodeInfo(call otto.FunctionCall) otto.Value {
	return js.re.ToVal(js.ethereum.NodeInfo())
}

func (js *jsre) peers(call otto.FunctionCall) otto.Value {
	return js.re.ToVal(js.ethereum.PeersInfo())
}

func (js *jsre) importChain(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}
	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	if err := utils.ImportChain(js.ethereum.ChainManager(), fn); err != nil {
		fmt.Println("Import error: ", err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) exportChain(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) == 0 {
		fmt.Println("err: require file name")
		return otto.FalseValue()
	}

	fn, err := call.Argument(0).ToString()
	if err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	if err := utils.ExportChain(js.ethereum.ChainManager(), fn); err != nil {
		fmt.Println(err)
		return otto.FalseValue()
	}
	return otto.TrueValue()
}

func (js *jsre) printBlock(call otto.FunctionCall) otto.Value {
	var block *types.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = js.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = js.ethereum.ChainManager().GetBlock(common.HexToHash(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

	} else {
		block = js.ethereum.ChainManager().CurrentBlock()
	}
	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	fmt.Println(block)

	return otto.UndefinedValue()
}

func (js *jsre) dumpBlock(call otto.FunctionCall) otto.Value {
	var block *types.Block
	if len(call.ArgumentList) > 0 {
		if call.Argument(0).IsNumber() {
			num, _ := call.Argument(0).ToInteger()
			block = js.ethereum.ChainManager().GetBlockByNumber(uint64(num))
		} else if call.Argument(0).IsString() {
			hash, _ := call.Argument(0).ToString()
			block = js.ethereum.ChainManager().GetBlock(common.HexToHash(hash))
		} else {
			fmt.Println("invalid argument for dump. Either hex string or number")
		}

	} else {
		block = js.ethereum.ChainManager().CurrentBlock()
	}
	if block == nil {
		fmt.Println("block not found")
		return otto.UndefinedValue()
	}

	statedb := state.New(block.Root(), js.ethereum.StateDb())
	dump := statedb.RawDump()
	return js.re.ToVal(dump)

}
