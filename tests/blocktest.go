package tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
)

// Block Test JSON Format

type btJSON struct {
	Blocks             []btBlock
	GenesisBlockHeader btHeader
	Pre                map[string]btAccount
}

type btAccount struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
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

type btBlock struct {
	BlockHeader  *btHeader
	Rlp          string
	Transactions []btTransaction
	UncleHeaders []string
}

type BlockTest struct {
	Genesis *types.Block
	Blocks  []*types.Block

	preAccounts map[string]btAccount
}

// LoadBlockTests loads a block test JSON file.
func LoadBlockTests(file string) (map[string]*BlockTest, error) {
	bt := make(map[string]*btJSON)
	if err := loadJSON(file, &bt); err != nil {
		return nil, err
	}
	out := make(map[string]*BlockTest)
	for name, in := range bt {
		var err error
		if out[name], err = convertTest(in); err != nil {
			return nil, fmt.Errorf("bad test %q: %v", err)
		}
	}
	return out, nil
}

// InsertPreState populates the given database with the genesis
// accounts defined by the test.
func (t *BlockTest) InsertPreState(db common.Database) error {
	statedb := state.New(nil, db)
	for addrString, acct := range t.preAccounts {
		// XXX: is is worth it checking for errors here?
		//addr, _ := hex.DecodeString(addrString)
		code, _ := hex.DecodeString(strings.TrimPrefix(acct.Code, "0x"))
		balance, _ := new(big.Int).SetString(acct.Balance, 0)
		nonce, _ := strconv.ParseUint(acct.Nonce, 16, 64)

		obj := statedb.NewStateObject(common.HexToAddress(addrString))
		obj.SetCode(code)
		obj.SetBalance(balance)
		obj.SetNonce(nonce)
		// for k, v := range acct.Storage {
		// 	obj.SetState(k, v)
		// }
	}
	// sync objects to trie
	statedb.Update(nil)
	// sync trie to disk
	statedb.Sync()

	if !bytes.Equal(t.Genesis.Root().Bytes(), statedb.Root()) {
		return errors.New("computed state root does not match genesis block")
	}
	return nil
}

func convertTest(in *btJSON) (out *BlockTest, err error) {
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
	out = &BlockTest{preAccounts: in.Pre}
	out.Genesis = mustConvertGenesis(in.GenesisBlockHeader)
	out.Blocks = mustConvertBlocks(in.Blocks)
	return out, err
}

func mustConvertGenesis(testGenesis btHeader) *types.Block {
	hdr := mustConvertHeader(testGenesis)
	hdr.Number = big.NewInt(0)
	b := types.NewBlockWithHeader(hdr)
	b.Td = new(big.Int)
	b.Reward = new(big.Int)
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
		Extra:       string(mustConvertBytes(in.ExtraData)),
		GasUsed:     mustConvertBigInt10(in.GasUsed),
		GasLimit:    mustConvertBigInt10(in.GasLimit),
		Difficulty:  mustConvertBigInt10(in.Difficulty),
		Time:        mustConvertUint(in.Timestamp),
	}
	// XXX cheats? :-)
	header.SetNonce(common.BytesToHash(mustConvertBytes(in.Nonce)).Big().Uint64())
	return header
}

func mustConvertBlocks(testBlocks []btBlock) []*types.Block {
	var out []*types.Block
	for i, inb := range testBlocks {
		var b types.Block
		r := bytes.NewReader(mustConvertBytes(inb.Rlp))
		if err := rlp.Decode(r, &b); err != nil {
			panic(fmt.Errorf("invalid block %d: %q", i, inb.Rlp))
		}
		out = append(out, &b)
	}
	return out
}

func mustConvertBytes(in string) []byte {
	out, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", in))
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

func mustConvertBloom(in string) core.Bloom {
	out, err := hex.DecodeString(strings.TrimPrefix(in, "0x"))
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", in))
	}
	return core.BytesToBloom(out)
}

func mustConvertBigInt10(in string) *big.Int {
	out, ok := new(big.Int).SetString(in, 10)
	if !ok {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

func mustConvertUint(in string) uint64 {
	out, err := strconv.ParseUint(in, 0, 64)
	if err != nil {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

// loadJSON reads the given file and unmarshals its content.
func loadJSON(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}
