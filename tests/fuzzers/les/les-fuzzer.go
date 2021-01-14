package les

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	l "github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	//fuzz "github.com/google/gofuzz"
)

var (
	bankKey, _ = crypto.GenerateKey()
	bankAddr   = crypto.PubkeyToAddress(bankKey.PublicKey)
	bankFunds  = big.NewInt(1000000000000000000)

	userKey1, _ = crypto.GenerateKey()
	userKey2, _ = crypto.GenerateKey()
	userAddr1   = crypto.PubkeyToAddress(userKey1.PublicKey)
	userAddr2   = crypto.PubkeyToAddress(userKey2.PublicKey)

	testContractAddr common.Address
	testContractCode = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
)

func makeChain(n int) (bc *core.BlockChain, addrHashes, txHashes []common.Hash) {
	db := rawdb.NewMemoryDatabase()
	gspec := core.Genesis{
		Config:   params.TestChainConfig,
		Alloc:    core.GenesisAlloc{bankAddr: {Balance: bankFunds}},
		GasLimit: 100000000,
	}
	genesis := gspec.MustCommit(db)
	signer := types.HomesteadSigner{}
	blocks, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, n,
		func(i int, gen *core.BlockGen) {
			var (
				tx   *types.Transaction
				addr common.Address
			)
			nonce := uint64(i)
			if i%16 == 0 {
				tx, _ = types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), 200000, big.NewInt(0), testContractCode), signer, bankKey)
				addr = crypto.CreateAddress(userAddr1, nonce)
			} else {
				key, _ := crypto.GenerateKey()
				addr = crypto.PubkeyToAddress(key.PublicKey)
				tx, _ = types.SignTx(types.NewTransaction(nonce, addr, big.NewInt(10000), params.TxGas, big.NewInt(100000000000), nil), signer, bankKey)
			}
			gen.AddTx(tx)
			addrHashes = append(addrHashes, crypto.Keccak256Hash(addr[:]))
			txHashes = append(txHashes, tx.Hash())
		})
	bc, _ = core.NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{}, nil, nil)

	if _, err := bc.InsertChain(blocks); err != nil {
		panic(err)
	}
	return
}

type fuzzer struct {
	chain     *core.BlockChain
	pool      *core.TxPool
	trie      *trie.Trie
	addr, txs []common.Hash

	input     io.Reader
	exhausted bool
}

func newFuzzer(input []byte) *fuzzer {
	f := &fuzzer{
		input: bytes.NewReader(input),
	}
	f.chain, f.addr, f.txs = makeChain(int(f.randomByte()))
	f.pool = core.NewTxPool(core.DefaultTxPoolConfig, params.TestChainConfig, f.chain)
	f.trie, _ = trie.New(common.Hash{}, trie.NewDatabase(rawdb.NewMemoryDatabase()))
	r := rand.New(rand.NewSource(42))
	for i := 0; i < 100; i++ {
		key := make([]byte, r.Intn(32)+1)
		r.Read(key)
		value := make([]byte, r.Intn(64))
		r.Read(value)
		f.trie.Update(key, value)
	}
	return f
}

func (f *fuzzer) read(size int) []byte {
	out := make([]byte, size)
	if _, err := f.input.Read(out); err != nil {
		f.exhausted = true
	}
	return out
}

func (f *fuzzer) randomByte() byte {
	d := f.read(1)
	return d[0]
}

func (f *fuzzer) randomBool() bool {
	d := f.read(1)
	return d[0]&1 == 1
}

func (f *fuzzer) randomInt(max int) int {
	if max == 0 {
		return 0
	}
	if max <= 256 {
		return int(f.randomByte()) % max
	}
	var a uint16
	if err := binary.Read(f.input, binary.LittleEndian, &a); err != nil {
		f.exhausted = true
	}
	return int(a % uint16(max))
}

func (f *fuzzer) randomX(max int) uint64 {
	var a uint16
	if err := binary.Read(f.input, binary.LittleEndian, &a); err != nil {
		f.exhausted = true
	}
	if a < 0x8000 {
		return uint64(a%uint16(max+1)) - 1
	}
	return (uint64(1)<<(a%64+1) - 1) & (uint64(a) * 343897772345826595)
}

func (f *fuzzer) randomBlockHash() common.Hash {
	b := f.randomByte()
	h := f.chain.GetCanonicalHash(uint64(b))
	if h == (common.Hash{}) {
		if b&1 == 1 {
			h[0] = b
		}
	}
	return h
}

