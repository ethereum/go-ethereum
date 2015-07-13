// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

// Block Test JSON Format
type BlockTest struct {
	Genesis *types.Block

	Json        *btJSON
	preAccounts map[string]btAccount
}

type btJSON struct {
	Blocks             []btBlock
	GenesisBlockHeader btHeader
	Pre                map[string]btAccount
	PostState          map[string]btAccount
}

type btBlock struct {
	BlockHeader  *btHeader
	Rlp          string
	Transactions []btTransaction
	UncleHeaders []*btHeader
}

type btAccount struct {
	Balance    string
	Code       string
	Nonce      string
	Storage    map[string]string
	PrivateKey string
}

type btHeader struct {
	Bloom            string
	Coinbase         string
	MixHash          string
	Nonce            string
	Number           string
	ParentHash       string
	ReceiptTrie      string
	SeedHash         string
	StateRoot        string
	TransactionsTrie string
	UncleHash        string

	ExtraData  string
	Difficulty string
	GasLimit   string
	GasUsed    string
	Timestamp  string
}

type btTransaction struct {
	Data     string
	GasLimit string
	GasPrice string
	Nonce    string
	R        string
	S        string
	To       string
	V        string
	Value    string
}

func RunBlockTestWithReader(r io.Reader, skipTests []string) error {
	btjs := make(map[string]*btJSON)
	if err := readJson(r, &btjs); err != nil {
		return err
	}

	bt, err := convertBlockTests(btjs)
	if err != nil {
		return err
	}

	if err := runBlockTests(bt, skipTests); err != nil {
		return err
	}
	return nil
}

func RunBlockTest(file string, skipTests []string) error {
	btjs := make(map[string]*btJSON)
	if err := readJsonFile(file, &btjs); err != nil {
		return err
	}

	bt, err := convertBlockTests(btjs)
	if err != nil {
		return err
	}
	if err := runBlockTests(bt, skipTests); err != nil {
		return err
	}
	return nil
}

func runBlockTests(bt map[string]*BlockTest, skipTests []string) error {
	skipTest := make(map[string]bool, len(skipTests))
	for _, name := range skipTests {
		skipTest[name] = true
	}

	for name, test := range bt {
		// if the test should be skipped, return
		if skipTest[name] {
			glog.Infoln("Skipping block test", name)
			continue
		}

		// test the block
		if err := runBlockTest(test); err != nil {
			return err
		}
		glog.Infoln("Block test passed: ", name)

	}
	return nil

}
func runBlockTest(test *BlockTest) error {
	cfg := test.makeEthConfig()
	cfg.GenesisBlock = test.Genesis

	ethereum, err := eth.New(cfg)
	if err != nil {
		return err
	}

	err = ethereum.Start()
	if err != nil {
		return err
	}

	// import pre accounts
	statedb, err := test.InsertPreState(ethereum)
	if err != nil {
		return fmt.Errorf("InsertPreState: %v", err)
	}

	err = test.TryBlocksInsert(ethereum.ChainManager())
	if err != nil {
		return err
	}

	if err = test.ValidatePostState(statedb); err != nil {
		return fmt.Errorf("post state validation failed: %v", err)
	}
	return nil
}

func (test *BlockTest) makeEthConfig() *eth.Config {
	ks := crypto.NewKeyStorePassphrase(filepath.Join(common.DefaultDataDir(), "keystore"))

	return &eth.Config{
		DataDir:        common.DefaultDataDir(),
		Verbosity:      5,
		Etherbase:      common.Address{},
		AccountManager: accounts.NewManager(ks),
		NewDB:          func(path string) (common.Database, error) { return ethdb.NewMemDatabase() },
	}
}

