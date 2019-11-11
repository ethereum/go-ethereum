// Copyright 2019 The go-ethereum Authors
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

// package accountbook implements the contract based micropayment for les server.
package accountbook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contracts/accountbook/contract"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	ChallengeTimeWindow = 64              // The default challenge block numbers, which euqals to 16mins
	deployTimeout       = 5 * time.Minute // The maxmium waiting time for contract deployment.
)

// Cheque is a document that orders a bank(contract) to pay a specific amount
// of money from a person's account to the person in whose name the cheque has
// been issued(contract owner). The cheque is signed by drawer so that he can't
// deny it.
//
// What is different from traditional cheques is: the amount of the cheque is
// cumulative. So that contract can easily check whether the cheque is double-cash
// by the payee.
//
// TODO(rjl493456442) add CHAINID
type Cheque struct {
	Drawer       common.Address // The drawer of the cheque
	ContractAddr common.Address // The address of the accountbook contract(bank address)
	Amount       *big.Int       // The cumulative amount of the issued cheque
	Sig          [crypto.SignatureLength]byte
}

type chequeRLP struct {
	ContractAddr common.Address // The address of the accountbook contract(bank address)
	Amount       *big.Int       // The cumulative amount of the issued cheque
	Sig          [crypto.SignatureLength]byte
}

// EncodeRLP implements rlp.Encoder, and flattens the necessary fields of a cheque
// into an RLP stream.
func (c *Cheque) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &chequeRLP{ContractAddr: c.ContractAddr, Amount: c.Amount, Sig: c.Sig})
}

// DecodeRLP implements rlp.Decoder, and loads the rlp-encoded fields of a cheque
// from an RLP stream.
func (c *Cheque) DecodeRLP(s *rlp.Stream) error {
	var dec chequeRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	c.ContractAddr, c.Amount, c.Sig = dec.ContractAddr, dec.Amount, dec.Sig
	// If the cheque doesn't contain a signature, skip resolving the drawer address.
	if c.Sig == [65]byte{} {
		return nil
	}
	drawer, err := c.recoverDrawer()
	if err != nil {
		return err
	}
	c.Drawer = drawer
	return nil
}

// recoverDrawer resolves the drawer address from the cheque content
// and signed signature.
func (c *Cheque) recoverDrawer() (common.Address, error) {
	// EIP 191 style signatures
	//
	// Arguments when calculating hash to validate
	// 1: byte(0x19) - the initial 0x19 byte
	// 2: byte(0) - the version byte (data with intended validator)
	// 3: this - the validator address
	// --  Application specific data
	// 4: amount(uint256) big endian 32bytes
	buf := make([]byte, 32)
	copy(buf[32-len(c.Amount.Bytes()):], c.Amount.Bytes())
	data := append([]byte{0x19, 0x00}, append(c.ContractAddr.Bytes(), buf...)...)

	// Transform V from 27/28 to 0/1 according to the yellow paper
	c.Sig[64] -= 27
	defer func() {
		c.Sig[64] += 27
	}()
	pubkey, err := crypto.SigToPub(crypto.Keccak256(data), c.Sig[:])
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*pubkey), nil
}

// validate verifies whether the cheque is signed properly and all fields
// are filled.
func (c *Cheque) validate(chanAddr common.Address) error {
	drawer, err := c.recoverDrawer()
	if err != nil {
		return err
	}
	if drawer != c.Drawer {
		return errors.New("invalid signature")
	}
	if c.ContractAddr != chanAddr {
		return errors.New("unsolicited cheque")
	}
	if c.Amount == nil {
		return errors.New("incomplete cheque")
	}
	return nil
}

// sign generates the digital signature for cheque by clef. It's a bit
// different with signWithKey, we need to construct a RPC call with clef
// format.
func (c *Cheque) sign(signFn func(data []byte) ([]byte, error)) error {
	// EIP 191 style signatures
	//
	// Arguments when calculating hash to validate
	// 1: byte(0x19) - the initial 0x19 byte
	// 2: byte(0) - the version byte (data with intended validator)
	// 3: this - the validator address
	// --  Application specific data
	// 4 : amount(uint256) big endian 32bytes
	p := make(map[string]string)
	p["address"] = c.ContractAddr.Hex()
	buf := make([]byte, 32)
	copy(buf[32-len(c.Amount.Bytes()):], c.Amount.Bytes())
	p["message"] = hexutil.Encode(buf)
	encoded, err := json.Marshal(p)
	if err != nil {
		return err
	}
	sig, err := signFn(encoded)
	if err != nil {
		return err
	}
	copy(c.Sig[:], sig)
	return nil
}

// signWithKey signes the cheque with privatekey. Only use it in testing.
func (c *Cheque) signWithKey(signFn func(digestHash []byte) ([]byte, error)) error {
	// EIP 191 style signatures
	//
	// Arguments when calculating hash to validate
	// 1: byte(0x19) - the initial 0x19 byte
	// 2: byte(0) - the version byte (data with intended validator)
	// 3: this - the validator address
	// --  Application specific data
	// 4 : amount(uint256) big endian 32bytes
	buf := make([]byte, 32)
	copy(buf[32-len(c.Amount.Bytes()):], c.Amount.Bytes())
	data := append([]byte{0x19, 0x00}, append(c.ContractAddr.Bytes(), buf...)...)
	sig, err := signFn(crypto.Keccak256(data))
	if err != nil {
		return err
	}
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	copy(c.Sig[:], sig)
	return nil
}

// AccountBook represents a contract instance which holds all drawer's deposits.
type AccountBook struct {
	address  common.Address
	contract *contract.AccountBook
}

// NewAccountBook deploys a new accountbook contract or initializes
// a exist contract by given address.
//
// Note this function can take several minutes for execution.
func newAccountBook(address common.Address, contractBackend bind.ContractBackend) (*AccountBook, error) {
	log.Info("Initialized accountbook contract", "address", address)
	c, err := contract.NewAccountBook(address, contractBackend)
	if err != nil {
		return nil, err
	}
	return &AccountBook{contract: c, address: address}, nil
}

// deployAccountBook deploys the accountbook smart contract and waits the transaction
// is confirmed by network.
func deployAccountBook(auth *bind.TransactOpts, contractBackend bind.ContractBackend, deployBackend bind.DeployBackend) (common.Address, error) {
	log.Info("Deploying accountbook contract")
	start := time.Now()
	_, tx, _, err := contract.DeployAccountBook(auth, contractBackend, uint64(ChallengeTimeWindow))
	if err != nil {
		return common.Address{}, err
	}
	context, cancelFn := context.WithTimeout(context.Background(), deployTimeout)
	defer cancelFn()
	addr, err := bind.WaitDeployed(context, deployBackend, tx)
	if err != nil {
		return common.Address{}, err
	}
	log.Info("Deployed accountbook contract", "address", addr, "elapsed", common.PrettyDuration(time.Since(start)))
	return addr, nil
}
