package utils

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func TestGetM1M2FromCheckpointHeader(t *testing.T) {
	masternodes := []common.Address{
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
	}
	validators := []int64{
		2,
		1,
		0,
	}
	epoch := uint64(900)
	config := &params.ChainConfig{
		XDPoS: &params.XDPoSConfig{
			Epoch: uint64(epoch),
		},
	}
	testMoveM2 := []uint64{0, 0, 0, 1, 1, 1, 2, 2, 2, 0, 0, 0, 1, 1, 1, 2, 2, 2}
	//try from block 3410001 to 3410018
	for i := uint64(3464001); i <= 3464018; i++ {
		currentNumber := int64(i)
		currentHeader := &types.Header{
			Number: big.NewInt(currentNumber),
		}
		m1m2, moveM2, err := GetM1M2(masternodes, validators, currentHeader, config)
		if err != nil {
			t.Error("can't get m1m2", "err", err)
		}
		fmt.Printf("block: %v, moveM2: %v\n", currentHeader.Number.Int64(), moveM2)
		for _, k := range masternodes {
			fmt.Printf("m1: %v - m2: %v\n", k.Str(), m1m2[k].Str())
		}
		if moveM2 != testMoveM2[i-3464001] {
			t.Error("wrong moveM2", "currentNumber", currentNumber, "want", testMoveM2[i-3464001], "have", moveM2)
		}
	}
}

func TestCompareSignersLists(t *testing.T) {
	list1 := []common.Address{
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
	}
	list2 := []common.Address{
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	list3 := []common.Address{
		common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc"),
		common.StringToAddress("dddddddddddddddddddddddddddddddddddddddd"),
		common.StringToAddress("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}
	if !CompareSignersLists(list1, list2) {
		t.Error("list1 should be equal to list2", "list1", list1, "list2", list2)
	}
	if CompareSignersLists(list1, list3) {
		t.Error("list1 and list3 should not be same", "list1", list1, "list3", list3)
	}
	if !CompareSignersLists([]common.Address{}, []common.Address{}) {
		t.Error("Failed with empty list")
	}
	if !CompareSignersLists([]common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}, []common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}) {
		t.Error("Failed with list has only one signer")
	}
	if CompareSignersLists([]common.Address{common.StringToAddress("aaaaaaaaaaaaaaaa")}, []common.Address{common.StringToAddress("cccccccccccccccccccccccccccccccccccccccc")}) {
		t.Error("Failed with list has only one signer")
	}
}

func toyExtraFields() *ExtraFields_v2 {
	round := Round(307)
	blockInfo := BlockInfo{Hash: common.BigToHash(big.NewInt(2047)), Round: round - 1, Number: big.NewInt(1)}
	signature := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	signatures := [][]byte{signature}
	quorumCert := QuorumCert{ProposedBlockInfo: blockInfo, Signatures: signatures}
	e := &ExtraFields_v2{Round: round, QuorumCert: quorumCert}
	return e
}
func TestExtraFieldsEncodeDecode(t *testing.T) {
	extraFields := toyExtraFields()
	encoded, err := extraFields.EncodeToBytes()
	if err != nil {
		t.Errorf("Error when encoding extra fields")
	}
	var decoded ExtraFields_v2
	err = DecodeBytesExtraFields(encoded, &decoded)
	if err != nil {
		t.Errorf("Error when decoding extra fields")
	}
	if !reflect.DeepEqual(*extraFields, decoded) {
		t.Fatalf("Decoded not equal to original extra field, original: %v; decoded: %v", extraFields, decoded)
	}
}

func TestHashAndSigHash(t *testing.T) {
	round := Round(307)
	blockInfo1 := BlockInfo{Hash: common.BigToHash(big.NewInt(2047)), Round: round - 1, Number: big.NewInt(1)}
	blockInfo2 := BlockInfo{Hash: common.BigToHash(big.NewInt(4095)), Round: round - 1, Number: big.NewInt(1)}
	signature1 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	signature2 := []byte{1, 2, 3, 4, 5, 6, 7, 7}
	signatures1 := [][]byte{signature1}
	quorumCert1 := QuorumCert{ProposedBlockInfo: blockInfo1, Signatures: signatures1}
	signatures2 := [][]byte{signature2}
	quorumCert2 := QuorumCert{ProposedBlockInfo: blockInfo1, Signatures: signatures2}
	vote1 := Vote{ProposedBlockInfo: blockInfo1, Signature: signature1}
	vote2 := Vote{ProposedBlockInfo: blockInfo1, Signature: signature2}
	if vote1.Hash() == vote2.Hash() {
		t.Fatalf("Hash of two votes shouldn't equal")
	}
	timeout1 := Timeout{Round: 10, Signature: signature1}
	timeout2 := Timeout{Round: 10, Signature: signature2}
	if timeout1.Hash() == timeout2.Hash() {
		t.Fatalf("Hash of two timeouts shouldn't equal")
	}
	syncInfo1 := SyncInfo{HighestQuorumCert: quorumCert1}
	syncInfo2 := SyncInfo{HighestQuorumCert: quorumCert2}
	if syncInfo1.Hash() == syncInfo2.Hash() {
		t.Fatalf("Hash of two sync info shouldn't equal")
	}
	if VoteSigHash(&blockInfo1) == VoteSigHash(&blockInfo2) {
		t.Fatalf("SigHash of two block info shouldn't equal")
	}
	round2 := Round(999)
	if TimeoutSigHash(&round) == TimeoutSigHash(&round2) {
		t.Fatalf("SigHash of two round shouldn't equal")
	}
}
