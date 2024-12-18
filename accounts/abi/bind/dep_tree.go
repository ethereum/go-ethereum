package bind

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"maps"
	"strings"
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

// Accumulate merges `other` into `d`
func (d *DeploymentResult) Accumulate(other *DeploymentResult) {
	maps.Copy(d.Txs, other.Txs)
	maps.Copy(d.Addrs, other.Addrs)
}

// DeployFn deploys a contract given a deployer and optional input.  It returns
// the address and a pending transaction, or an error if the deployment failed.
type DeployFn func(input, deployer []byte) (common.Address, *types.Transaction, error)

// depTreeDeployer is responsible for taking a dependency, deploying-and-linking its components in the proper
// order.  A depTreeDeployer cannot be used after calling LinkAndDeploy other than to retrieve the deployment result.
type depTreeDeployer struct {
	deployedAddrs map[string]common.Address
	deployerTxs   map[string]*types.Transaction
	input         map[string][]byte // map of the root contract pattern to the constructor input (if there is any)
	deploy        DeployFn
}

// linkAndDeploy recursively deploys a contract and its dependencies:  starting by linking/deploying its dependencies.
// The deployment result (deploy addresses/txs or an error) is stored in the depTreeDeployer object.
func (d *depTreeDeployer) linkAndDeploy(metadata *MetaData) error {
	// don't deploy contracts specified as overrides.  don't deploy their dependencies.
	if _, ok := d.deployedAddrs[metadata.Pattern]; ok {
		return nil
	}

	// if this contract/library depends on other libraries deploy them (and their dependencies) first
	for _, dep := range metadata.Deps {
		if err := d.linkAndDeploy(dep); err != nil {
			return err
		}
	}
	// if we just deployed any prerequisite contracts, link their deployed addresses into the bytecode to produce
	// a deployer bytecode for this contract.
	deployerCode := metadata.Bin
	for _, dep := range metadata.Deps {
		linkAddr := d.deployedAddrs[dep.Pattern]
		deployerCode = strings.ReplaceAll(deployerCode, "__$"+dep.Pattern+"$__", strings.ToLower(linkAddr.String()[2:]))
	}

	// Finally, deploy the contract.
	addr, tx, err := d.deploy(d.input[metadata.Pattern], common.Hex2Bytes(deployerCode))
	if err != nil {
		return err
	}

	d.deployedAddrs[metadata.Pattern] = addr
	d.deployerTxs[metadata.Pattern] = tx
	return nil
}

// result returns a result for this deployment, or an error if it failed.
func (d *depTreeDeployer) result() *DeploymentResult {
	return &DeploymentResult{
		Txs:   d.deployerTxs,
		Addrs: d.deployedAddrs,
	}
}

func newDepTreeDeployer(deploy DeployFn) *depTreeDeployer {
	return &depTreeDeployer{
		deploy:        deploy,
		deployedAddrs: make(map[string]common.Address),
		deployerTxs:   make(map[string]*types.Transaction)}
}

// LinkAndDeploy deploys a specified set of contracts and their dependent
// libraries.  If an error occurs, only contracts which were successfully
// deployed are returned in the result.
func LinkAndDeploy(deployParams *DeploymentParams, deploy DeployFn) (res *DeploymentResult, err error) {
	accumRes := &DeploymentResult{
		Txs:   make(map[string]*types.Transaction),
		Addrs: make(map[string]common.Address),
	}
	deployer := newDepTreeDeployer(deploy)
	for _, contract := range deployParams.contracts {
		if deployParams.inputs != nil {
			deployer.input = map[string][]byte{contract.Pattern: deployParams.inputs[contract.Pattern]}
		}
		err := deployer.linkAndDeploy(contract)
		res := deployer.result()
		accumRes.Accumulate(res)
		if err != nil {
			return accumRes, err
		}
	}
	return accumRes, nil
}
