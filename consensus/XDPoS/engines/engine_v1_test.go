// Copyright 2021 XDC Network
// This file is part of the XDC library.

package engines

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// mockChain implements consensus.ChainHeaderReader for testing
type mockChain struct {
	headers map[common.Hash]*types.Header
}

func newMockChain() *mockChain {
	return &mockChain{
		headers: make(map[common.Hash]*types.Header),
	}
}

func (m *mockChain) Config() *params.ChainConfig {
	return &params.ChainConfig{}
}

func (m *mockChain) CurrentHeader() *types.Header {
	return nil
}

func (m *mockChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	return m.headers[hash]
}

func (m *mockChain) GetHeaderByNumber(number uint64) *types.Header {
	for _, h := range m.headers {
		if h.Number.Uint64() == number {
			return h
		}
	}
	return nil
}

func (m *mockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return m.headers[hash]
}

func (m *mockChain) GetTd(hash common.Hash, number uint64) *big.Int {
	return big.NewInt(1)
}

func (m *mockChain) addHeader(header *types.Header) {
	m.headers[header.Hash()] = header
}

// mockDB implements Database for testing
type mockDB struct {
	data map[string][]byte
}

func newMockDB() *mockDB {
	return &mockDB{
		data: make(map[string][]byte),
	}
}

func (m *mockDB) Get(key []byte) ([]byte, error) {
	return m.data[string(key)], nil
}

func (m *mockDB) Put(key []byte, value []byte) error {
	m.data[string(key)] = value
	return nil
}

func (m *mockDB) Delete(key []byte) error {
	delete(m.data, string(key))
	return nil
}

func (m *mockDB) Has(key []byte) (bool, error) {
	_, ok := m.data[string(key)]
	return ok, nil
}

// createTestHeader creates a test header
func createTestHeader(parent *types.Header, signer common.Address) *types.Header {
	number := big.NewInt(1)
	parentHash := common.Hash{}
	timestamp := uint64(time.Now().Unix())

	if parent != nil {
		number = new(big.Int).Add(parent.Number, big.NewInt(1))
		parentHash = parent.Hash()
		timestamp = parent.Time + 2
	}

	header := &types.Header{
		ParentHash: parentHash,
		UncleHash:  types.EmptyUncleHash,
		Coinbase:   signer,
		Root:       common.Hash{},
		TxHash:     types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:      types.Bloom{},
		Difficulty: big.NewInt(1),
		Number:     number,
		GasLimit:   8000000,
		GasUsed:    0,
		Time:       timestamp,
		Extra:      make([]byte, ExtraVanity+crypto.SignatureLength),
		MixDigest:  common.Hash{},
		Nonce:      types.BlockNonce{},
	}

	return header
}

func TestNewEngineV1(t *testing.T) {
	config := &params.XDPoSConfig{
		Period: 2,
		Epoch:  900,
	}
	db := newMockDB()

	engine := NewEngineV1(config, db)

	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	if engine.period != 2 {
		t.Errorf("Expected period 2, got %d", engine.period)
	}
}

func TestEngineV1_Author(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	// Create a test key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create and sign a header
	header := createTestHeader(nil, signer)

	// Sign the header
	sig, err := crypto.Sign(sigHash(header).Bytes(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	copy(header.Extra[ExtraVanity:], sig)

	// Test Author
	author, err := engine.Author(header)
	if err != nil {
		t.Fatal(err)
	}

	if author != signer {
		t.Errorf("Expected author %s, got %s", signer.Hex(), author.Hex())
	}
}

func TestEngineV1_CalcDifficulty(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	chain := newMockChain()
	parent := createTestHeader(nil, common.Address{})

	difficulty := engine.CalcDifficulty(chain, uint64(time.Now().Unix()), parent)

	if difficulty.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Expected difficulty 1, got %s", difficulty.String())
	}
}

func TestEngineV1_Prepare(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	// Set up signer
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := crypto.PubkeyToAddress(privateKey.PublicKey)
	engine.Authorize(signer, func(account common.Address, data []byte) ([]byte, error) {
		return crypto.Sign(data, privateKey)
	})

	// Create chain with parent
	chain := newMockChain()
	parent := createTestHeader(nil, signer)
	chain.addHeader(parent)

	// Create child header
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, big.NewInt(1)),
	}

	// Prepare
	err = engine.Prepare(chain, header)
	if err != nil {
		t.Fatal(err)
	}

	// Check header was prepared correctly
	if header.Coinbase != signer {
		t.Errorf("Expected coinbase %s, got %s", signer.Hex(), header.Coinbase.Hex())
	}

	if len(header.Extra) != ExtraVanity+crypto.SignatureLength {
		t.Errorf("Expected extra length %d, got %d", ExtraVanity+crypto.SignatureLength, len(header.Extra))
	}
}

func TestEngineV1_VerifyUncles(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	// Block without uncles should be valid
	block := types.NewBlockWithHeader(&types.Header{})
	err := engine.VerifyUncles(nil, block)
	if err != nil {
		t.Errorf("Expected no error for block without uncles, got %v", err)
	}
}

func TestEngineV1_SealHash(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	header := createTestHeader(nil, common.Address{})

	hash := engine.SealHash(header)

	if hash == (common.Hash{}) {
		t.Error("Expected non-zero seal hash")
	}
}

func TestEngineV1_Authorize(t *testing.T) {
	config := &params.XDPoSConfig{Period: 2}
	db := newMockDB()
	engine := NewEngineV1(config, db)

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := crypto.PubkeyToAddress(privateKey.PublicKey)

	signFn := func(account common.Address, data []byte) ([]byte, error) {
		return crypto.Sign(data, privateKey)
	}

	engine.Authorize(signer, signFn)

	engine.lock.RLock()
	defer engine.lock.RUnlock()

	if engine.signer != signer {
		t.Errorf("Expected signer %s, got %s", signer.Hex(), engine.signer.Hex())
	}

	if engine.signFn == nil {
		t.Error("Expected signFn to be set")
	}
}

func TestSigHash(t *testing.T) {
	header := createTestHeader(nil, common.Address{})

	hash1 := sigHash(header)
	hash2 := sigHash(header)

	if hash1 != hash2 {
		t.Error("Expected consistent hash")
	}

	// Modify header
	header.GasLimit = 9000000
	hash3 := sigHash(header)

	if hash1 == hash3 {
		t.Error("Expected different hash after modification")
	}
}

func TestEcrecover(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := crypto.PubkeyToAddress(privateKey.PublicKey)

	header := createTestHeader(nil, signer)

	// Sign the header
	sig, err := crypto.Sign(sigHash(header).Bytes(), privateKey)
	if err != nil {
		t.Fatal(err)
	}
	copy(header.Extra[ExtraVanity:], sig)

	// Recover
	cache := newLRU(10)
	recovered, err := ecrecover(header, cache)
	if err != nil {
		t.Fatal(err)
	}

	if recovered != signer {
		t.Errorf("Expected %s, got %s", signer.Hex(), recovered.Hex())
	}

	// Check cache
	if _, ok := cache.Get(header.Hash()); !ok {
		t.Error("Expected address to be cached")
	}
}

func TestLRU(t *testing.T) {
	cache := newLRU(10)

	key := common.HexToHash("0x1234")
	value := "test"

	// Test Get on empty cache
	_, ok := cache.Get(key)
	if ok {
		t.Error("Expected cache miss")
	}

	// Test Add and Get
	cache.Add(key, value)
	got, ok := cache.Get(key)
	if !ok {
		t.Error("Expected cache hit")
	}
	if got != value {
		t.Errorf("Expected %s, got %s", value, got)
	}
}
