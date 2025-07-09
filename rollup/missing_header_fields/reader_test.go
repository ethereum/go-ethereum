package missing_header_fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type header struct {
	number     uint64
	difficulty uint64
	stateRoot  common.Hash
	coinbase   common.Address
	nonce      types.BlockNonce
	extra      []byte
}

var expectedMissingHeaders = []header{
	{0, 2, common.HexToHash("0x195dc9e93ed59fcd1d51e3262739761574b1d1518c6188e27a28357d9d93fb36"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("000000000000000000000000000000000000000000000000000000000000000048c3f81f3d998b6652900e1c3183736c238fe4290000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")},
	{1, 2, common.HexToHash("0x1c652497074d4a193fb16d61ffd6fc7727983e2d9e7010ac9ca31c241bdec3cb"), common.HexToAddress("0x687E0E85AD67ff71aC134CF61b65905b58Ab43b2"), types.BlockNonce(common.FromHex("ffffffffffffffff")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000228b5b48f65de89b35c77ae3791bfe26c1a3a91cffa2f30f50a58208f2b013ed1a86d1f5601e41b90a5345518e967985f2e949e53476ffc6d8781fa64e4e999101")},
	{2, 2, common.HexToHash("0x2ec6f0f086734c1d88f8ca915cbd2968804688c384aa6d95982435dc65ed476f"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000dbe3e10ae4c0e34d69bff96e86d4a0221361626e8cade7d9d69d870e37f7338f06a446b5461edd7f85d5dda1c54a41b9d0877e5fe9965c6a89a5f4aa3d0d6adc00")},
	{3, 2, common.HexToHash("0x16fc61ab25b479c4c367a85e3f537a598b92fb0e188464cbaca70b5dec08908d"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000f649dccc68b8a96a7d06d4a69e9667c63f90c2a2819870c695c2ef95e6862df55f5953891a5f9a13dfafbdff4d23a53a5a6ded505fc87203ac78142bb787dcf200")},
	{4, 2, common.HexToHash("0x001b539a66e87624114117e4d643d5aac7c716b6468c82a8daffea747d16b1a4"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e7578000000000000009e49f8090a1e4941660730b5f11bc0d648da7c04cf3c608f5e0725395c9690f44e1f1c46a9ccd26d3d4db7db973749acb83b88b1564352fe70afc30245242b6c01")},
	{5, 2, common.HexToHash("0x264d833b19677ec62af0f34a4040b9f8e50b3914a16bf961b7d1e198902f127a"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000f30ac95495ebb0a35a44657f0c361a7d1ea2d0a086ff7036f84875b47082094504809634f82e379b5ece2f43eead70b12e5d44c1befc4bb8763fac92b4fe8fbb01")},
	{6, 2, common.HexToHash("0x25ed6f6829966b24668510d4c828a596da881001470ab4e7e633761b6bdaba45"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000c9e8cf7bf35df3de8b7d0e98426a62a9e8ec302592f5622d527845ad78b53176558730f88dc79486ada7ed7aea3de4f8ca881b612440e31ab3b84ec7dff0cace01")},
	{7, 2, common.HexToHash("0x0e5e6c49b7c7cbcf3392c52f64b287c5a974b30f20f6695024231b5cf2155d0e"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000d8ebea45d74ad882718f97488083530d2d968edf1d3ff6d18d2c757feb0e40206b158f9d4e829a795849a1e3148dd3a6f5251fa600631a74825e3ea041222f5001")},
	{8, 2, common.HexToHash("0x2f6d586a8ce1fc4f476887aaffd8b143d4f5604ce2242f945ae5e8874ede7084"), common.HexToAddress("0x48C3F81f3D998b6652900e1C3183736C238Fe429"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000cecebe754d3c81738e2e76e6a0b34756006b38cd1237728ecf2f41f9a6e325634941b1b29e0e4c1338d1c0fb1190da361a5880253b116493e7cff288f9f165e300")},
	{9, 2, common.HexToHash("0x2c7f91ed6610d3823da4ed730968c40d62a99b5b6245e3f0d4f83011c3be9422"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000d6fa3b8ca99ca18fab9096d6791e70b7468a901f4cda0f48e7218e9807e79cee7b4fac8edeb23be5a593bbf9e0fc8ab678c0c58e7b4fb11869a1a4ac0f93657300")},
	{10, 2, common.HexToHash("0x1ae6b4a4b4f311a4f4a1b944445a32197f60d8070adcf7b92ed1f1cc42766504"), common.HexToAddress("0x0000000000000000000000000000000000000000"), types.BlockNonce(common.FromHex("0000000000000000")), common.FromHex("d88305050c846765746888676f312e32312e31856c696e75780000000000000016d5b03fbb592eb6cc5ec7ac98acc10b1b02184b69348b0ffb87817077e3066765573d90201d0b6690d49be1fda5a76f9edfddbdd45ebc69db3d59857e13bdf101")},
}

func TestReader_Read(t *testing.T) {
	expectedVanities := map[int][32]byte{
		0: [32]byte(common.FromHex("0000000000000000000000000000000000000000000000000000000000000000")),
		1: [32]byte(common.FromHex("d88305050c846765746888676f312e32312e31856c696e757800000000000000")),
	}

	reader, err := NewReader("testdata/missing-headers.bin")
	require.NoError(t, err)

	require.Len(t, reader.sortedVanities, len(expectedVanities))
	for i, expectedVanity := range expectedVanities {
		require.Equal(t, expectedVanity, reader.sortedVanities[i])
	}

	readAndAssertHeader(t, reader, expectedMissingHeaders, 0)
	readAndAssertHeader(t, reader, expectedMissingHeaders, 0)
	readAndAssertHeader(t, reader, expectedMissingHeaders, 1)
	readAndAssertHeader(t, reader, expectedMissingHeaders, 6)

	// reading previous headers resets the file reader
	readAndAssertHeader(t, reader, expectedMissingHeaders, 5)

	readAndAssertHeader(t, reader, expectedMissingHeaders, 8)
	readAndAssertHeader(t, reader, expectedMissingHeaders, 8)

	// reading previous headers resets the file reader
	readAndAssertHeader(t, reader, expectedMissingHeaders, 6)

	readAndAssertHeader(t, reader, expectedMissingHeaders, 9)
	readAndAssertHeader(t, reader, expectedMissingHeaders, 10)

	// no data anymore
	_, _, _, _, _, err = reader.Read(11)
	require.Error(t, err)
}

func readAndAssertHeader(t *testing.T, reader *Reader, expectedHeaders []header, headerNum uint64) {
	difficulty, stateRoot, coinbase, nonce, extra, err := reader.Read(headerNum)
	require.NoError(t, err)
	require.Equalf(t, expectedHeaders[headerNum].difficulty, difficulty, "expected difficulty %d, got %d", expectedHeaders[headerNum].difficulty, difficulty)
	require.Equalf(t, expectedHeaders[headerNum].stateRoot, stateRoot, "expected state root %s, got %s", expectedHeaders[headerNum].stateRoot.Hex(), stateRoot.Hex())
	require.Equalf(t, expectedHeaders[headerNum].coinbase, coinbase, "expected coinbase %s, got %s", expectedHeaders[headerNum].coinbase.Hex(), coinbase.Hex())
	require.Equalf(t, expectedHeaders[headerNum].nonce, nonce, "expected nonce %s, got %s", common.Bytes2Hex(expectedHeaders[headerNum].nonce[:]), common.Bytes2Hex(nonce[:]))
	require.Equal(t, expectedHeaders[headerNum].extra, extra)
}
