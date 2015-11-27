// Copyright 2015 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
)

// Block Test JSON Format
type BlockTest struct {
	Genesis *types.Block

	Json          *btJSON
	preAccounts   map[string]btAccount
	postAccounts  map[string]btAccount
	lastblockhash string
}

type btJSON struct {
	Blocks             []btBlock
	GenesisBlockHeader btHeader
	Pre                map[string]btAccount
	PostState          map[string]btAccount
	Lastblockhash      string
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
	Hash             string
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
			return fmt.Errorf("%s: %v", name, err)
		}
		glog.Infoln("Block test passed: ", name)

	}
	return nil

}
func runBlockTest(test *BlockTest) error {
	ks := crypto.NewKeyStorePassphrase(filepath.Join(common.DefaultDataDir(), "keystore"), crypto.StandardScryptN, crypto.StandardScryptP)
	am := accounts.NewManager(ks)
	db, _ := ethdb.NewMemDatabase()

	// import pre accounts & construct test genesis block & state root
	_, err := test.InsertPreState(db, am)
	if err != nil {
		return fmt.Errorf("InsertPreState: %v", err)
	}

	cfg := &eth.Config{
		TestGenesisState: db,
		TestGenesisBlock: test.Genesis,
		Etherbase:        common.Address{},
		AccountManager:   am,
	}
	ethereum, err := eth.New(&node.ServiceContext{EventMux: new(event.TypeMux)}, cfg)
	if err != nil {
		return err
	}
	cm := ethereum.BlockChain()
	validBlocks, err := test.TryBlocksInsert(cm)
	if err != nil {
		return err
	}

	lastblockhash := common.HexToHash(test.lastblockhash)
	cmlast := cm.LastBlockHash()
	if lastblockhash != cmlast {
		return fmt.Errorf("lastblockhash validation mismatch: want: %x, have: %x", lastblockhash, cmlast)
	}

	newDB, err := cm.State()
	if err != nil {
		return err
	}
	if err = test.ValidatePostState(newDB); err != nil {
		return fmt.Errorf("post state validation failed: %v", err)
	}

	return test.ValidateImportedHeaders(cm, validBlocks)
}

