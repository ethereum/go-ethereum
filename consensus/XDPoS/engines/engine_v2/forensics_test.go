package engine_v2

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
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
	dir, _ := ioutil.TempDir("", fmt.Sprintf("eth-getSignerAndSignFn-test-%v", RandStringBytes(5)))

	new := func(kd string) *keystore.KeyStore {
		return keystore.NewKeyStore(kd, veryLightScryptN, veryLightScryptP)
	}

	defer os.RemoveAll(dir)
	ks := new(dir)
	pass := "" // not used but required by API
	a1, err := ks.ImportECDSA(pk, pass)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf(err.Error())
	}
	if err := ks.Unlock(a1, ""); err != nil {
		return a1.Address, nil, fmt.Errorf(err.Error())
	}
	return a1.Address, ks.SignHash, nil
}

func TestFindCommonSigners(t *testing.T) {
	forensics := &Forensics{}
	proposedBlockInfo := &utils.BlockInfo{
		Hash:   common.StringToHash("123"),
		Round:  utils.Round(10),
		Number: big.NewInt(910),
	}
	gapNumber := 450
	voteForSign := &utils.VoteForSign{
		ProposedBlockInfo: proposedBlockInfo,
		GapNumber:         uint64(gapNumber),
	}
	signatureFromSigner1 := SignHashByPK(signer1, utils.VoteSigHash(voteForSign).Bytes())
	signatureFromSigner2 := SignHashByPK(signer2, utils.VoteSigHash(voteForSign).Bytes())
	signatureFromSigner3 := SignHashByPK(signer3, utils.VoteSigHash(voteForSign).Bytes())

	// If ONE in common
	var signaturesForQC1 []utils.Signature
	qc1 := &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC1, signatureFromSigner1, signatureFromSigner2),
		GapNumber:         uint64(gapNumber),
	}

	var signaturesForQC2 []utils.Signature
	qc2 := &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC2, signatureFromSigner2, signatureFromSigner3),
		GapNumber:         uint64(gapNumber),
	}

	commonSigners := forensics.findCommonSigners(*qc1, *qc2)
	assert.Equal(t, 1, len(commonSigners))
	assert.Equal(t, crypto.PubkeyToAddress(signer2.PublicKey), commonSigners[0])

	// If none in common
	var signaturesForQC1NoneInCommon []utils.Signature
	qc1 = &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC1NoneInCommon, signatureFromSigner1),
		GapNumber:         uint64(gapNumber),
	}

	var signaturesForQC2NoneInCommon []utils.Signature
	qc2 = &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC2NoneInCommon, signatureFromSigner2, signatureFromSigner3),
		GapNumber:         uint64(gapNumber),
	}

	commonSigners = forensics.findCommonSigners(*qc1, *qc2)
	assert.Equal(t, 0, len(commonSigners))

	// All in common
	var signaturesForQC1AllInCommon []utils.Signature
	qc1 = &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC1AllInCommon, signatureFromSigner1, signatureFromSigner2, signatureFromSigner3),
		GapNumber:         uint64(gapNumber),
	}

	var signaturesForQC2AllInCommon []utils.Signature
	qc2 = &utils.QuorumCert{
		ProposedBlockInfo: proposedBlockInfo,
		Signatures:        append(signaturesForQC2AllInCommon, signatureFromSigner1, signatureFromSigner2, signatureFromSigner3),
		GapNumber:         uint64(gapNumber),
	}

	commonSigners = forensics.findCommonSigners(*qc1, *qc2)
	assert.Equal(t, 3, len(commonSigners))
	assert.Equal(t, crypto.PubkeyToAddress(signer1.PublicKey), commonSigners[0])
	assert.Equal(t, crypto.PubkeyToAddress(signer2.PublicKey), commonSigners[1])
	assert.Equal(t, crypto.PubkeyToAddress(signer3.PublicKey), commonSigners[2])
}

func TestFindQCsInSameRound(t *testing.T) {
	forensics := &Forensics{}
	gapNumber := 450

	// If ONE in common
	var sig []utils.Signature
	qc1 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc1"),
			Round:  utils.Round(10),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc2 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc2"),
			Round:  utils.Round(12),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc3 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc3"),
			Round:  utils.Round(13),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc4 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc4"),
			Round:  utils.Round(12),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc5 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc5"),
			Round:  utils.Round(13),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	qc6 := &utils.QuorumCert{
		ProposedBlockInfo: &utils.BlockInfo{
			Hash:   common.StringToHash("qc6"),
			Round:  utils.Round(15),
			Number: big.NewInt(910),
		},
		Signatures: sig,
		GapNumber:  uint64(gapNumber),
	}

	var qcSet1 []utils.QuorumCert
	var qcSet2 []utils.QuorumCert

	found, first, second := forensics.findQCsInSameRound(append(qcSet1, *qc1, *qc2, *qc3), append(qcSet2, *qc4, *qc5, *qc6))
	assert.True(t, found)
	assert.Equal(t, *qc2, first)
	assert.Equal(t, *qc4, second)
}
