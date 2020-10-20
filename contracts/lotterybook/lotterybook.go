// Copyright 2020 The go-ethereum Authors
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

// lotterybook is an on-chain lotterybook which light client can use to
// send payments to several different servers cheap and efficiently.
package lotterybook

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// Blockchain defines a small collection of methods needed to access
// the local blockchain.
type Blockchain interface {
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.Header

	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByNumber(number uint64) *types.Header

	// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

// Lottery payment protocol constraints
//
// In order to handle block reorg properly, all newly created lottery has to wait
// at least lotteryProcessConfirms confirms before it's used.
//
// Besides we have to consider the blockchain height bias between the payment sender
// and receiver. E.g. the sender is lag behine(block N) while the receiver is already
// in N+5. But the lottery reveal number is at N+2. So in this case we can't use this
// almost expired lottery anymore. And also for receiver, it's not very safe to accpet
// the cheques based on the almost expired lottery since the sender may already know
// the reveal hash. In these two cases lotterySafetyMargin is applied.
const (
	// lotteryProcessConfirms is the number of confirmations before a on-chain
	// status/event is considered stable.
	lotteryProcessConfirms = 6

	// lotteryClaimPeriod is the maximum block number lottery winner can claim
	// the whole deposit.
	lotteryClaimPeriod = 256

	// lotterySafetyMargin is the constraint for accepting or issuing cheque.
	lotterySafetyMargin = 2

	// lotterySafetyThreshold is the constraint that we can consider the sender
	// is deliberately sending us useless cheques.
	lotterySafetyThreshold = 30

	// maxSignedRange is the maximum uint64 which is used to represent the cheque
	// is never used.
	maxSignedRange = math.MaxUint64
)

var txTimeout = 5 * time.Minute // The maxmium waiting time for blockchain to include on-chain transaction

// Lottery defines the minimal required fields
type Lottery struct {
	Id           common.Hash      // The id of lottery
	Amount       uint64           // The amount of lottery
	RevealNumber uint64           // The reveal number of lottery
	Receivers    []common.Address // A batch of receivers included in this lottery

	// Additional helper fields. These fields are only derived
	// if the on-chain deposit transaction has been confirmed.
	GasPrice  *big.Int
	Nonce     uint64
	CreateAt  uint64
	Confirmed bool

	// Additional helper fields but shouldn't be persisted.
	NextCheck uint64 `rlp:"-"`
	Checks    int    `rlp:"-"`
	Lost      bool   `rlp:"-"`
}

// hasReceiver returns an indicator whether the given address is the
// receiver of lottery.
func (l *Lottery) hasReceiver(address common.Address) bool {
	for _, r := range l.Receivers {
		if r == address {
			return true
		}
	}
	return false
}

// balance returns the corresponding balance of the given receiver
// contained in this lottery.
func (l *Lottery) balance(address common.Address, cheque *Cheque) uint64 {
	if !l.hasReceiver(address) {
		return 0
	}
	assigned := l.Amount >> (len(cheque.Witness) - 1)
	if cheque.SignedRange == maxSignedRange {
		return assigned
	}
	return uint64(float64(cheque.UpperLimit-cheque.SignedRange) / float64(cheque.UpperLimit-cheque.LowerLimit+1) * float64(assigned))
}

// copy returns the deep copied lottery.
func (l *Lottery) copy() *Lottery {
	var receiver []common.Address
	copy(receiver, l.Receivers)
	return &Lottery{
		Id:           l.Id,
		Amount:       l.Amount,
		RevealNumber: l.RevealNumber,
		Receivers:    receiver,
	}
}

// LotteryByRevealTime implements the sort interface to allow sorting a list
// of lotteries by their reveal time.
type LotteryByRevealTime []*Lottery

func (s LotteryByRevealTime) Len() int           { return len(s) }
func (s LotteryByRevealTime) Less(i, j int) bool { return s[i].RevealNumber < s[j].RevealNumber }
func (s LotteryByRevealTime) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// LotteryStatus defines the status of locally created lottery.
type LotteryStatus int

const (
	LotteryPending LotteryStatus = iota
	LotteryActive
	LotteryRevealed
	LotteryExpired
	LotteryLost
)

// LotteryEvent wraps a lottery id and corresponding status.
type LotteryEvent struct {
	Id     common.Hash
	Status LotteryStatus

	// Additional field, can be nil
	Lottery *Lottery
}

// Cheque is a document that orders a bank(contract) to pay a specific amount
// of money from a person's account to the person in whose name the cheque has
// been issued(contract owner). The cheque is signed by drawer so that he can't
// deny it.
//
// What is different from traditional cheques is it's a lottery. Cheque payees
// don’t have a 100% chance of getting paid at the end. In the contract, instead
// of a batch of deposits created for different payees, only a single lottery is
// created with a batch of payees included. Different payee has different chance
// to win the lottery. When the lottery is revealed, payee can check whether it's
// the lucky winner to claim the whole lottery or nothing. Each payee has a probability
// range to claim the lottery, if the block hash of reveal height falls into this
// range, then the payee is the winner. The expect payment amount is chance * lottery_amount.
//
// Besides payer doesn't have to pay the whole amount in a single cheque. It can divide
// into several cheques with a floated claim upper hash range. The higher upper range,
// the higher amount is paid.
//
// TODO(rjl493456442) add CHAINID
type Cheque struct {
	Witness      []common.Hash  // The merkle proof that proves the drawee is included in the lottery
	ContractAddr common.Address // The address of the lotterybook contract(bank address)

	// RevealRange is the upper reveal range for payee to claim lottery.
	// Each payee will have a default probability range which derived by
	// witness. However the real reveal range can be a subset of this
	// probability range by setting different upper reveal range. In this
	// way we can divide a "deposit" for payee into different small parts.
	//
	// RevealRange is encoded in big-endian order.
	//
	// If the revealRange is nil, it means the cheque is not used yet.
	// Otherwise the length of revealRange must be 4(uint32).
	RevealRange []byte

	// Salt is the random number which used to calculate lottery id.
	// The id of lottery is derived by formula: keccak256<merkle_root, salt>
	Salt uint64

	// ReceiverSalt is the random number which used to calculate receiver hash.
	// The hash is derived by formula: keccak256<receiver_addr, receiver_salt>
	ReceiverSalt uint64

	// The signature to protect the correctness of all above fields.
	Sig [crypto.SignatureLength]byte

	// **Receiver side speicific fields**. These fields will never be assigned
	// or used in the sender side, but they are important for the receiver.
	RevealNumber uint64
	Amount       uint64

	// These following fields are defined and derived locally
	MerkleRoot common.Hash // The merkle tree root hash of corresponding lottery
	LotteryId  common.Hash // The id of corresponding lottery
	LowerLimit uint64      // The lower limit for claiming lottery
	UpperLimit uint64      // The upper limit for claiming lottery

	// The real signed amount is: (SignedRange - LowerLimit + 1) / (UpperLimit - LowerLimit + 1) * Assigned
	SignedRange uint64         // The uint64 format of revealRange, maxUint64 if revealRange is empty(cheque is not used)
	signer      common.Address // The signer of the cheque
}

type chequeRLP struct {
	Witness      []common.Hash  // The merkle proof that proves the drawee is included in the lottery
	ContractAddr common.Address // The address of the accountbook contract(bank address)
	RevealRange  []byte         // The upper reveal range for payee to claim lottery
	Salt         uint64         // The random number of lottery
	ReceiverSalt uint64         // The random number of receiver
	Sig          [crypto.SignatureLength]byte

	RevealNumber uint64 // The associated lottery reveal number(0 in sender side)
	Amount       uint64 // The associated lottery amount(0 in sender side)
}

// EncodeRLP implements rlp.Encoder, and flattens the necessary fields of a cheque
// into an RLP stream.
func (c *Cheque) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &chequeRLP{Witness: c.Witness, ContractAddr: c.ContractAddr, RevealRange: c.RevealRange, Salt: c.Salt, ReceiverSalt: c.ReceiverSalt, Sig: c.Sig, RevealNumber: c.RevealNumber, Amount: c.Amount})
}

