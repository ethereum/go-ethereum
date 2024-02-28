package portalwire

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var maxUint256 = uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

// https://github.com/ethereum/portal-network-specs/blob/master/portal-wire-test-vectors.md
// we remove the message type here
func TestPingMessage(t *testing.T) {
	dataRadius := maxUint256.Sub(maxUint256, uint256.NewInt(1))
	reverseBytes, err := dataRadius.MarshalSSZ()
	require.NoError(t, err)
	customData := &PingPongCustomData{
		Radius: reverseBytes,
	}
	dataBytes, err := customData.MarshalSSZ()
	assert.NoError(t, err)
	ping := &Ping{
		EnrSeq:        1,
		CustomPayload: dataBytes,
	}

	expected := "0x01000000000000000c000000feffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

	data, err := ping.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))
}

func TestPongMessage(t *testing.T) {
	dataRadius := maxUint256.Div(maxUint256, uint256.NewInt(2))
	reverseBytes, err := dataRadius.MarshalSSZ()
	require.NoError(t, err)
	customData := &PingPongCustomData{
		Radius: reverseBytes,
	}

	dataBytes, err := customData.MarshalSSZ()
	assert.NoError(t, err)
	pong := &Pong{
		EnrSeq:        1,
		CustomPayload: dataBytes,
	}

	expected := "0x01000000000000000c000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f"

	data, err := pong.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))
}

func TestFindNodesMessage(t *testing.T) {
	distances := []uint16{256, 255}

	distancesBytes := make([][2]byte, len(distances))
	for i, distance := range distances {
		copy(distancesBytes[i][:], ssz.MarshalUint16(make([]byte, 0), distance))
	}

	findNode := &FindNodes{
		Distances: distancesBytes,
	}

	data, err := findNode.MarshalSSZ()
	expected := "0x040000000001ff00"
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))
}

func TestNodes(t *testing.T) {
	enrs := []string{
		"enr:-HW4QBzimRxkmT18hMKaAL3IcZF1UcfTMPyi3Q1pxwZZbcZVRI8DC5infUAB_UauARLOJtYTxaagKoGmIjzQxO2qUygBgmlkgnY0iXNlY3AyNTZrMaEDymNMrg1JrLQB2KTGtv6MVbcNEVv0AHacwUAPMljNMTg",
		"enr:-HW4QNfxw543Ypf4HXKXdYxkyzfcxcO-6p9X986WldfVpnVTQX1xlTnWrktEWUbeTZnmgOuAY_KUhbVV1Ft98WoYUBMBgmlkgnY0iXNlY3AyNTZrMaEDDiy3QkHAxPyOgWbxp5oF1bDdlYE6dLCUUp8xfVw50jU",
	}

	enrsBytes := make([][]byte, 0)
	for _, enr := range enrs {
		n, err := enode.Parse(enode.ValidSchemes, enr)
		assert.NoError(t, err)

		enrBytes, err := rlp.EncodeToBytes(n.Record())
		assert.NoError(t, err)
		enrsBytes = append(enrsBytes, enrBytes)
	}

	testCases := []struct {
		name     string
		input    [][]byte
		expected string
	}{
		{
			name:     "empty nodes",
			input:    make([][]byte, 0),
			expected: "0x0105000000",
		},
		{
			name:     "two nodes",
			input:    enrsBytes,
			expected: "0x0105000000080000007f000000f875b8401ce2991c64993d7c84c29a00bdc871917551c7d330fca2dd0d69c706596dc655448f030b98a77d4001fd46ae0112ce26d613c5a6a02a81a6223cd0c4edaa53280182696482763489736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd3138f875b840d7f1c39e376297f81d7297758c64cb37dcc5c3beea9f57f7ce9695d7d5a67553417d719539d6ae4b445946de4d99e680eb8063f29485b555d45b7df16a1850130182696482763489736563703235366b31a1030e2cb74241c0c4fc8e8166f1a79a05d5b0dd95813a74b094529f317d5c39d235",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			nodes := &Nodes{
				Total: 1,
				Enrs:  test.input,
			}

			data, err := nodes.MarshalSSZ()
			assert.NoError(t, err)
			assert.Equal(t, test.expected, fmt.Sprintf("0x%x", data))
		})
	}
}

