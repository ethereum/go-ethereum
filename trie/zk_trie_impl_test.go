package trie

import (
	"math/big"
	"testing"

	"github.com/iden3/go-iden3-crypto/constants"
	cryptoUtils "github.com/iden3/go-iden3-crypto/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	zktrie "github.com/scroll-tech/zktrie/trie"
	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

// we do not need zktrie impl anymore, only made a wrapper for adapting testing
type zkTrieImplTestWrapper struct {
	*zktrie.ZkTrieImpl
}

func newZkTrieImpl(storage *ZktrieDatabase, maxLevels int) (*zkTrieImplTestWrapper, error) {
	return newZkTrieImplWithRoot(storage, &zkt.HashZero, maxLevels)
}

// NewZkTrieImplWithRoot loads a new ZkTrieImpl. If in the storage already exists one
// will open that one, if not, will create a new one.
func newZkTrieImplWithRoot(storage *ZktrieDatabase, root *zkt.Hash, maxLevels int) (*zkTrieImplTestWrapper, error) {
	impl, err := zktrie.NewZkTrieImplWithRoot(storage, root, maxLevels)
	if err != nil {
		return nil, err
	}

	return &zkTrieImplTestWrapper{impl}, nil
}

// AddWord
// Deprecated: Add a Bytes32 kv to ZkTrieImpl, only for testing
func (mt *zkTrieImplTestWrapper) AddWord(kPreimage, vPreimage *zkt.Byte32) error {

	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}

	if v, _ := mt.TryGet(k.Bytes()); v != nil {
		return zktrie.ErrEntryIndexAlreadyExists
	}

	return mt.ZkTrieImpl.TryUpdate(zkt.NewHashFromBigInt(k), 1, []zkt.Byte32{*vPreimage})
}

// GetLeafNodeByWord
// Deprecated: Get a Bytes32 kv to ZkTrieImpl, only for testing
func (mt *zkTrieImplTestWrapper) GetLeafNodeByWord(kPreimage *zkt.Byte32) (*zktrie.Node, error) {
	k, err := kPreimage.Hash()
	if err != nil {
		return nil, err
	}
	return mt.ZkTrieImpl.GetLeafNode(zkt.NewHashFromBigInt(k))
}

// Deprecated: only for testing
func (mt *zkTrieImplTestWrapper) UpdateWord(kPreimage, vPreimage *zkt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}

	return mt.ZkTrieImpl.TryUpdate(zkt.NewHashFromBigInt(k), 1, []zkt.Byte32{*vPreimage})
}

// Deprecated: only for testing
func (mt *zkTrieImplTestWrapper) DeleteWord(kPreimage *zkt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	return mt.ZkTrieImpl.TryDelete(zkt.NewHashFromBigInt(k))
}

func (mt *zkTrieImplTestWrapper) TryGet(key []byte) ([]byte, error) {
	return mt.ZkTrieImpl.TryGet(zkt.NewHashFromBytes(key))
}

func (mt *zkTrieImplTestWrapper) TryDelete(key []byte) error {
	return mt.ZkTrieImpl.TryDelete(zkt.NewHashFromBytes(key))
}

// TryUpdateAccount will abstract the write of an account to the trie
func (mt *zkTrieImplTestWrapper) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	value, flag := acc.MarshalFields()
	return mt.ZkTrieImpl.TryUpdate(zkt.NewHashFromBytes(key), flag, value)
}

// NewHashFromHex returns a *Hash representation of the given hex string
func NewHashFromHex(h string) (*zkt.Hash, error) {
	return zkt.NewHashFromCheckedBytes(common.FromHex(h))
}

type Fatalable interface {
	Fatal(args ...interface{})
}

func newTestingMerkle(f Fatalable, numLevels int) *zkTrieImplTestWrapper {
	mt, err := newZkTrieImpl(NewZktrieDatabase((memorydb.New())), numLevels)
	if err != nil {
		f.Fatal(err)
		return nil
	}
	return mt
}