// InsertPreState populates the given database with the genesis
// accounts defined by the test.
func (t *BlockTest) InsertPreState(ethereum *eth.Ethereum) (*state.StateDB, error) {
	db := ethereum.StateDb()
	statedb := state.New(common.Hash{}, db)
	for addrString, acct := range t.preAccounts {
		addr, err := hex.DecodeString(addrString)
		if err != nil {
			return nil, err
		}
		code, err := hex.DecodeString(strings.TrimPrefix(acct.Code, "0x"))
		if err != nil {
			return nil, err
		}
		balance, ok := new(big.Int).SetString(acct.Balance, 0)
		if !ok {
			return nil, err
		}
		nonce, err := strconv.ParseUint(prepInt(16, acct.Nonce), 16, 64)
		if err != nil {
			return nil, err
		}

		if acct.PrivateKey != "" {
			privkey, err := hex.DecodeString(strings.TrimPrefix(acct.PrivateKey, "0x"))
			err = crypto.ImportBlockTestKey(privkey)
			err = ethereum.AccountManager().TimedUnlock(common.BytesToAddress(addr), "", 999999*time.Second)
			if err != nil {
				return nil, err
			}
		}

		obj := statedb.CreateAccount(common.HexToAddress(addrString))
		obj.SetCode(code)
		obj.SetBalance(balance)
		obj.SetNonce(nonce)
		for k, v := range acct.Storage {
			statedb.SetState(common.HexToAddress(addrString), common.HexToHash(k), common.HexToHash(v))
		}
	}
	// sync objects to trie
	statedb.SyncObjects()
	// sync trie to disk
	statedb.Sync()

	if !bytes.Equal(t.Genesis.Root().Bytes(), statedb.Root().Bytes()) {
		return nil, fmt.Errorf("computed state root does not match genesis block %x %x", t.Genesis.Root().Bytes()[:4], statedb.Root().Bytes()[:4])
	}
	return statedb, nil
}

/* See https://github.com/ethereum/tests/wiki/Blockchain-Tests-II

   Whether a block is valid or not is a bit subtle, it's defined by presence of
   blockHeader, transactions and uncleHeaders fields. If they are missing, the block is
   invalid and we must verify that we do not accept it.

   Since some tests mix valid and invalid blocks we need to check this for every block.

   If a block is invalid it does not necessarily fail the test, if it's invalidness is
   expected we are expected to ignore it and continue processing and then validate the
   post state.
*/
func (t *BlockTest) TryBlocksInsert(chainManager *core.ChainManager) error {
	// insert the test blocks, which will execute all transactions
	for _, b := range t.Json.Blocks {
		cb, err := mustConvertBlock(b)
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return fmt.Errorf("Block RLP decoding failed when expected to succeed: %v", err)
			}
		}
		// RLP decoding worked, try to insert into chain:
		_, err = chainManager.InsertChain(types.Blocks{cb})
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return fmt.Errorf("Block insertion into chain failed: %v", err)
			}
		}
		if b.BlockHeader == nil {
			return fmt.Errorf("Block insertion should have failed")
		}
		err = t.validateBlockHeader(b.BlockHeader, cb.Header())
		if err != nil {
			return fmt.Errorf("Block header validation failed: %v", err)
		}
	}
	return nil
}