func TestContent(t *testing.T) {
	contentKey := "0x706f7274616c"

	content := &FindContent{
		ContentKey: hexutil.MustDecode(contentKey),
	}
	expected := "0x04000000706f7274616c"
	data, err := content.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))

	expected = "0x7468652063616b652069732061206c6965"

	contentRes := &Content{
		Content: hexutil.MustDecode("0x7468652063616b652069732061206c6965"),
	}

	data, err = contentRes.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))

	expectData := &Content{}
	err = expectData.UnmarshalSSZ(data)
	assert.NoError(t, err)
	assert.Equal(t, contentRes.Content, expectData.Content)

	enrs := []string{
		"enr:-HW4QBzimRxkmT18hMKaAL3IcZF1UcfTMPyi3Q1pxwZZbcZVRI8DC5infUAB_UauARLOJtYTxaagKoGmIjzQxO2qUygBgmlkgnY0iXNlY3AyNTZrMaEDymNMrg1JrLQB2KTGtv6MVbcNEVv0AHacwUAPMljNMTg",
		"enr:-HW4QNfxw543Ypf4HXKXdYxkyzfcxcO-6p9X986WldfVpnVTQX1xlTnWrktEWUbeTZnmgOuAY_KUhbVV1Ft98WoYUBMBgmlkgnY0iXNlY3AyNTZrMaEDDiy3QkHAxPyOgWbxp5oF1bDdlYE6dLCUUp8xfVw50jU",
	}

	enrsBytes := make([][]byte, 0)
	for _, enr := range enrs {
		n, err := enode.Parse(enode.ValidSchemes, enr)
		assert.NoError(t, err)

		enrBytes, err := rlp.EncodeToBytes(n.Record())
		assert.NoError(t, err)
		enrsBytes = append(enrsBytes, enrBytes)
	}

	enrsRes := &Enrs{
		Enrs: enrsBytes,
	}

	expected = "0x080000007f000000f875b8401ce2991c64993d7c84c29a00bdc871917551c7d330fca2dd0d69c706596dc655448f030b98a77d4001fd46ae0112ce26d613c5a6a02a81a6223cd0c4edaa53280182696482763489736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd3138f875b840d7f1c39e376297f81d7297758c64cb37dcc5c3beea9f57f7ce9695d7d5a67553417d719539d6ae4b445946de4d99e680eb8063f29485b555d45b7df16a1850130182696482763489736563703235366b31a1030e2cb74241c0c4fc8e8166f1a79a05d5b0dd95813a74b094529f317d5c39d235"

	data, err = enrsRes.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))

	expectEnrs := &Enrs{}
	err = expectEnrs.UnmarshalSSZ(data)
	assert.NoError(t, err)
	assert.Equal(t, expectEnrs.Enrs, enrsRes.Enrs)
}

func TestOfferAndAcceptMessage(t *testing.T) {
	contentKey := "0x010203"
	contentBytes := hexutil.MustDecode(contentKey)
	contentKeys := [][]byte{contentBytes}
	offer := &Offer{
		ContentKeys: contentKeys,
	}

	expected := "0x0400000004000000010203"

	data, err := offer.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))

	contentKeyBitlist := bitfield.NewBitlist(8)
	contentKeyBitlist.SetBitAt(0, true)
	accept := &Accept{
		ConnectionId: []byte{0x01, 0x02},
		ContentKeys:  contentKeyBitlist,
	}

	expected = "0x0102060000000101"

	data, err = accept.MarshalSSZ()
	assert.NoError(t, err)
	assert.Equal(t, expected, fmt.Sprintf("0x%x", data))
}
