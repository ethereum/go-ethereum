package gethclient

import (
	"context"
	"math/big"
	"runtime"
	"runtime/debug"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client is a wrapper around the rpc.Client that
// implements geth specific functionalities.
// Please use ethclient.Client instead if you want to
// access the standardized Etheeum RPC functionality.
type Client struct {
	c *rpc.Client
}

// New creates a client that uses the given RPC client.
func New(c *rpc.Client) *Client {
	return &Client{c}
}

// CreateAccessList tries to create an access list for a specific transaction based on the
// current pending state of the blockchain.
func (ec *Client) CreateAccessList(ctx context.Context, msg ethereum.CallMsg) (*types.AccessList, uint64, string, error) {
	type accessListResult struct {
		Accesslist *types.AccessList `json:"accessList"`
		Error      string            `json:"error,omitempty"`
		GasUsed    hexutil.Uint64    `json:"gasUsed"`
	}
	var result accessListResult
	if err := ec.c.CallContext(ctx, &result, "eth_createAccessList", toCallArg(msg)); err != nil {
		return nil, 0, "", err
	}
	return result.Accesslist, uint64(result.GasUsed), result.Error, nil
}

// GetProof returns the account and storage values of the specified account including the Merkle-proof.
// The block number can be nil, in which case the value is taken from the latest known block.
func (ec *Client) GetProof(ctx context.Context, account common.Address, keys []string, blockNumber *big.Int) (*ethapi.AccountResult, error) {
	var result ethapi.AccountResult
	err := ec.c.CallContext(ctx, &result, "eth_getProof", account, keys, toBlockNumArg(blockNumber))
	return &result, err
}

// OverrideAccount specifies the state of an account to be overridden.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.Big                `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

// CallContract executes a message call transaction, which is directly executed in the VM
// of the node, but never mined into the blockchain.
//
// blockNumber selects the block height at which the call runs. It can be nil, in which
// case the code is taken from the latest known block. Note that state from very old
// blocks might not be available.
//
// overrides specifies a map of contract states that should be overwritten before executing
// the message call.
// Please use ethclient.CallContract instead if you don't need the override functionality.
func (ec *Client) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int, overrides *map[common.Address]OverrideAccount) ([]byte, error) {
	var hex hexutil.Bytes
	err := ec.c.CallContext(
		ctx, &hex, "eth_call", toCallArg(msg),
		toBlockNumArg(blockNumber), overrides,
	)
	return hex, err
}

func (ec *Client) GCStats(ctx context.Context) (*debug.GCStats, error) {
	var result debug.GCStats
	err := ec.c.CallContext(ctx, &result, "debug_gcStats")
	return &result, err
}

func (ec *Client) MemStats(ctx context.Context) (*runtime.MemStats, error) {
	var result runtime.MemStats
	err := ec.c.CallContext(ctx, &result, "debug_memStats")
	return &result, err
}

// SetHead sets the current head of the local chain by block number.
// Note, this is a destructive action and may severely damage your chain.
// Use with extreme caution.
func (ec *Client) SetHead(ctx context.Context, number *big.Int) error {
	return ec.c.CallContext(ctx, nil, "debug_setHead", toBlockNumArg(number))
}

func (ec *Client) GetNodeInfo(ctx context.Context) (*p2p.NodeInfo, error) {
	var result p2p.NodeInfo
	err := ec.c.CallContext(ctx, &result, "admin_nodeInfo")
	return &result, err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
