package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNibblesEncodeDecode(t *testing.T) {
	type fields struct {
		Nibbles []byte
	}
	type args struct {
		buf bytes.Buffer
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		encodeds string
	}{
		{
			name: "emptyNibbles",
			fields: fields{
				Nibbles: []byte{},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00",
		},
		{
			name: "singleNibble",
			fields: fields{
				Nibbles: []byte{10},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x1a",
		},
		{
			name: "evenNumberNibbles",
			fields: fields{
				Nibbles: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00123456789abc",
		},
		{
			name: "oddNumberNibbles",
			fields: fields{
				Nibbles: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13},
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x1123456789abcd",
		},
		{
			name: "maxNumberNibbles",
			fields: fields{
				Nibbles: initSlice(64, 10),
			},
			args: args{
				bytes.Buffer{},
			},
			encodeds: "0x00aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := FromUnpackedNibbles(tt.fields.Nibbles)
			assert.NoError(t, err)
			err = n.Serialize(codec.NewEncodingWriter(&tt.args.buf))
			assert.NoError(t, err)
			assert.Equal(t, tt.encodeds, hexutil.Encode(tt.args.buf.Bytes()))

			newNibble := &Nibbles{}
			err = newNibble.Deserialize(codec.NewDecodingReader(&tt.args.buf, uint64(len(tt.args.buf.Bytes()))))
			require.NoError(t, err)
			require.Equal(t, newNibble.Nibbles, n.Nibbles)
		})
	}
}

func TestFromUnpackedShouldFailForInvalidNibbles(t *testing.T) {
	type fields struct {
		Nibbles []byte
	}

	tests := []struct {
		name     string
		fields   fields
		encodeds string
	}{
		{
			name: "singleNibble",
			fields: fields{
				Nibbles: []byte{0x10},
			},
		},
		{
			name: "firstOutOfTwo",
			fields: fields{
				Nibbles: []byte{0x11, 0x01},
			},
		},
		{
			name: "secondOutOfTwo",
			fields: fields{
				Nibbles: []byte{0x01, 0x12},
			},
		},
		{
			name: "firstOutOfThree",
			fields: fields{
				Nibbles: []byte{0x11, 0x02, 0x03},
			},
		},
		{
			name: "secondOutOfThree",
			fields: fields{
				Nibbles: []byte{0x01, 0x12, 0x03},
			},
		},
		{
			name: "thirdOutOfThree",
			fields: fields{
				Nibbles: []byte{0x01, 0x02, 0x13},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromUnpackedNibbles(tt.fields.Nibbles)
			assert.Error(t, err)
		})
	}
}

func TestDecodeShouldFailForInvalidBytes(t *testing.T) {
	type fields struct {
		Nibbles string
	}

	tests := []struct {
		name     string
		fields   fields
		encodeds string
	}{
		{
			name: "empty",
			fields: fields{
				Nibbles: "0x",
			},
		},
		{
			name: "invalid flag",
			fields: fields{
				Nibbles: "0x20",
			},
		},
		{
			name: "low bits not empty for even length",
			fields: fields{
				Nibbles: "0x01",
			},
		},
		{
			name: "too long",
			fields: fields{
				Nibbles: "0x1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nibbles := hexutil.MustDecode(tt.fields.Nibbles)
			var n Nibbles
			err := n.Deserialize(codec.NewDecodingReader(bytes.NewReader(nibbles), uint64(len(nibbles))))
			assert.Error(t, err)
		})
	}
}

func TestFromUnpackedShouldFailForTooManyNibbles(t *testing.T) {
	_, err := FromUnpackedNibbles(initSlice(65, 10))
	assert.Error(t, err)
}

func initSlice(n int, v byte) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func TestAccountTrieNode(t *testing.T) {
	n, err := FromUnpackedNibbles([]byte{8, 6, 7, 9, 14, 8, 14, 13})
	require.NoError(t, err)

	accountTrieNode := &AccountTrieNodeKey{
		Path:     *n,
		NodeHash: common.Bytes32(hexutil.MustDecode("0x6225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296")),
	}
	var buf bytes.Buffer
	err = accountTrieNode.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	hexStr := hexutil.Encode(buf.Bytes())
	require.Equal(t, hexStr, "0x240000006225fcc63b22b80301d9f2582014e450e91f9b329b7cc87ad16894722fff5296008679e8ed")

	newAccount := &AccountTrieNodeKey{}
	err = newAccount.Deserialize(codec.NewDecodingReader(&buf, uint64(len(buf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newAccount.NodeHash, accountTrieNode.NodeHash)
	require.Equal(t, newAccount.Path.Nibbles, accountTrieNode.Path.Nibbles)
}

func TestContractStorageTrieNode(t *testing.T) {
	path, err := FromUnpackedNibbles([]byte{4, 0, 5, 7, 8, 7})
	require.NoError(t, err)
	contractStorage := &ContractStorageTrieNodeKey{
		Address:  common.Eth1Address(hexutil.MustDecode("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")),
		Path:     *path,
		NodeHash: common.Bytes32(hexutil.MustDecode("0xeb43d68008d216e753fef198cf51077f5a89f406d9c244119d1643f0f2b19011")),
	}

	var buf bytes.Buffer
	err = contractStorage.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	hexStr := hexutil.Encode(buf.Bytes())
	require.Equal(t, hexStr, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc238000000eb43d68008d216e753fef198cf51077f5a89f406d9c244119d1643f0f2b1901100405787")

	newContractStorage := &ContractStorageTrieNodeKey{}
	err = newContractStorage.Deserialize(codec.NewDecodingReader(&buf, uint64(len(buf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newContractStorage.NodeHash, contractStorage.NodeHash)
	require.Equal(t, newContractStorage.Path.Nibbles, contractStorage.Path.Nibbles)
	require.Equal(t, newContractStorage.Address, contractStorage.Address)
}

func TestContractBytecode(t *testing.T) {
	bytecode := &ContractBytecodeKey{
		Address:  common.Eth1Address(hexutil.MustDecode("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")),
		NodeHash: common.Bytes32(hexutil.MustDecode("0xd0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23")),
	}

	var buf bytes.Buffer
	err := bytecode.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	hexStr := hexutil.Encode(buf.Bytes())
	require.Equal(t, hexStr, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2d0a06b12ac47863b5c7be4185c2deaad1c61557033f56c7d4ea74429cbb25e23")

	newBytecode := &ContractBytecodeKey{}
	err = newBytecode.Deserialize(codec.NewDecodingReader(&buf, uint64(len(buf.Bytes()))))
	require.NoError(t, err)
	require.Equal(t, newBytecode.NodeHash, bytecode.NodeHash)
	require.Equal(t, newBytecode.Address, bytecode.Address)
}
