package bind

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"regexp"
	"strings"
)

// DeploymentParams represents parameters needed to deploy a
// set of contracts.  It takes an optional override
// list to specify contracts/libraries that have already been deployed on-chain.
type DeploymentParams struct {
	Contracts []*MetaData
	// map of solidity library pattern to constructor input.
	Inputs map[string][]byte
	// Overrides is an optional map of pattern to deployment address.
	// Contracts/libraries that refer to dependencies in the override
	// set are linked to the provided address (an already-deployed contract).
	Overrides map[string]common.Address
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

// Accumulate merges two DeploymentResult objects together.
func (d *DeploymentResult) Accumulate(other *DeploymentResult) {
	for pattern, tx := range other.Txs {
		d.Txs[pattern] = tx
	}
	for pattern, addr := range other.Addrs {
		d.Addrs[pattern] = addr
	}
}

// depTreeBuilder turns a set of unlinked contracts libraries into a set of one
// or more dependency trees.
type depTreeBuilder struct {
	overrides map[string]common.Address
	// map of pattern to unlinked contract bytecode (for libraries or contracts)
	contracts map[string]string
	// map of pattern to subtree represented by contract
	subtrees map[string]*depTreeNode
	// map of nodes that aren't referenced by other dependencies (these can be libraries too if user is doing lib-only deployment)
	roots map[string]struct{}
}

// depTreeNode represents a node (contract) in a dependency tree.  it contains its unlinked code, and references to any
// library contracts that it requires.  If it is specified as an override, it contains the address where it has already
// been deployed at.
type depTreeNode struct {
	pattern      string
	unlinkedCode string
	children     []*depTreeNode
	overrideAddr *common.Address
}

// Flatten returns the subtree into a map of pattern -> unlinked contract bytecode.
func (n *depTreeNode) Flatten() (res map[string]string) {
	res = map[string]string{n.pattern: n.unlinkedCode}
	for _, child := range n.children {
		subtree := child.Flatten()

		for k, v := range subtree {
			res[k] = v
		}
	}
	return res
}

// buildDepTrees is the internal version of BuildDepTrees that recursively calls itself.
func (d *depTreeBuilder) buildDepTrees(pattern, contract string) {
	// if the node is in the subtree set already, it has already been fully recursed/built so we can bail out.
	if _, ok := d.subtrees[pattern]; ok {
		return
	}
	node := &depTreeNode{
		pattern:      pattern,
		unlinkedCode: contract,
	}
	if addr, ok := d.overrides[pattern]; ok {
		node.overrideAddr = &addr
	}
	// iterate each referenced library in the unlinked code, recurse and build its subtree.
	reMatchSpecificPattern, err := regexp.Compile(`__\$([a-f0-9]+)\$__`)
	if err != nil {
		panic(err)
	}
	for _, match := range reMatchSpecificPattern.FindAllStringSubmatch(contract, -1) {
		depPattern := match[1]
		d.buildDepTrees(depPattern, d.contracts[depPattern])
		node.children = append(node.children, d.subtrees[depPattern])

		// this library can't be a root dependency if it is referenced by other contracts.
		delete(d.roots, depPattern)
	}
	d.subtrees[pattern] = node
}

// BuildDepTrees will compute a set of dependency trees from a set of unlinked contracts.  The root of each tree
// corresponds to a contract/library that is not referenced as a dependency anywhere else.  Children of each node are
// its library dependencies.  It returns nodes that are roots of a dependency tree and nodes that aren't.
func (d *depTreeBuilder) BuildDepTrees() (roots []*depTreeNode, nonRoots []*depTreeNode) {
	// before the trees of dependencies are known, consider that any provided contract could be a root.
	for pattern, _ := range d.contracts {
		d.roots[pattern] = struct{}{}
	}
	for pattern, contract := range d.contracts {
		d.buildDepTrees(pattern, contract)
	}
	for pattern, _ := range d.contracts {
		if _, ok := d.roots[pattern]; ok {
			roots = append(roots, d.subtrees[pattern])
		} else {
			nonRoots = append(nonRoots, d.subtrees[pattern])
		}
	}
	return roots, nonRoots
}

func newDepTreeBuilder(overrides map[string]common.Address, contracts map[string]string) *depTreeBuilder {
	return &depTreeBuilder{
		overrides: overrides,
		contracts: contracts,
		subtrees:  make(map[string]*depTreeNode),
		roots:     make(map[string]struct{}),
	}
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
func (d *depTreeDeployer) linkAndDeploy(node *depTreeNode) error {
	// don't deploy contracts specified as overrides.  don't deploy their dependencies.
	if node.overrideAddr != nil {
		return nil
	}

	// if this contract/library depends on other libraries deploy them (and their dependencies) first
	for _, childNode := range node.children {
		if err := d.linkAndDeploy(childNode); err != nil {
			return err
		}
	}
	// if we just deployed any prerequisite contracts, link their deployed addresses into the bytecode to produce
	// a deployer bytecode for this contract.
	deployerCode := node.unlinkedCode
	for _, child := range node.children {
		var linkAddr common.Address
		if child.overrideAddr != nil {
			linkAddr = *child.overrideAddr
		} else {
			linkAddr = d.deployedAddrs[child.pattern]
		}
		deployerCode = strings.ReplaceAll(deployerCode, "__$"+child.pattern+"$__", strings.ToLower(linkAddr.String()[2:]))
	}

	// Finally, deploy the contract.
	addr, tx, err := d.deploy(d.input[node.pattern], common.Hex2Bytes(deployerCode))
	if err != nil {
		return err
	}

	d.deployedAddrs[node.pattern] = addr
	d.deployerTxs[node.pattern] = tx
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
func LinkAndDeploy(deployParams DeploymentParams, deploy DeployFn) (res *DeploymentResult, err error) {
	unlinkedContracts := make(map[string]string)
	accumRes := &DeploymentResult{
		Txs:   make(map[string]*types.Transaction),
		Addrs: make(map[string]common.Address),
	}
	for _, meta := range deployParams.Contracts {
		unlinkedContracts[meta.Pattern] = meta.Bin[2:]
	}
	treeBuilder := newDepTreeBuilder(deployParams.Overrides, unlinkedContracts)
	deps, _ := treeBuilder.BuildDepTrees()

	for _, tr := range deps {
		deployer := newDepTreeDeployer(deploy)
		if deployParams.Inputs != nil {
			deployer.input = map[string][]byte{tr.pattern: deployParams.Inputs[tr.pattern]}
		}
		err := deployer.linkAndDeploy(tr)
		res := deployer.result()
		accumRes.Accumulate(res)
		if err != nil {
			return accumRes, err
		}
	}
	return accumRes, nil
}