// DecodeRLP implements rlp.Decoder, loads the rlp-encoded fields of a cheque
// from an RLP stream and derive all local defined fields.
func (c *Cheque) DecodeRLP(s *rlp.Stream) error {
	var dec chequeRLP
	if err := s.Decode(&dec); err != nil {
		return err
	}
	c.Witness, c.ContractAddr, c.RevealRange, c.Salt, c.ReceiverSalt, c.Sig, c.RevealNumber, c.Amount = dec.Witness, dec.ContractAddr, dec.RevealRange, dec.Salt, dec.ReceiverSalt, dec.Sig, dec.RevealNumber, dec.Amount

	// RLP wil convert the nil slice to []byte{}, set it back
	if len(c.RevealRange) == 0 {
		c.RevealRange = nil
	}
	if err := c.deriveFields(); err != nil {
		return err
	}
	return nil
}

// newCheque creates a blank cheque for sepcific receiver. All internal
// fields will be derived here. Note the returned cheque is NOT signed.
func newCheque(witness []common.Hash, contractAddr common.Address, salt, receiverSalt uint64) (*Cheque, error) {
	cheque := &Cheque{
		Witness:      witness,
		ContractAddr: contractAddr,
		Salt:         salt,
		ReceiverSalt: receiverSalt,
	}
	if err := cheque.deriveFields(); err != nil {
		return nil, err
	}
	return cheque, nil
}

