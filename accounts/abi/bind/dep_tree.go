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
	contracts []*MetaData
	// map of solidity library pattern to constructor input.
	inputs map[string][]byte
	// Overrides is an optional map of pattern to deployment address.
	// Contracts/libraries that refer to dependencies in the override
	// set are linked to the provided address (an already-deployed contract).
	overrides map[string]common.Address
}

// NewDeploymentParams instantiates a DeploymentParams instance.
func NewDeploymentParams(contracts []*MetaData, inputs map[string][]byte, overrides map[string]common.Address) *DeploymentParams {
	return &DeploymentParams{
		contracts,
		inputs,
		overrides,
	}
}

// DeploymentResult encapsulates information about the result of the deployment
// of a set of contracts: the pending deployment transactions, and the addresses
// where the contracts will be deployed at.
type DeploymentResult struct {
	// map of contract library pattern -> deploy transaction
	Txs map[string]*types.Transaction
	// map of contract library pattern -> deployed address
	Addrs map[string]common.Address
}

// DeployFn deploys a contract given a deployer and optional input.  It returns
// the address and a pending transaction, or an error if the deployment failed.
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
	return &depTreeDeployer{
		deployFn:      deployFn,
		deployedAddrs: maps.Clone(deployParams.overrides),
		deployerTxs:   make(map[string]*types.Transaction),
		inputs:        maps.Clone(deployParams.inputs),
	}
}

// linkAndDeploy recursively deploys a contract and its dependencies:  starting by linking/deploying its dependencies.
// The deployment result (deploy addresses/txs or an error) is stored in the depTreeDeployer object.
func (d *depTreeDeployer) linkAndDeploy(metadata *MetaData) (common.Address, error) {
	// Don't deploy already deployed contracts
	if addr, ok := d.deployedAddrs[metadata.Pattern]; ok {
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
		deployerCode = strings.ReplaceAll(deployerCode, "__$"+dep.Pattern+"$__", strings.ToLower(addr.String()[2:]))
	}
	// Finally, deploy the contract.
	code, err := hex.DecodeString(deployerCode[2:])
	if err != nil {
		panic(fmt.Sprintf("error decoding contract deployer hex %s:\n%v", deployerCode[2:], err))
	}
	addr, tx, err := d.deployFn(d.inputs[metadata.Pattern], code)
	if err != nil {
		return common.Address{}, err
	}
	d.deployedAddrs[metadata.Pattern] = addr
	d.deployerTxs[metadata.Pattern] = tx
	return addr, nil
}

// result returns a result for this deployment, or an error if it failed.
func (d *depTreeDeployer) result() *DeploymentResult {
	return &DeploymentResult{
		Txs:   d.deployerTxs,
		Addrs: d.deployedAddrs,
	}
}

// LinkAndDeploy deploys a specified set of contracts and their dependent
// libraries.  If an error occurs, only contracts which were successfully
// deployed are returned in the result.
func LinkAndDeploy(deployParams *DeploymentParams, deploy DeployFn) (res *DeploymentResult, err error) {
	deployer := newDepTreeDeployer(deployParams, deploy)
	for _, contract := range deployParams.contracts {
		if _, err := deployer.linkAndDeploy(contract); err != nil {
			return deployer.result(), err
		}
	}
	return deployer.result(), nil
}
