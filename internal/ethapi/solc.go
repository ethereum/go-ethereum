// Copyright 2016 The go-ethereum Authors
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

package ethapi

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/rpc"
)

func makeCompilerAPIs(solcPath string) []rpc.API {
	c := &compilerAPI{solc: solcPath}
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   (*PublicCompilerAPI)(c),
			Public:    true,
		},
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   (*CompilerAdminAPI)(c),
			Public:    true,
		},
	}
}

type compilerAPI struct {
	// This lock guards the solc path set through the API.
	// It also ensures that only one solc process is used at
	// any time.
	mu   sync.Mutex
	solc string
}

type CompilerAdminAPI compilerAPI

// SetSolc sets the Solidity compiler path to be used by the node.
func (api *CompilerAdminAPI) SetSolc(path string) (string, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	info, err := compiler.SolidityVersion(path)
	if err != nil {
		return "", err
	}
	api.solc = path
	return info.FullVersion, nil
}

type PublicCompilerAPI compilerAPI

// CompileSolidity compiles the given solidity source.
func (api *PublicCompilerAPI) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	return compiler.CompileSolidityString(api.solc, source)
}

func (api *PublicCompilerAPI) GetCompilers() ([]string, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	if _, err := compiler.SolidityVersion(api.solc); err == nil {
		return []string{"Solidity"}, nil
	}
	return []string{}, nil
}