// DeriveFields fills the cheque with their computed fields based on RLP-encoded data.
func (c *Cheque) deriveFields() error {
	// Derive merkle tree root hash and the position of signed entity based on witness.
	var position uint64
	if len(c.Witness) == 0 {
		return errors.New("empty witness")
	}
	// The first witness element is the hash of leaf.
	c.MerkleRoot = c.Witness[0]
	if len(c.Witness) != 1 {
		for i := 1; i < len(c.Witness); i++ {
			if bytes.Compare(c.MerkleRoot.Bytes(), c.Witness[i].Bytes()) < 0 {
				c.MerkleRoot = crypto.Keccak256Hash(append(c.MerkleRoot.Bytes(), c.Witness[i].Bytes()...))
			} else {
				c.MerkleRoot = crypto.Keccak256Hash(append(c.Witness[i].Bytes(), c.MerkleRoot.Bytes()...))
				position += 1 << (i - 1)
			}
		}
	}
	// Derive lottery id based on merkle tree and salt
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, c.Salt)
	c.LotteryId = crypto.Keccak256Hash(append(c.MerkleRoot.Bytes(), buf...))

	// Derive claim hash range based on the position, the assigned
	// reveal range is [lowerLimit, upperLimit]
	interval := uint64((math.MaxUint32 + 1) >> (len(c.Witness) - 1))
	c.LowerLimit = interval * position
	c.UpperLimit = interval*(position+1) - 1

	// Derive signed range, also ensure the reveal range is a reasonable value.
	if len(c.RevealRange) == 0 {
		c.SignedRange = maxSignedRange
		return nil
	}
	if len(c.RevealRange) != 4 {
		return fmt.Errorf("invalid reveal range length %d", len(c.RevealRange))
	}
	c.SignedRange = uint64(binary.BigEndian.Uint32(c.RevealRange))
	if c.SignedRange < c.LowerLimit || c.SignedRange > c.UpperLimit {
		return errors.New("invalid reveal range")
	}
	return nil
}

// sigHash returns the hash for signinng. Please ensure all fields are derived.
func (c *Cheque) sigHash() common.Hash {
	// EIP 191 style signatures
	//
	// Arguments when calculating hash to validate
	// 1: byte(0x19) - the initial 0x19 byte
	// 2: byte(0) - the version byte (data with intended validator)
	// 3: this - the validator address
	// --  Application specific data
	// 4: id - the id of lottery which is derived by keccak256(root+salt)
	// 5: range - the promised hash range allowed for lottery redemption
	var appContent []byte
	appContent = append(appContent, c.LotteryId.Bytes()...)
	appContent = append(appContent, c.RevealRange...)
	data := append([]byte{0x19, 0x00}, append(c.ContractAddr.Bytes(), appContent...)...)
	return crypto.Keccak256Hash(data)
}