func (f *fuzzer) randomAddrKey() []byte {
	i := int(f.randomByte())
	if i < len(f.addr) {
		return f.addr[i].Bytes()
	} else {
		h := make([]byte, int(f.randomByte()))
		if i < len(h) {
			h[i] = byte(i) + 1
		}
		return h
	}
}

func (f *fuzzer) randomTxHash() common.Hash {
	i := int(f.randomByte())
	if i < len(f.txs) {
		return f.txs[i]
	} else {
		var h common.Hash
		if i&1 == 1 {
			h[0] = byte(i)
		}
		return h
	}
}

func (f *fuzzer) randomBytes(maxLen int) []byte {
	return f.read(f.randomInt(maxLen + 1))
}

func (f *fuzzer) BlockChain() *core.BlockChain {
	return f.chain
}

func (f *fuzzer) TxPool() *core.TxPool {
	return f.pool
}

func (f *fuzzer) ArchiveMode() bool {
	return false
}

func (f *fuzzer) AddTxsSync() bool {
	return false
}

func (f *fuzzer) GetHelperTrie(typ uint, index uint64) *trie.Trie {
	if index < 2 {
		return f.trie
	}
	return nil
}

func (f *fuzzer) doFuzz(req l.HandlerRequest) {
	defer f.chain.Stop()
	//fuzz.NewFromGoFuzz(input).Fuzz(req)
	peer := l.NewFuzzerPeer(3)
	req.Serve(f, 42, peer, func() bool { return true })
}

func Fuzz(input []byte) int {
	f := newFuzzer(input)
	if f.exhausted {
		return -1
	}
	for !f.exhausted {
		switch f.randomInt(8) {
		case 0:
			req := &l.GetBlockHeadersReq{
				Amount:  f.randomX(l.MaxHeaderFetch + 1),
				Skip:    f.randomX(10),
				Reverse: f.randomBool(),
			}
			if f.randomBool() {
				req.Origin.Hash = f.randomBlockHash()
			} else {
				req.Origin.Number = uint64(f.randomByte())
			}
			f.doFuzz(req)

		case 1:
			req := make(l.GetBlockBodiesReq, f.randomInt(l.MaxBodyFetch+1))
			for i := range req {
				req[i] = f.randomBlockHash()
			}
			f.doFuzz(req)

		case 2:
			req := make(l.GetCodeReq, f.randomInt(l.MaxCodeFetch+1))
			for i := range req {
				req[i] = l.CodeReq{
					BHash:  f.randomBlockHash(),
					AccKey: f.randomAddrKey(),
				}
			}
			f.doFuzz(req)

		case 3:
			req := make(l.GetReceiptsReq, f.randomInt(l.MaxReceiptFetch+1))
			for i := range req {
				req[i] = f.randomBlockHash()
			}
			f.doFuzz(req)

		case 4:
			req := make(l.GetProofsReq, f.randomInt(l.MaxProofsFetch+1))
			for i := range req {
				req[i] = l.ProofReq{
					BHash:     f.randomBlockHash(),
					AccKey:    f.randomAddrKey(),
					Key:       f.randomAddrKey(),
					FromLevel: uint(f.randomX(3)),
				}
			}
			f.doFuzz(req)

		case 5:
			req := make(l.GetHelperTrieProofsReq, f.randomInt(l.MaxHelperTrieProofsFetch+1))
			for i := range req {
				req[i] = l.HelperTrieReq{
					Type:      uint(f.randomX(3)),
					TrieIdx:   f.randomX(3),
					Key:       f.randomAddrKey(),
					FromLevel: uint(f.randomX(3)),
					AuxReq:    uint(f.randomX(3)),
				}
			}
			f.doFuzz(req)

		case 6:
			req := make(l.SendTxReq, f.randomInt(l.MaxTxSend+1))
			signer := types.HomesteadSigner{}
			for i := range req {
				nonce := uint64(f.randomByte())
				if nonce%1 == 0 {
					nonce = uint64(len(f.txs))
				}
				req[i], _ = types.SignTx(types.NewTransaction(nonce, common.Address{}, big.NewInt(10000), params.TxGas, big.NewInt(1000000000*int64(f.randomByte())), nil), signer, bankKey)
			}
			f.doFuzz(req)

		case 7:
			req := make(l.GetTxStatusReq, f.randomInt(l.MaxTxStatus+1))
			for i := range req {
				req[i] = f.randomTxHash()
			}
			f.doFuzz(req)
		}
	}
	return 0
}
