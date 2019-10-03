// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

var (
	codefile string
	input    string
	expected string
)

var flagsParsed = func() bool {
	flag.StringVar(&codefile, "codefile", "", "EVM code to run")
	flag.StringVar(&input, "input", "", "input calldata")
	flag.StringVar(&expected, "expected", "", "expected return data")
	flag.Parse()
	return true
}()

func BenchmarkEvmCode(b *testing.B) {

	state, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
	address := common.HexToAddress("0x0a")

	var (
		codebytes []byte
		err       error
		ret       []byte
		codehex   []byte
		gasLeft   uint64
	)

	fmt.Println("codefile:", codefile)

	if len(codefile) > 0 {
		codehex, err = ioutil.ReadFile(codefile)
		if err != nil {
			panic(err)
		}
	} else {
		panic("Need to pass --codefile arg!")
	}

	codehexstr := string(codehex)

	fmt.Println("code hex length:", len(codehexstr))

	codebytes = common.Hex2Bytes(codehexstr)

	state.SetCode(address, codebytes)
	evmChainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
	}

	startGas := uint64(100000000) // 100 million
	inputBytes := common.Hex2Bytes(input)
	config := &Config{ChainConfig: evmChainConfig, State: state, EVMConfig: vm.Config{}, GasLimit: startGas}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ret, gasLeft, err = Call(address, inputBytes, config)
	}
	b.StopTimer()
	//Check if it is correct
	if err != nil {
		b.Error(err)
		return
	}

	fmt.Println("got return bytes:", common.Bytes2Hex(ret))
	if common.Bytes2Hex(ret) != expected {
		b.Error(fmt.Sprintf("Expected %v, got %v", expected, common.Bytes2Hex(ret)))
		return
	}

	gasUsed := startGas - gasLeft
	fmt.Println("gasUsed:", gasUsed)

}