func (s *BlockTest) validateBlockHeader(h *btHeader, h2 *types.Header) error {
	expectedBloom := mustConvertBytes(h.Bloom)
	if !bytes.Equal(expectedBloom, h2.Bloom.Bytes()) {
		return fmt.Errorf("Bloom: expected: %v, decoded: %v", expectedBloom, h2.Bloom.Bytes())
	}

	expectedCoinbase := mustConvertBytes(h.Coinbase)
	if !bytes.Equal(expectedCoinbase, h2.Coinbase.Bytes()) {
		return fmt.Errorf("Coinbase: expected: %v, decoded: %v", expectedCoinbase, h2.Coinbase.Bytes())
	}

	expectedMixHashBytes := mustConvertBytes(h.MixHash)
	if !bytes.Equal(expectedMixHashBytes, h2.MixDigest.Bytes()) {
		return fmt.Errorf("MixHash: expected: %v, decoded: %v", expectedMixHashBytes, h2.MixDigest.Bytes())
	}

	expectedNonce := mustConvertBytes(h.Nonce)
	if !bytes.Equal(expectedNonce, h2.Nonce[:]) {
		return fmt.Errorf("Nonce: expected: %v, decoded: %v", expectedNonce, h2.Nonce)
	}

	expectedNumber := mustConvertBigInt(h.Number, 16)
	if expectedNumber.Cmp(h2.Number) != 0 {
		return fmt.Errorf("Number: expected: %v, decoded: %v", expectedNumber, h2.Number)
	}

	expectedParentHash := mustConvertBytes(h.ParentHash)
	if !bytes.Equal(expectedParentHash, h2.ParentHash.Bytes()) {
		return fmt.Errorf("Parent hash: expected: %v, decoded: %v", expectedParentHash, h2.ParentHash.Bytes())
	}

	expectedReceiptHash := mustConvertBytes(h.ReceiptTrie)
	if !bytes.Equal(expectedReceiptHash, h2.ReceiptHash.Bytes()) {
		return fmt.Errorf("Receipt hash: expected: %v, decoded: %v", expectedReceiptHash, h2.ReceiptHash.Bytes())
	}

	expectedTxHash := mustConvertBytes(h.TransactionsTrie)
	if !bytes.Equal(expectedTxHash, h2.TxHash.Bytes()) {
		return fmt.Errorf("Tx hash: expected: %v, decoded: %v", expectedTxHash, h2.TxHash.Bytes())
	}

	expectedStateHash := mustConvertBytes(h.StateRoot)
	if !bytes.Equal(expectedStateHash, h2.Root.Bytes()) {
		return fmt.Errorf("State hash: expected: %v, decoded: %v", expectedStateHash, h2.Root.Bytes())
	}

	expectedUncleHash := mustConvertBytes(h.UncleHash)
	if !bytes.Equal(expectedUncleHash, h2.UncleHash.Bytes()) {
		return fmt.Errorf("Uncle hash: expected: %v, decoded: %v", expectedUncleHash, h2.UncleHash.Bytes())
	}

	expectedExtraData := mustConvertBytes(h.ExtraData)
	if !bytes.Equal(expectedExtraData, h2.Extra) {
		return fmt.Errorf("Extra data: expected: %v, decoded: %v", expectedExtraData, h2.Extra)
	}

	expectedDifficulty := mustConvertBigInt(h.Difficulty, 16)
	if expectedDifficulty.Cmp(h2.Difficulty) != 0 {
		return fmt.Errorf("Difficulty: expected: %v, decoded: %v", expectedDifficulty, h2.Difficulty)
	}

	expectedGasLimit := mustConvertBigInt(h.GasLimit, 16)
	if expectedGasLimit.Cmp(h2.GasLimit) != 0 {
		return fmt.Errorf("GasLimit: expected: %v, decoded: %v", expectedGasLimit, h2.GasLimit)
	}
	expectedGasUsed := mustConvertBigInt(h.GasUsed, 16)
	if expectedGasUsed.Cmp(h2.GasUsed) != 0 {
		return fmt.Errorf("GasUsed: expected: %v, decoded: %v", expectedGasUsed, h2.GasUsed)
	}

	expectedTimestamp := mustConvertUint(h.Timestamp, 16)
	if expectedTimestamp != h2.Time {
		return fmt.Errorf("Timestamp: expected: %v, decoded: %v", expectedTimestamp, h2.Time)
	}

	return nil
}

func (t *BlockTest) ValidatePostState(statedb *state.StateDB) error {
	for addrString, acct := range t.preAccounts {
		// XXX: is is worth it checking for errors here?
		addr, err := hex.DecodeString(addrString)
		if err != nil {
			return err
		}
		code, err := hex.DecodeString(strings.TrimPrefix(acct.Code, "0x"))
		if err != nil {
			return err
		}
		balance, ok := new(big.Int).SetString(acct.Balance, 0)
		if !ok {
			return err
		}
		nonce, err := strconv.ParseUint(prepInt(16, acct.Nonce), 16, 64)
		if err != nil {
			return err
		}

		// address is indirectly verified by the other fields, as it's the db key
		code2 := statedb.GetCode(common.BytesToAddress(addr))
		balance2 := statedb.GetBalance(common.BytesToAddress(addr))
		nonce2 := statedb.GetNonce(common.BytesToAddress(addr))
		if !bytes.Equal(code2, code) {
			return fmt.Errorf("account code mismatch, addr, found, expected: ", addrString, hex.EncodeToString(code2), hex.EncodeToString(code))
		}
		if balance2.Cmp(balance) != 0 {
			return fmt.Errorf("account balance mismatch, addr, found, expected: ", addrString, balance2, balance)
		}
		if nonce2 != nonce {
			return fmt.Errorf("account nonce mismatch, addr, found, expected: ", addrString, nonce2, nonce)
		}
	}
	return nil
}