// InsertPreState populates the given database with the genesis
// accounts defined by the test.
func (t *BlockTest) InsertPreState(db ethdb.Database, am *accounts.Manager) (*state.StateDB, error) {
	statedb, err := state.New(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
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
			err = am.TimedUnlock(common.BytesToAddress(addr), "", 999999*time.Second)
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

	root, err := statedb.Commit()
	if err != nil {
		return nil, fmt.Errorf("error writing state: %v", err)
	}
	if t.Genesis.Root() != root {
		return nil, fmt.Errorf("computed state root does not match genesis block: genesis=%x computed=%x", t.Genesis.Root().Bytes()[:4], root.Bytes()[:4])
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
func (t *BlockTest) TryBlocksInsert(blockchain *core.BlockChain) ([]btBlock, error) {
	validBlocks := make([]btBlock, 0)
	// insert the test blocks, which will execute all transactions
	for _, b := range t.Json.Blocks {
		cb, err := mustConvertBlock(b)
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return nil, fmt.Errorf("Block RLP decoding failed when expected to succeed: %v", err)
			}
		}
		// RLP decoding worked, try to insert into chain:
		_, err = blockchain.InsertChain(types.Blocks{cb})
		if err != nil {
			if b.BlockHeader == nil {
				continue // OK - block is supposed to be invalid, continue with next block
			} else {
				return nil, fmt.Errorf("Block insertion into chain failed: %v", err)
			}
		}
		if b.BlockHeader == nil {
			return nil, fmt.Errorf("Block insertion should have failed")
		}

		// validate RLP decoding by checking all values against test file JSON
		if err = validateHeader(b.BlockHeader, cb.Header()); err != nil {
			return nil, fmt.Errorf("Deserialised block header validation failed: %v", err)
		}
		validBlocks = append(validBlocks, b)
	}
	return validBlocks, nil
}

func validateHeader(h *btHeader, h2 *types.Header) error {
	expectedBloom := mustConvertBytes(h.Bloom)
	if !bytes.Equal(expectedBloom, h2.Bloom.Bytes()) {
		return fmt.Errorf("Bloom: want: %x have: %x", expectedBloom, h2.Bloom.Bytes())
	}

	expectedCoinbase := mustConvertBytes(h.Coinbase)
	if !bytes.Equal(expectedCoinbase, h2.Coinbase.Bytes()) {
		return fmt.Errorf("Coinbase: want: %x have: %x", expectedCoinbase, h2.Coinbase.Bytes())
	}

	expectedMixHashBytes := mustConvertBytes(h.MixHash)
	if !bytes.Equal(expectedMixHashBytes, h2.MixDigest.Bytes()) {
		return fmt.Errorf("MixHash: want: %x have: %x", expectedMixHashBytes, h2.MixDigest.Bytes())
	}

	expectedNonce := mustConvertBytes(h.Nonce)
	if !bytes.Equal(expectedNonce, h2.Nonce[:]) {
		return fmt.Errorf("Nonce: want: %x have: %x", expectedNonce, h2.Nonce)
	}

	expectedNumber := mustConvertBigInt(h.Number, 16)
	if expectedNumber.Cmp(h2.Number) != 0 {
		return fmt.Errorf("Number: want: %v have: %v", expectedNumber, h2.Number)
	}

	expectedParentHash := mustConvertBytes(h.ParentHash)
	if !bytes.Equal(expectedParentHash, h2.ParentHash.Bytes()) {
		return fmt.Errorf("Parent hash: want: %x have: %x", expectedParentHash, h2.ParentHash.Bytes())
	}

	expectedReceiptHash := mustConvertBytes(h.ReceiptTrie)
	if !bytes.Equal(expectedReceiptHash, h2.ReceiptHash.Bytes()) {
		return fmt.Errorf("Receipt hash: want: %x have: %x", expectedReceiptHash, h2.ReceiptHash.Bytes())
	}

	expectedTxHash := mustConvertBytes(h.TransactionsTrie)
	if !bytes.Equal(expectedTxHash, h2.TxHash.Bytes()) {
		return fmt.Errorf("Tx hash: want: %x have: %x", expectedTxHash, h2.TxHash.Bytes())
	}

	expectedStateHash := mustConvertBytes(h.StateRoot)
	if !bytes.Equal(expectedStateHash, h2.Root.Bytes()) {
		return fmt.Errorf("State hash: want: %x have: %x", expectedStateHash, h2.Root.Bytes())
	}

	expectedUncleHash := mustConvertBytes(h.UncleHash)
	if !bytes.Equal(expectedUncleHash, h2.UncleHash.Bytes()) {
		return fmt.Errorf("Uncle hash: want: %x have: %x", expectedUncleHash, h2.UncleHash.Bytes())
	}

	expectedExtraData := mustConvertBytes(h.ExtraData)
	if !bytes.Equal(expectedExtraData, h2.Extra) {
		return fmt.Errorf("Extra data: want: %x have: %x", expectedExtraData, h2.Extra)
	}

	expectedDifficulty := mustConvertBigInt(h.Difficulty, 16)
	if expectedDifficulty.Cmp(h2.Difficulty) != 0 {
		return fmt.Errorf("Difficulty: want: %v have: %v", expectedDifficulty, h2.Difficulty)
	}

	expectedGasLimit := mustConvertBigInt(h.GasLimit, 16)
	if expectedGasLimit.Cmp(h2.GasLimit) != 0 {
		return fmt.Errorf("GasLimit: want: %v have: %v", expectedGasLimit, h2.GasLimit)
	}
	expectedGasUsed := mustConvertBigInt(h.GasUsed, 16)
	if expectedGasUsed.Cmp(h2.GasUsed) != 0 {
		return fmt.Errorf("GasUsed: want: %v have: %v", expectedGasUsed, h2.GasUsed)
	}

	expectedTimestamp := mustConvertBigInt(h.Timestamp, 16)
	if expectedTimestamp.Cmp(h2.Time) != 0 {
		return fmt.Errorf("Timestamp: want: %v have: %v", expectedTimestamp, h2.Time)
	}

	return nil
}

func (t *BlockTest) ValidatePostState(statedb *state.StateDB) error {
	// validate post state accounts in test file against what we have in state db
	for addrString, acct := range t.postAccounts {
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
			return fmt.Errorf("account code mismatch for addr: %s want: %s have: %s", addrString, hex.EncodeToString(code), hex.EncodeToString(code2))
		}
		if balance2.Cmp(balance) != 0 {
			return fmt.Errorf("account balance mismatch for addr: %s, want: %d, have: %d", addrString, balance, balance2)
		}
		if nonce2 != nonce {
			return fmt.Errorf("account nonce mismatch for addr: %s want: %d have: %d", addrString, nonce, nonce2)
		}
	}
	return nil
}

func (test *BlockTest) ValidateImportedHeaders(cm *core.BlockChain, validBlocks []btBlock) error {
	// to get constant lookup when verifying block headers by hash (some tests have many blocks)
	bmap := make(map[string]btBlock, len(test.Json.Blocks))
	for _, b := range validBlocks {
		bmap[b.BlockHeader.Hash] = b
	}

	// iterate over blocks backwards from HEAD and validate imported
	// headers vs test file. some tests have reorgs, and we import
	// block-by-block, so we can only validate imported headers after
	// all blocks have been processed by ChainManager, as they may not
	// be part of the longest chain until last block is imported.
	for b := cm.CurrentBlock(); b != nil && b.NumberU64() != 0; b = cm.GetBlock(b.Header().ParentHash) {
		bHash := common.Bytes2Hex(b.Hash().Bytes()) // hex without 0x prefix
		if err := validateHeader(bmap[bHash].BlockHeader, b.Header()); err != nil {
			return fmt.Errorf("Imported block header validation failed: %v", err)
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
	out = &BlockTest{preAccounts: in.Pre, postAccounts: in.PostState, Json: in, lastblockhash: in.Lastblockhash}
	out.Genesis = mustConvertGenesis(in.GenesisBlockHeader)
	return out, err
}

func mustConvertGenesis(testGenesis btHeader) *types.Block {
	hdr := mustConvertHeader(testGenesis)
	hdr.Number = big.NewInt(0)

	return types.NewBlockWithHeader(hdr)
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
		Time:        mustConvertBigInt(in.Timestamp, 16),
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
