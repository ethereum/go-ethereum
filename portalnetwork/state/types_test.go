package state

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/ztyp/codec"
	"github.com/stretchr/testify/assert"
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
