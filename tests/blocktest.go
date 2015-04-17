package tests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Block Test JSON Format
type btJSON struct {
	Blocks             []btBlock
	GenesisBlockHeader btHeader
	Pre                map[string]btAccount
	PostState          map[string]btAccount
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
	UncleHeaders []*btHeader
}

type BlockTest struct {
	Genesis *types.Block
	Blocks  []*types.Block

	preAccounts map[string]btAccount
}

// LoadBlockTests loads a block test JSON file.
func LoadBlockTests(file string) (map[string]*BlockTest, error) {
	bt := make(map[string]*btJSON)
	if err := LoadJSON(file, &bt); err != nil {
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
func (t *BlockTest) InsertPreState(db common.Database) (*state.StateDB, error) {
	statedb := state.New(common.Hash{}, db)
	for addrString, acct := range t.preAccounts {
		// XXX: is is worth it checking for errors here?
		//addr, _ := hex.DecodeString(addrString)
		code, _ := hex.DecodeString(strings.TrimPrefix(acct.Code, "0x"))
		balance, _ := new(big.Int).SetString(acct.Balance, 0)
		nonce, _ := strconv.ParseUint(acct.Nonce, 16, 64)

		obj := statedb.CreateAccount(common.HexToAddress(addrString))
		obj.SetCode(code)
		obj.SetBalance(balance)
		obj.SetNonce(nonce)
		for k, v := range acct.Storage {
			statedb.SetState(common.HexToAddress(addrString), common.HexToHash(k), common.FromHex(v))
		}
	}
	// sync objects to trie
	statedb.Update()
	// sync trie to disk
	statedb.Sync()

	if !bytes.Equal(t.Genesis.Root().Bytes(), statedb.Root().Bytes()) {
		return nil, fmt.Errorf("computed state root does not match genesis block %x %x", t.Genesis.Root().Bytes()[:4], statedb.Root().Bytes()[:4])
	}
	return statedb, nil
}

func (t *BlockTest) ValidatePostState(statedb *state.StateDB) error {
	for addrString, acct := range t.preAccounts {
		// XXX: is is worth it checking for errors here?
		addr, _ := hex.DecodeString(addrString)
		code, _ := hex.DecodeString(strings.TrimPrefix(acct.Code, "0x"))
		balance, _ := new(big.Int).SetString(acct.Balance, 0)
		nonce, _ := strconv.ParseUint(acct.Nonce, 16, 64)

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
		GasUsed:     mustConvertBigInt(in.GasUsed),
		GasLimit:    mustConvertBigInt(in.GasLimit),
		Difficulty:  mustConvertBigInt(in.Difficulty),
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
	if in == "0x" {
		return []byte{}
	}
	h := strings.TrimPrefix(unfuckCPPHexInts(in), "0x")
	out, err := hex.DecodeString(h)
	if err != nil {
		panic(fmt.Errorf("invalid hex: %q", h))
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

func mustConvertBigInt(in string) *big.Int {
	out, ok := new(big.Int).SetString(unfuckCPPHexInts(in), 0)
	if !ok {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

func mustConvertUint(in string) uint64 {
	out, err := strconv.ParseUint(unfuckCPPHexInts(in), 0, 64)
	if err != nil {
		panic(fmt.Errorf("invalid integer: %q", in))
	}
	return out
}

// LoadJSON reads the given file and unmarshals its content.
func LoadJSON(file string, val interface{}) error {
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

func unfuckCPPHexInts(s string) string {
	if s == "0x" { // no respect for the empty value :(
		return "0x00"
	}
	if (len(s) % 2) != 0 { // motherfucking nibbles
		return "0x0" + s[2:]
	}
	return s
}