func TestHashParsers(t *testing.T) {
	h0 := zkt.NewHashFromBigInt(big.NewInt(0))
	assert.Equal(t, "0", h0.String())
	h1 := zkt.NewHashFromBigInt(big.NewInt(1))
	assert.Equal(t, "1", h1.String())
	h10 := zkt.NewHashFromBigInt(big.NewInt(10))
	assert.Equal(t, "10", h10.String())

	h7l := zkt.NewHashFromBigInt(big.NewInt(1234567))
	assert.Equal(t, "1234567", h7l.String())
	h8l := zkt.NewHashFromBigInt(big.NewInt(12345678))
	assert.Equal(t, "12345678...", h8l.String())

	b, ok := new(big.Int).SetString("4932297968297298434239270129193057052722409868268166443802652458940273154854", 10) //nolint:lll
	assert.True(t, ok)
	h := zkt.NewHashFromBigInt(b)
	assert.Equal(t, "4932297968297298434239270129193057052722409868268166443802652458940273154854", h.BigInt().String()) //nolint:lll
	assert.Equal(t, "49322979...", h.String())
	assert.Equal(t, "0ae794eb9c3d8bbb9002e993fc2ed301dcbd2af5508ed072c375e861f1aa5b26", h.Hex())

	b1, err := zkt.NewBigIntFromHashBytes(b.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, new(big.Int).SetBytes(b.Bytes()).String(), b1.String())

	b2, err := zkt.NewHashFromCheckedBytes(b.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, b.String(), b2.BigInt().String())

	h2, err := NewHashFromHex(h.Hex())
	assert.Nil(t, err)
	assert.Equal(t, h, h2)
	_, err = NewHashFromHex("0x12")
	assert.NotNil(t, err)

	// check limits
	a := new(big.Int).Sub(constants.Q, big.NewInt(1))
	testHashParsers(t, a)
	a = big.NewInt(int64(1))
	testHashParsers(t, a)
}

func testHashParsers(t *testing.T, a *big.Int) {
	require.True(t, cryptoUtils.CheckBigIntInField(a))
	h := zkt.NewHashFromBigInt(a)
	assert.Equal(t, a, h.BigInt())
	hFromBytes, err := zkt.NewHashFromCheckedBytes(h.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, h, hFromBytes)
	assert.Equal(t, a, hFromBytes.BigInt())
	assert.Equal(t, a.String(), hFromBytes.BigInt().String())
	hFromHex, err := NewHashFromHex(h.Hex())
	assert.Nil(t, err)
	assert.Equal(t, h, hFromHex)

	aBIFromHBytes, err := zkt.NewBigIntFromHashBytes(h.Bytes())
	assert.Nil(t, err)
	assert.Equal(t, a, aBIFromHBytes)
	assert.Equal(t, new(big.Int).SetBytes(a.Bytes()).String(), aBIFromHBytes.String())
}

func TestMerkleTree_AddUpdateGetWord(t *testing.T) {
	mt := newTestingMerkle(t, 10)
	err := mt.AddWord(&zkt.Byte32{1}, &zkt.Byte32{2})
	assert.Nil(t, err)
	err = mt.AddWord(&zkt.Byte32{3}, &zkt.Byte32{4})
	assert.Nil(t, err)
	err = mt.AddWord(&zkt.Byte32{5}, &zkt.Byte32{6})
	assert.Nil(t, err)
	err = mt.AddWord(&zkt.Byte32{5}, &zkt.Byte32{7})
	assert.Equal(t, zktrie.ErrEntryIndexAlreadyExists, err)

	node, err := mt.GetLeafNodeByWord(&zkt.Byte32{1})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{2})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNodeByWord(&zkt.Byte32{3})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{4})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNodeByWord(&zkt.Byte32{5})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{6})[:], node.ValuePreimage[0][:])

	err = mt.UpdateWord(&zkt.Byte32{1}, &zkt.Byte32{7})
	assert.Nil(t, err)
	err = mt.UpdateWord(&zkt.Byte32{3}, &zkt.Byte32{8})
	assert.Nil(t, err)
	err = mt.UpdateWord(&zkt.Byte32{5}, &zkt.Byte32{9})
	assert.Nil(t, err)

	node, err = mt.GetLeafNodeByWord(&zkt.Byte32{1})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{7})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNodeByWord(&zkt.Byte32{3})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{8})[:], node.ValuePreimage[0][:])
	node, err = mt.GetLeafNodeByWord(&zkt.Byte32{5})
	assert.Nil(t, err)
	assert.Equal(t, len(node.ValuePreimage), 1)
	assert.Equal(t, (&zkt.Byte32{9})[:], node.ValuePreimage[0][:])
	_, err = mt.GetLeafNodeByWord(&zkt.Byte32{100})
	assert.Equal(t, zktrie.ErrKeyNotFound, err)
}