// deriveSigner resolves the drawer address from the cheque content
// and signed signature.
func (c *Cheque) Signer() common.Address {
	// Short circuit if it's already derived.
	if c.signer != (common.Address{}) {
		return c.signer
	}
	// Transform V from 27/28 to 0/1 according to the yellow paper
	c.Sig[64] -= 27
	defer func() {
		c.Sig[64] += 27
	}()
	pubkey, err := crypto.SigToPub(c.sigHash().Bytes(), c.Sig[:])
	if err != nil {
		return common.Address{}
	}
	c.signer = crypto.PubkeyToAddress(*pubkey)
	return c.signer
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
	// 4: id - the id of lottery which is derived by keccak256(root+salt)
	// 5: range - the promised hash range allowed for lottery redemption
	p := make(map[string]string)
	p["address"] = c.ContractAddr.Hex()
	var appContent []byte
	appContent = append(appContent, c.LotteryId.Bytes()...)
	appContent = append(appContent, c.RevealRange...)
	p["message"] = hexutil.Encode(appContent)
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
	sig, err := signFn(c.sigHash().Bytes())
	if err != nil {
		return err
	}
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	copy(c.Sig[:], sig)
	return nil
}

// reveal returns an indicator whether this cheque is the winner.
func (c *Cheque) reveal(hash common.Hash) bool {
	// Short circuit if the cheque is never used yet.
	if c.SignedRange == maxSignedRange {
		return false
	}
	// Use the highest eight bytes in big-endian order to construct reveal number.
	var trimmed [4]byte
	copy(trimmed[:], hash.Bytes()[common.HashLength-4:])
	value := uint64(binary.BigEndian.Uint32(trimmed[:]))
	return c.SignedRange >= value && c.LowerLimit <= value
}

// copy returns the deep copied cheque.
func (c *Cheque) copy() *Cheque {
	var witness []common.Hash
	copy(witness, c.Witness)
	return &Cheque{
		Witness:      witness,
		ContractAddr: c.ContractAddr,
		RevealRange:  common.CopyBytes(c.RevealRange),
		Salt:         c.Salt,
		ReceiverSalt: c.ReceiverSalt,
		Sig:          c.Sig,
		RevealNumber: c.RevealNumber,
		Amount:       c.Amount,
		MerkleRoot:   c.MerkleRoot,
		LotteryId:    c.LotteryId,
		LowerLimit:   c.LowerLimit,
		UpperLimit:   c.UpperLimit,
		SignedRange:  c.SignedRange,
		signer:       c.signer,
	}
}

// validateCheque checks whether the provided cheque is valid or not.
func validateCheque(c *Cheque, sender, receiver, contract common.Address) error {
	if c.ContractAddr != contract {
		return errors.New("unsolicited cheque")
	}
	if c.Signer() != sender {
		return errors.New("invalid sender")
	}
	// During the RLP-decode we will filter invalid cheque whose
	// witness is empty. Ensure the receiver of cheque actually
	// matches the given address.
	var buff [8]byte
	binary.BigEndian.PutUint64(buff[:], c.ReceiverSalt)
	if crypto.Keccak256Hash(append(receiver.Bytes(), buff[:]...)) != c.Witness[0] {
		return errors.New("invalid receiver")
	}
	return nil
}

// LotteryBook represents a contract instance which used to issue and verify payments
type LotteryBook struct {
	address  common.Address
	contract *contract.LotteryBook
}

// NewLotteryBook deploys a new lotterybook contract or initializes
// a exist contract by given address.
//
// Note this function can take several minutes for execution.
func newLotteryBook(address common.Address, contractBackend bind.ContractBackend) (*LotteryBook, error) {
	log.Info("Initialized lottery contract", "address", address)
	c, err := contract.NewLotteryBook(address, contractBackend)
	if err != nil {
		return nil, err
	}
	return &LotteryBook{contract: c, address: address}, nil
}

// DeployLotteryBook deploys the lotterybook smart contract.
func DeployLotteryBook(auth *bind.TransactOpts, contractBackend bind.ContractBackend) (common.Address, *LotteryBook, error) {
	addr, _, c, err := contract.DeployLotteryBook(auth, contractBackend)
	if err != nil {
		return common.Address{}, nil, err
	}
	log.Info("Deployed lotterybook contract", "address", addr)
	return addr, &LotteryBook{contract: c, address: addr}, nil
}
