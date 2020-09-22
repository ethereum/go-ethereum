package accounts

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// gnosisSafe represents the functionality of
// https://github.com/gnosis/safe-contracts/blob/v1.1.1/contracts/GnosisSafe.sol
type gnosisSafe struct {
	addr common.Address
}

func (safe *gnosisSafe) domainSeparator() []byte {
	// crypto.Keccak256Hash([]byte("EIP712Domain(address verifyingContract)"))
	var domainSeparatorType = []byte("\x03Z\xff\x83\xd8i7\xd3[2\xe0O\r\xdco\xf4i)\x0e\xef/\x1bi-\x8a\x81\\\x89@MGI")
	var ds []byte
	ds = append(ds, domainSeparatorType...)
	ds = append(ds, common.LeftPadBytes(safe.addr[:], 32)...)
	return crypto.Keccak256(ds)
}

// GnosisSafeSigningHash returns a tuple, txHash and signingHash. The latter is tied to the
// actual gnosis-safe used, whereas the former is transaction-specific.
// In order to confirm/sign a safe-tx, the latter needs to be signed by the
// keyholder
//
// The Gnosis safe uses a scheme where keyholders submit their signatures in a batch, and
// each keyholder thus signs the 'GnosisSafeTx' individually. The collected signatures are
// later submitted using only one single transaction.
//
// This method calculates the hash to sign based on a 'GnosisSafeTx' (the tx that is the end-goal
// of the entire signing round).
func GnosisSafeSigningHash(tx *GnosisSafeTx) (signingHash common.Hash, preimage []byte, err error) {
	// Calc the particular "safetx"-hash for the transaction
	safeTxHash, err := tx.hash()
	if err != nil {
		return common.Hash{}, nil, err
	}
	safe := gnosisSafe{tx.Safe.Address()}

	// Put together the final preimage
	msg := []byte{0x19, 0x01}
	// Calc the safe-specific domain separator (depends on address)
	msg = append(msg, safe.domainSeparator()...)
	msg = append(msg, safeTxHash[:]...)
	// Final hash for signing
	signingHash = crypto.Keccak256Hash(msg)

	if tx.InputExpHash != (common.Hash{}) && tx.InputExpHash != signingHash {
		return common.Hash{}, nil,
			fmt.Errorf("expected hash differs from calculated, input-expectation was %x, got %x",
				tx.InputExpHash, signingHash)
	}
	return signingHash, msg, nil
}

type GnosisSafeTx struct {
	Safe           common.MixedcaseAddress `json:"safe"`
	To             common.MixedcaseAddress `json:"to"`
	Value          math.Decimal256         `json:"value"`
	GasPrice       math.Decimal256         `json:"gasPrice"`
	Data           *hexutil.Bytes          `json:"data"`
	Operation      uint8                   `json:"operation"`
	GasToken       common.Address          `json:"gasToken"`
	RefundReceiver common.Address          `json:"refundReceiver"`
	BaseGas        big.Int                 `json:"baseGas"`
	SafeTxGas      big.Int                 `json:"safeTxGas"`
	Nonce          big.Int                 `json:"nonce"`
	InputExpHash   common.Hash             `json:"safeTxHash"`
}

/**
hash implements the following solidity construct:

  function encodeTransactionData(
       address to,
       uint256 value,
       bytes memory data,
       Enum.Operation operation,
       uint256 safeTxGas,
       uint256 baseGas,
       uint256 gasPrice,
       address gasToken,
       address refundReceiver,
       uint256 nonce)

bytes32 safeTxHash = keccak256(
	abi.encode(SAFE_TX_TYPEHASH, to, value, keccak256(data), operation, safeTxGas, baseGas, gasPrice, gasToken, refundReceiver, _nonce)
);
*/
func (tx *GnosisSafeTx) hash() (common.Hash, error) {
	if tx.Operation != 0 {
		return common.Hash{}, fmt.Errorf("Signing type %d not implemented", tx.Operation)
	}
	// crypto.Keccak256Hash([]byte("SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"))
	var safeTxHashType = common.Hash{0xbb, 0x83, 0x10, 0xd4, 0x86, 0x36, 0x8d, 0xb6, 0xbd, 0x6f, 0x84, 0x94, 0x02, 0xfd, 0xd7, 0x3a, 0xd5, 0x3d, 0x31, 0x6b, 0x5a, 0x4b, 0x26, 0x44, 0xad, 0x6e, 0xfe, 0x0f, 0x94, 0x12, 0x86, 0xd8}
	// We use the ABI below to aid packing the preimage
	var txPackABI = `[{
		"name": "encodeTransactionData",
		"inputs": [
			{"name": "hashType","type": "bytes32"},
			{"name": "to","type": "address"},
			{"name": "value","type": "uint256"},
			{"name": "datahash","type": "bytes32"},
			{"name": "operation","type": "uint8"},
			{"name": "safeTxGas","type": "uint256"},
			{"name": "baseGas","type": "uint256"},
			{"name": "gasPrice","type": "uint256"},
			{"name": "gasToken","type": "address"},
			{"name": "refundReceiver","type": "address"},
			{"name": "_nonce","type": "uint256"}
		],
		"type": "function"
	}]`
	abispec, err := abi.JSON(strings.NewReader(txPackABI))
	if err != nil {
		return (common.Hash{}), err
	}
	val := big.Int(tx.Value)
	gasPrice := big.Int(tx.GasPrice)
	// Pack the fields
	var data []byte
	if tx.Data != nil {
		data = []byte(*tx.Data)
	}
	packed, err := abispec.Methods["encodeTransactionData"].Inputs.Pack(
		safeTxHashType, tx.To.Address(), &val, crypto.Keccak256Hash(data), tx.Operation,
		&tx.SafeTxGas, &tx.BaseGas, &gasPrice, tx.GasToken, tx.RefundReceiver, &tx.Nonce,
	)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(packed), nil
}
