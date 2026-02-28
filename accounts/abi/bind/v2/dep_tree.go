// Copyright 2025 The go-ethereum Authors
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

package bind

import (
	"encoding/hex"
	"fmt"
	"maps"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// DeploymentParams contains parameters needed to deploy one or more contracts via LinkAndDeploy
type DeploymentParams struct {
	// list of all contracts targeted for the deployment
	Contracts []*MetaData

	// optional map of ABI-encoded constructor inputs keyed by the MetaData.ID.
	Inputs map[string][]byte

	// optional map of override addresses for specifying already-deployed
	// contracts.  It is keyed by the MetaData.ID.
	Overrides map[string]common.Address
}

// validate determines whether the contracts specified in the DeploymentParams
// instance have embedded deployer code in their provided MetaData instances.
func (d *DeploymentParams) validate() error {
	for _, meta := range d.Contracts {
		if meta.Bin == "" {
			return fmt.Errorf("cannot deploy contract %s: deployer code missing from metadata", meta.ID)
		}
	}
	return nil
}

// DeploymentResult contains information about the result of a pending
// deployment made by LinkAndDeploy.
type DeploymentResult struct {
	// Map of contract MetaData.ID to pending deployment transaction
	Txs map[string]*types.Transaction

	// Map of contract MetaData.ID to the address where it will be deployed
	Addresses map[string]common.Address
}

// DeployFn deploys a contract given a deployer and optional input.  It returns
// the address and a pending transaction, or an error if the deployment failed.
type DeployFn func(input, deployer []byte) (common.Address, *types.Transaction, error)

// depTreeDeployer is responsible for taking a dependency, deploying-and-linking
// its components in the proper order. A depTreeDeployer cannot be used after
// calling LinkAndDeploy other than to retrieve the deployment result.
type depTreeDeployer struct {
	deployedAddrs map[string]common.Address
	deployerTxs   map[string]*types.Transaction
	inputs        map[string][]byte // map of the root contract pattern to the constructor input (if there is any)
	deployFn      DeployFn
}

func newDepTreeDeployer(deployParams *DeploymentParams, deployFn DeployFn) *depTreeDeployer {
	deployedAddrs := maps.Clone(deployParams.Overrides)
	if deployedAddrs == nil {
		deployedAddrs = make(map[string]common.Address)
	}
	inputs := deployParams.Inputs
	if inputs == nil {
		inputs = make(map[string][]byte)
	}
	return &depTreeDeployer{
		deployFn:      deployFn,
		deployedAddrs: deployedAddrs,
		deployerTxs:   make(map[string]*types.Transaction),
		inputs:        inputs,
	}
}

// linkAndDeploy deploys a contract and it's dependencies.  Because libraries
// can in-turn have their own library dependencies, linkAndDeploy performs
// deployment recursively (deepest-dependency first).  The address of the
// pending contract deployment for the top-level contract is returned.
func (d *depTreeDeployer) linkAndDeploy(metadata *MetaData) (common.Address, error) {
	// Don't re-deploy aliased or previously-deployed contracts
	if addr, ok := d.deployedAddrs[metadata.ID]; ok {
		return addr, nil
	}
	// If this contract/library depends on other libraries deploy them
	// (and their dependencies) first
	deployerCode := metadata.Bin
	for _, dep := range metadata.Deps {
		addr, err := d.linkAndDeploy(dep)
		if err != nil {
			return common.Address{}, err
		}
		// Link their deployed addresses into the bytecode to produce
		deployerCode = strings.ReplaceAll(deployerCode, "__$"+dep.ID+"$__", strings.ToLower(addr.String()[2:]))
	}
	// Finally, deploy the top-level contract.
	code, err := hex.DecodeString(deployerCode[2:])
	if err != nil {
		panic(fmt.Sprintf("error decoding contract deployer hex %s:\n%v", deployerCode[2:], err))
	}
	addr, tx, err := d.deployFn(d.inputs[metadata.ID], code)
	if err != nil {
		return common.Address{}, err
	}
	d.deployedAddrs[metadata.ID] = addr
	d.deployerTxs[metadata.ID] = tx
	return addr, nil
}

// result returns a DeploymentResult instance referencing contracts deployed
// and not including any overrides specified for this deployment.
func (d *depTreeDeployer) result() *DeploymentResult {
	// filter the override addresses from the deployed address set.
	for pattern := range d.deployedAddrs {
		if _, ok := d.deployerTxs[pattern]; !ok {
			delete(d.deployedAddrs, pattern)
		}
	}
	return &DeploymentResult{
		Txs:       d.deployerTxs,
		Addresses: d.deployedAddrs,
	}
}

// LinkAndDeploy performs the contract deployment specified by params using the
// provided DeployFn to create, sign and submit transactions.
//
// Contracts can depend on libraries, which in-turn can have their own library
// dependencies.  Therefore, LinkAndDeploy performs the deployment recursively,
// starting with libraries (and contracts) that don't have dependencies, and
// progressing through the contracts that depend upon them.
//
// If an error is encountered, the returned DeploymentResult only contains
// entries for the contracts whose deployment submission succeeded.
//
// LinkAndDeploy performs creation and submission of creation transactions,
// but does not ensure that the contracts are included in the chain.
func LinkAndDeploy(params *DeploymentParams, deploy DeployFn) (*DeploymentResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}
	deployer := newDepTreeDeployer(params, deploy)
	for _, contract := range params.Contracts {
		if _, err := deployer.linkAndDeploy(contract); err != nil {
			return deployer.result(), err
		}
	}
	return deployer.result(), nil
}
