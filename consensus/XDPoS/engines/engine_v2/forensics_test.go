package engine_v2

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/stretchr/testify/assert"
)

// Utils to help mocking the signing of signatures
var (
	signer1, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	signer2, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	signer3, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func SignHashByPK(pk *ecdsa.PrivateKey, itemToSign []byte) []byte {
	signer, signFn, err := getSignerAndSignFn(pk)
	if err != nil {
		panic(err)
	}
	signedHash, err := signFn(accounts.Account{Address: signer}, itemToSign)
	if err != nil {
		panic(err)
	}
	return signedHash
}
func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func getSignerAndSignFn(pk *ecdsa.PrivateKey) (common.Address, func(account accounts.Account, hash []byte) ([]byte, error), error) {
	veryLightScryptN := 2
	veryLightScryptP := 1
	dir, _ := os.MkdirTemp("", fmt.Sprintf("eth-getSignerAndSignFn-test-%v", RandStringBytes(5)))
	defer os.RemoveAll(dir)

	ks := keystore.NewKeyStore(dir, veryLightScryptN, veryLightScryptP)
	pass := "" // not used but required by API
	a1, err := ks.ImportECDSA(pk, pass)
	if err != nil {
		return common.Address{}, nil, err
	}
	if err := ks.Unlock(a1, ""); err != nil {
		return a1.Address, nil, err
	}
	return a1.Address, ks.SignHash, nil
}

func TestFindQCsInSameRound(t *testing.T) {
	forensics := &Forensics{}
	gapNumber := 450

	// If ONE in common
	var sig []types.Signature
	qc1 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc1"),
			Round:  types.Round(10),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc2 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc2"),
			Round:  types.Round(12),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc3 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc3"),
			Round:  types.Round(13),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc4 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc4"),
			Round:  types.Round(12),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc5 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc5"),
			Round:  types.Round(13),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc6 := &types.QuorumCert{
		ProposedBlockInfo: &types.BlockInfo{
			Hash:   common.StringToHash("qc6"),
			Round:  types.Round(15),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	var qcSet1 []types.QuorumCert
	var qcSet2 []types.QuorumCert

	found, first, second := forensics.findQCsInSameRound(append(qcSet1, *qc1, *qc2, *qc3), append(qcSet2, *qc4, *qc5, *qc6))
	assert.True(t, found)
	assert.Equal(t, *qc2, first)
	assert.Equal(t, *qc4, second)
}

// TODO: Add test for FindAncestorBlockHash
