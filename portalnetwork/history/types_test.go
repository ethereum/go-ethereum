package history

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/view"
	"github.com/stretchr/testify/require"
)

// testcases from https://github.com/ethereum/portal-network-specs/blob/master/content-keys-test-vectors.md
func TestContentKey(t *testing.T) {
	testCases := []struct {
		name          string
		hash          string
		contentKey    string
		contentIdHex  string
		contentIdU256 string
		selector      ContentType
	}{
		{
			name:          "block header key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "00d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "3e86b3767b57402ea72e369ae0496ce47cc15be685bec3b4726b9f316e3895fe",
			contentIdU256: "28281392725701906550238743427348001871342819822834514257505083923073246729726",
			selector:      BlockHeaderType,
		},
		{
			name:          "block body key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "01d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "ebe414854629d60c58ddd5bf60fd72e41760a5f7a463fdcb169f13ee4a26786b",
			contentIdU256: "106696502175825986237944249828698290888857178633945273402044845898673345165419",
			selector:      BlockBodyType,
		},
		{
			name:          "receipt key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "02d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "a888f4aafe9109d495ac4d4774a6277c1ada42035e3da5e10a04cc93247c04a4",
			contentIdU256: "76230538398907151249589044529104962263309222250374376758768131420767496438948",
			selector:      ReceiptsType,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			hashByte, err := hex.DecodeString(c.hash)
			require.NoError(t, err)

			contentKey := newContentKey(c.selector, hashByte).encode()
			hexKey := hex.EncodeToString(contentKey)
			require.Equal(t, hexKey, c.contentKey)
			contentId := ContentId(contentKey)
			require.Equal(t, c.contentIdHex, hex.EncodeToString(contentId))

			bigNum := big.NewInt(0).SetBytes(contentId)
			u256Format, isOverflow := uint256.FromBig(bigNum)
			require.False(t, isOverflow)
			u256Str := fmt.Sprint(u256Format)
			require.Equal(t, u256Str, c.contentIdU256)
		})
	}
}

func TestBlockNumber(t *testing.T) {
	blockNumber := 12345678
	contentKey := "0x034e61bc0000000000"
	contentId := "0x2113990747a85ab39785d21342fa5db1f68acc0011605c0c73f68fc331643dcf"
	contentIdU256 := "14960950260935695396511307566164035182676768442501235074589175304147024756175"

	key := view.Uint64View(blockNumber)
	var buf bytes.Buffer
	err := key.Serialize(codec.NewEncodingWriter(&buf))
	require.NoError(t, err)
	keyData := []byte{byte(BlockHeaderNumberType)}
	keyData = append(keyData, buf.Bytes()...)
	require.Equal(t, hexutil.MustDecode(contentKey), keyData)

	contentIdData := ContentId(keyData)
	require.Equal(t, contentId, hexutil.Encode(contentIdData))

	bigNum := big.NewInt(0).SetBytes(contentIdData)
	u256Format, isOverflow := uint256.FromBig(bigNum)
	require.False(t, isOverflow)
	u256Str := fmt.Sprint(u256Format)
	require.Equal(t, u256Str, contentIdU256)
}