func TestMerkleTree_UpdateAccount(t *testing.T) {

	mt := newTestingMerkle(t, 10)

	acc1 := &types.StateAccount{
		Nonce:            1,
		Balance:          big.NewInt(10000000),
		Root:             common.HexToHash("22fb59aa5410ed465267023713ab42554c250f394901455a3366e223d5f7d147"),
		KeccakCodeHash:   common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		PoseidonCodeHash: common.HexToHash("0c0a77f6e063b4b62eb7d9ed6f427cf687d8d0071d751850cfe5d136bc60d3ab").Bytes(),
		CodeSize:         0,
	}

	err := mt.TryUpdateAccount(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes(), acc1)
	assert.Nil(t, err)

	acc2 := &types.StateAccount{
		Nonce:            5,
		Balance:          big.NewInt(50000000),
		Root:             common.HexToHash("0"),
		KeccakCodeHash:   common.HexToHash("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		PoseidonCodeHash: common.HexToHash("05d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes(),
		CodeSize:         5,
	}
	err = mt.TryUpdateAccount(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes(), acc2)
	assert.Nil(t, err)

	bt, err := mt.TryGet(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)

	acc, err := types.UnmarshalStateAccount(bt)
	assert.Nil(t, err)
	assert.Equal(t, acc1.Nonce, acc.Nonce)
	assert.Equal(t, acc1.Balance.Uint64(), acc.Balance.Uint64())
	assert.Equal(t, acc1.Root.Bytes(), acc.Root.Bytes())
	assert.Equal(t, acc1.KeccakCodeHash, acc.KeccakCodeHash)
	assert.Equal(t, acc1.PoseidonCodeHash, acc.PoseidonCodeHash)
	assert.Equal(t, acc1.CodeSize, acc.CodeSize)

	bt, err = mt.TryGet(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)

	acc, err = types.UnmarshalStateAccount(bt)
	assert.Nil(t, err)
	assert.Equal(t, acc2.Nonce, acc.Nonce)
	assert.Equal(t, acc2.Balance.Uint64(), acc.Balance.Uint64())
	assert.Equal(t, acc2.Root.Bytes(), acc.Root.Bytes())
	assert.Equal(t, acc2.KeccakCodeHash, acc.KeccakCodeHash)
	assert.Equal(t, acc2.PoseidonCodeHash, acc.PoseidonCodeHash)
	assert.Equal(t, acc2.CodeSize, acc.CodeSize)

	bt, err = mt.TryGet(common.HexToAddress("0x8dE13967F19410A7991D63c2c0179feBFDA0c261").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)

	err = mt.TryDelete(common.HexToHash("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)

	bt, err = mt.TryGet(common.HexToAddress("0x05fDbDfaE180345C6Cff5316c286727CF1a43327").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)

	err = mt.TryDelete(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)

	bt, err = mt.TryGet(common.HexToAddress("0x4cb1aB63aF5D8931Ce09673EbD8ae2ce16fD6571").Bytes())
	assert.Nil(t, err)
	assert.Nil(t, bt)
}
