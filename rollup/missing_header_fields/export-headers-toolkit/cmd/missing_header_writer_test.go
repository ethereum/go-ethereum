package cmd

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreTypes "github.com/scroll-tech/go-ethereum/core/types"

	"github.com/scroll-tech/go-ethereum/common"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

func TestMissingHeaderWriter(t *testing.T) {
	vanity1 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	vanity2 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	vanity8 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08}

	stateRoot1 := common.HexToHash("0xabcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234")
	stateRoot2 := common.HexToHash("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")

	var expectedBytes []byte
	expectedBytes = append(expectedBytes, 0x03)
	expectedBytes = append(expectedBytes, vanity1[:]...)
	expectedBytes = append(expectedBytes, vanity2[:]...)
	expectedBytes = append(expectedBytes, vanity8[:]...)

	seenVanity := map[[32]byte]bool{
		vanity8: true,
		vanity1: true,
		vanity2: true,
	}
	var buf []byte
	bytesBuffer := bytes.NewBuffer(buf)
	mhw := newMissingHeaderWriter(bytesBuffer, seenVanity)

	mhw.writeVanities()
	assert.Equal(t, expectedBytes, bytesBuffer.Bytes())

	// Header0
	{
		seal := randomSeal(65)
		coinbase := common.Address{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
		nonce := coreTypes.BlockNonce{0, 1, 2, 3, 4, 5, 6, 7}
		header := types.NewHeader(0, 2, stateRoot1, coinbase, nonce, append(vanity1[:], seal...))
		mhw.write(header)

		// bit 6=0: difficulty 2, bit 7=0: seal length 65
		expectedBytes = append(expectedBytes, 0b00110000)
		expectedBytes = append(expectedBytes, 0x00) // vanity index0
		expectedBytes = append(expectedBytes, stateRoot1[:]...)
		expectedBytes = append(expectedBytes, coinbase[:]...)
		expectedBytes = append(expectedBytes, nonce[:]...)
		expectedBytes = append(expectedBytes, seal...)
		require.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header1
	{
		coinbase := common.Address{}
		nonce := coreTypes.BlockNonce{}
		seal := randomSeal(65)
		header := types.NewHeader(1, 1, stateRoot2, coinbase, nonce, append(vanity2[:], seal...))
		mhw.write(header)

		// bit 6=1: difficulty 1, bit 7=0: seal length 65
		expectedBytes = append(expectedBytes, 0b01000000)
		expectedBytes = append(expectedBytes, 0x01) // vanity index1
		expectedBytes = append(expectedBytes, stateRoot2[:]...)
		expectedBytes = append(expectedBytes, seal...)
		require.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header2
	coinbase := common.Address{1}
	nonce := coreTypes.BlockNonce{201}
	{
		seal := randomSeal(85)
		header := types.NewHeader(2, 2, stateRoot1, coinbase, nonce, append(vanity2[:], seal...))
		mhw.write(header)

		// bit 6=0: difficulty 2, bit 7=1: seal length 85
		expectedBytes = append(expectedBytes, 0b10110000)
		expectedBytes = append(expectedBytes, 0x01) // vanity index1
		expectedBytes = append(expectedBytes, stateRoot1[:]...)
		expectedBytes = append(expectedBytes, coinbase[:]...)
		expectedBytes = append(expectedBytes, nonce[:]...)
		expectedBytes = append(expectedBytes, seal...)
		require.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header3
	{
		seal := randomSeal(85)
		header := types.NewHeader(3, 1, stateRoot2, coinbase, nonce, append(vanity8[:], seal...))
		mhw.write(header)

		// bit 6=1: difficulty 1, bit 7=1: seal length 85
		expectedBytes = append(expectedBytes, 0b11110000)
		expectedBytes = append(expectedBytes, 0x02) // vanity index2
		expectedBytes = append(expectedBytes, stateRoot2[:]...)
		expectedBytes = append(expectedBytes, coinbase[:]...)
		expectedBytes = append(expectedBytes, nonce[:]...)
		expectedBytes = append(expectedBytes, seal...)
		require.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header4
	{
		stateRoot3 := common.Hash{123}
		seal := randomSeal(65)
		header := types.NewHeader(4, 2, stateRoot3, coinbase, nonce, append(vanity1[:], seal...))
		mhw.write(header)

		// bit 6=0: difficulty 2, bit 7=0: seal length 65
		expectedBytes = append(expectedBytes, 0b00110000)
		expectedBytes = append(expectedBytes, 0x00) // vanity index0
		expectedBytes = append(expectedBytes, stateRoot3[:]...)
		expectedBytes = append(expectedBytes, coinbase[:]...)
		expectedBytes = append(expectedBytes, nonce[:]...)
		expectedBytes = append(expectedBytes, seal...)
		require.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}
}

func randomSeal(length int) []byte {
	buf := make([]byte, length)
	_, _ = rand.Read(buf)
	return buf
}
