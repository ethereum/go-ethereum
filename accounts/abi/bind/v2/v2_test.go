package v2

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"io"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"strings"
	"testing"
)

const deployer = "6080604052348015600e575f80fd5b506102098061001c5f395ff3fe608060405234801561000f575f80fd5b506004361061003f575f3560e01c80636da1cd55146100435780637b0cb83914610061578063bf54fad41461006b575b5f80fd5b61004b610087565b60405161005891906100e9565b60405180910390f35b61006961008f565b005b61008560048036038101906100809190610130565b6100c8565b005b5f8054905090565b607b7f72c79b1cb25b1b49ae522446226e1591b80634619cef7e71846da52b61b7061d6040516100be906101b5565b60405180910390a2565b805f8190555050565b5f819050919050565b6100e3816100d1565b82525050565b5f6020820190506100fc5f8301846100da565b92915050565b5f80fd5b61010f816100d1565b8114610119575f80fd5b50565b5f8135905061012a81610106565b92915050565b5f6020828403121561014557610144610102565b5b5f6101528482850161011c565b91505092915050565b5f82825260208201905092915050565b7f737472696e6700000000000000000000000000000000000000000000000000005f82015250565b5f61019f60068361015b565b91506101aa8261016b565b602082019050919050565b5f6020820190508181035f8301526101cc81610193565b905091905056fea2646970667358221220212a3a765a98254b596386fdfd10318f9a4bf19e8c9ca9ffa363f990c1798bf664736f6c634300081a0033"

const contractABIStr = `
[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "uint256",
        "name": "firstArg",
        "type": "uint256"
      },
      {
        "indexed": false,
        "internalType": "string",
        "name": "secondArg",
        "type": "string"
      }
    ],
    "name": "ExampleEvent",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "emitEvent",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "num",
        "type": "uint256"
      }
    ],
    "name": "mutateStorageVal",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "retrieveStorageVal",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]
`

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (abi.ABI, error) {
	dec := json.NewDecoder(reader)

	var instance abi.ABI
	if err := dec.Decode(&instance); err != nil {
		return abi.ABI{}, err
	}
	return instance, nil
}

func TestV2(t *testing.T) {
	testAddr := crypto.PubkeyToAddress(testKey.PublicKey)
	backend := simulated.NewBackend(
		types.GenesisAlloc{
			testAddr: {Balance: big.NewInt(10000000000000000)},
		},
	)
	defer backend.Close()

	contractABI, err := JSON(strings.NewReader(contractABIStr))
	if err != nil {
		panic(err)
	}

	signer := types.LatestSigner(params.AllDevChainProtocolChanges)
	opts := bind.TransactOpts{
		From:  testAddr,
		Nonce: nil,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			signature, err := crypto.Sign(signer.Hash(tx).Bytes(), testKey)
			if err != nil {
				t.Fatal(err)
			}
			signedTx, err := tx.WithSignature(signer, signature)
			if err != nil {
				t.Fatal(err)
			}
			return signedTx, nil
		},
		Context: context.Background(),
		/*
			Value:      nil,
			GasPrice:   nil,
			GasFeeCap:  nil,
			GasTipCap:  nil,
			GasLimit:   0,
			AccessList: nil,
			NoSend:     false,
		*/
	}
	// we should just be able to use the backend directly, instead of using
	// this deprecated interface.  However, the simulated backend no longer
	// implements backends.SimulatedBackend...
	bindBackend := backends.SimulatedBackend{
		Backend: backend,
		Client:  backend.Client(),
	}
	_, _, _, err = bind.DeployContract(&opts, contractABI, common.Hex2Bytes(deployer), &bindBackend)
	if err != nil {
		t.Fatal(err)
	}
	// send a balance to our contract (contract must accept ether by default)
}
