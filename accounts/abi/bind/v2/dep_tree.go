package bind

import (
	"encoding/hex"
	"fmt"
	"maps"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// DeploymentParams represents parameters needed to deploy a
// set of contracts.  It takes an optional override
// list to specify contracts/libraries that have already been deployed on-chain.
type DeploymentParams struct {
	Contracts []*MetaData

	// Map of solidity library pattern to constructor input.
	Inputs map[string][]byte

	// Overrides is an optional map of pattern to deployment address.
	// Contracts/libraries that refer to dependencies in the override
	// set are linked to the provided address (an already-deployed contract).
	Overrides map[string]common.Address
}

// validate determines whether the contracts specified in the DeploymentParams instance have provided deployer bytecode.
func (d *DeploymentParams) validate() error {
	for _, meta := range d.Contracts {
		if meta.Bin == "" {
			return fmt.Errorf("cannot deploy contract %s: deployer code missing from metadata", meta.ID)
		}
	}
	return nil
}

// DeploymentResult encapsulates information about the result of the deployment
// of a set of contracts: the pending deployment transactions, and the addresses
// where the contracts will be deployed at.
type DeploymentResult struct {
	// map of contract MetaData Id to deploy transaction
	Txs map[string]*types.Transaction

	// map of contract MetaData Id to deployed contract address
	Addrs map[string]common.Address
}

// DeployFn deploys a contract given a deployer and optional input.  It returns
// the address of the deployed contract and the deployment transaction, or an error if the deployment failed.
type DeployFn func(input, deployer []byte) (common.Address, *types.Transaction, error)

// depTreeDeployer is responsible for taking a dependency, deploying-and-linking its components in the proper
// order.  A depTreeDeployer cannot be used after calling LinkAndDeploy other than to retrieve the deployment result.
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

// linkAndDeploy recursively deploys a contract and its dependencies:  starting by linking/deploying its dependencies.
// The deployment result (deploy addresses/txs or an error) is stored in the depTreeDeployer object.
func (d *depTreeDeployer) linkAndDeploy(metadata *MetaData) (common.Address, error) {
	// Don't deploy already deployed contracts
	if addr, ok := d.deployedAddrs[metadata.ID]; ok {
		return addr, nil
	}
	// if this contract/library depends on other libraries deploy them (and their dependencies) first
	deployerCode := metadata.Bin
	for _, dep := range metadata.Deps {
		addr, err := d.linkAndDeploy(dep)
		if err != nil {
			return common.Address{}, err
		}
		// link their deployed addresses into the bytecode to produce
		deployerCode = strings.ReplaceAll(deployerCode, "__$"+dep.ID+"$__", strings.ToLower(addr.String()[2:]))
	}
	// Finally, deploy the contract.
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

// result returns a result for this deployment, or an error if it failed.
func (d *depTreeDeployer) result() *DeploymentResult {
	// remove the override addresses from the resulting deployedAddrs
	for pattern := range d.deployedAddrs {
		if _, ok := d.deployerTxs[pattern]; !ok {
			delete(d.deployedAddrs, pattern)
		}
	}
	return &DeploymentResult{
		Txs:   d.deployerTxs,
		Addrs: d.deployedAddrs,
	}
}

// LinkAndDeploy deploys a specified set of contracts and their dependent
// libraries.  If an error occurs, only contracts which were successfully
// deployed are returned in the result.
//
// In the case where multiple contracts share a common dependency:  the shared dependency will only be deployed once.
func LinkAndDeploy(deployParams *DeploymentParams, deploy DeployFn) (res *DeploymentResult, err error) {
	if err := deployParams.validate(); err != nil {
		return nil, err
	}
	deployer := newDepTreeDeployer(deployParams, deploy)
	for _, contract := range deployParams.Contracts {
		if _, err := deployer.linkAndDeploy(contract); err != nil {
			return deployer.result(), err
		}
	}
	return deployer.result(), nil
}
