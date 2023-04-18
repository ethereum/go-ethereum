package zkproof

import (
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

type MPTWitnessType int

const (
	MPTWitnessNothing MPTWitnessType = iota
	MPTWitnessNatural
	MPTWitnessRWTbl
)

// SMTPathNode represent a node in the SMT Path, all hash is saved by the present of
// zktype.Hash
type SMTPathNode struct {
	Value   hexutil.Bytes `json:"value"`
	Sibling hexutil.Bytes `json:"sibling"`
}

// SMTPath is the whole path of SMT
type SMTPath struct {
	KeyPathPart *hexutil.Big  `json:"pathPart"` //the path part in key
	Root        hexutil.Bytes `json:"root"`
	Path        []SMTPathNode `json:"path,omitempty"` //path start from top
	Leaf        *SMTPathNode  `json:"leaf,omitempty"` //would be omitted for empty leaf, the sibling indicate key
}

// StateAccount is the represent of StateAccount in L2 circuit
// Notice in L2 we have different hash scheme against StateAccount.MarshalByte
type StateAccount struct {
	Nonce            int           `json:"nonce"`
	Balance          *hexutil.Big  `json:"balance"` //just the common hex expression of integer (big-endian)
	KeccakCodeHash   hexutil.Bytes `json:"keccakCodeHash,omitempty"`
	PoseidonCodeHash hexutil.Bytes `json:"poseidonCodeHash,omitempty"`
	CodeSize         uint64        `json:"codeSize,omitempty"`
}

// StateStorage is the represent of a stored key-value pair for specified account
type StateStorage struct {
	Key   hexutil.Bytes `json:"key"` //notice this is the preimage of storage key
	Value hexutil.Bytes `json:"value"`
}

// StorageTrace record the updating on state trie and (if changed) account trie
// represent by the [before, after] updating of SMTPath amont tries and Account
type StorageTrace struct {
	// which log the trace is responded for, -1 indicate not caused
	// by opcode (like gasRefund, coinbase, setNonce, etc)
	Address         hexutil.Bytes    `json:"address"`
	AccountKey      hexutil.Bytes    `json:"accountKey"`
	AccountPath     [2]*SMTPath      `json:"accountPath"`
	AccountUpdate   [2]*StateAccount `json:"accountUpdate"`
	StateKey        hexutil.Bytes    `json:"stateKey,omitempty"`
	CommonStateRoot hexutil.Bytes    `json:"commonStateRoot,omitempty"` //CommonStateRoot is used if there is no update on state storage
	StatePath       [2]*SMTPath      `json:"statePath,omitempty"`
	StateUpdate     [2]*StateStorage `json:"stateUpdate,omitempty"`
}