func convertBlockTests(in map[string]*btJSON) (map[string]*BlockTest, error) {
	out := make(map[string]*BlockTest)
	for name, test := range in {
		var err error
		if out[name], err = convertBlockTest(test); err != nil {
			return out, fmt.Errorf("bad test %q: %v", name, err)
		}
	}
	return out, nil
}

func convertBlockTest(in *btJSON) (out *BlockTest, err error) {
	// the conversion handles errors by catching panics.
	// you might consider this ugly, but the alternative (passing errors)
	// would be much harder to read.
	defer func() {
		if recovered := recover(); recovered != nil {
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf, false)]
			err = fmt.Errorf("%v\n%s", recovered, buf)
		}
	}()
	out = &BlockTest{preAccounts: in.Pre, Json: in}
	out.Genesis = mustConvertGenesis(in.GenesisBlockHeader)
	return out, err
}

func mustConvertGenesis(testGenesis btHeader) *types.Block {
	hdr := mustConvertHeader(testGenesis)
	hdr.Number = big.NewInt(0)
	b := types.NewBlockWithHeader(hdr)
	b.Td = new(big.Int)
	return b
}

func mustConvertHeader(in btHeader) *types.Header {
	// hex decode these fields
	header := &types.Header{
		//SeedHash:    mustConvertBytes(in.SeedHash),
		MixDigest:   mustConvertHash(in.MixHash),
		Bloom:       mustConvertBloom(in.Bloom),
		ReceiptHash: mustConvertHash(in.ReceiptTrie),
		TxHash:      mustConvertHash(in.TransactionsTrie),
		Root:        mustConvertHash(in.StateRoot),
		Coinbase:    mustConvertAddress(in.Coinbase),
		UncleHash:   mustConvertHash(in.UncleHash),
		ParentHash:  mustConvertHash(in.ParentHash),
		Extra:       mustConvertBytes(in.ExtraData),
		GasUsed:     mustConvertBigInt(in.GasUsed, 16),
		GasLimit:    mustConvertBigInt(in.GasLimit, 16),
		Difficulty:  mustConvertBigInt(in.Difficulty, 16),
		Time:        mustConvertUint(in.Timestamp, 16),
		Nonce:       types.EncodeNonce(mustConvertUint(in.Nonce, 16)),
	}
	return header
}

func mustConvertBlock(testBlock btBlock) (*types.Block, error) {
	var b types.Block
	r := bytes.NewReader(mustConvertBytes(testBlock.Rlp))
	err := rlp.Decode(r, &b)
	return &b, err
}

func mustConvertBytes(in string) []byte {
	if in == "0x" {
		return []byte{}
	}
	h := unfuckFuckedHex(strings.TrimPrefix(in, "0x"))
	out, err := hex.DecodeString(h)
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q: ", h, err))
	}
	return out
}

func mustConvertHash(in string) common.Hash {
	out, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", in))
	}
	return common.BytesToHash(out)
}

func mustConvertAddress(in string) common.Address {
	out, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", in))
	}
	return common.BytesToAddress(out)
}

func mustConvertBloom(in string) types.Bloom {
	out, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", in))
	}
	return types.BytesToBloom(out)
}

func mustConvertBigInt(in string, base int) *big.Int {
	in = prepInt(base, in)
	out, ok := new(big.Int).SetString(in, base)
	if !ok {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

func mustConvertUint(in string, base int) uint64 {
	in = prepInt(base, in)
	out, err := strconv.ParseUint(in, base, 64)
	if err != nil {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

func LoadBlockTests(file string) (map[string]*BlockTest, error) {
	btjs := make(map[string]*btJSON)
	if err := readJsonFile(file, &btjs); err != nil {
		return nil, err
	}

	return convertBlockTests(btjs)
}

// Nothing to see here, please move along...
func prepInt(base int, s string) string {
	if base == 16 {
		if strings.HasPrefix(s, "0x") {
			s = s[2:]
		}
		if len(s) == 0 {
			s = "00"
		}
		s = nibbleFix(s)
	}
	return s
}

// don't ask
func unfuckFuckedHex(almostHex string) string {
	return nibbleFix(strings.Replace(almostHex, "v", "", -1))
}

func nibbleFix(s string) string {
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return s
}
