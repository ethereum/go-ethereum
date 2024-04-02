package eccpow

import (
	"reflect"
	"testing"

	"github.com/cryptoecc/ETH-ECC/common"
	"github.com/cryptoecc/ETH-ECC/common/hexutil"
	"github.com/cryptoecc/ETH-ECC/core/types"
)

func TestRandomSeed(t *testing.T) {
	header := new(types.Header)
	header.Difficulty = ProbToDifficulty(Table[0].miningProb)
	parameters, _ := setParameters(header)

	a := generateH(parameters)
	b := generateH(parameters)

	if !reflect.DeepEqual(a, b) {
		t.Error("Wrong matrix")
	} else {
		t.Log("Pass")
	}
}

func TestLDPC(t *testing.T) {
	/*
		prevHash := hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000")
		curHash := hexutil.MustDecode("0xca2ff06caae7c94dc968be7d76d0fbf60dd2e1989ee9bf0d5931e48564d5143b")
		nonce, mixDigest := RunLDPC(prevHash, curHash)

		wantDigest := hexutil.MustDecode("0x535306ee4b42c92aecd0e71fca98572064f049c2babb2769faa3bbd87d67ec2d")

		if !bytes.Equal(mixDigest, wantDigest) {
			t.Errorf("light hashimoto digest mismatch: have %x, want %x", mixDigest, wantDigest)
		}

		t.Log(nonce)
	*/
	header := new(types.Header)
	//t.Log(hexutil.Encode(header.ParentHash))
	header.Difficulty = ProbToDifficulty(Table[0].miningProb)
	var hash []byte
	_, hashVector, outputWord, LDPCNonce, digest := RunOptimizedConcurrencyLDPC(header, hash)

	t.Logf("Hash vector : %v\n", hashVector)
	t.Logf("Outputword : %v\n", outputWord)
	t.Logf("LDPC Nonce : %v\n", LDPCNonce)
	t.Logf("Digest : %v\n", digest)
}

func BenchmarkECCPoW(b *testing.B) {
	//prevHash := hexutil.MustDecode("0xd783efa4d392943503f28438ad5830b2d5964696ffc285f338585e9fe0a37a05")
	//curHash := hexutil.MustDecode("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")

	header := new(types.Header)
	header.Difficulty = ProbToDifficulty(Table[0].miningProb)
	var hash []byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunOptimizedConcurrencyLDPC(header, hash)
	}
}


func TestHashRate(t *testing.T) {
	var (
		hashrate = []hexutil.Uint64{100, 200, 300}
		expect   uint64
		ids      = []common.Hash{common.HexToHash("a"), common.HexToHash("b"), common.HexToHash("c")}
	)
	ecc := NewTester(nil, false)
	defer ecc.Close()

	if tot := ecc.Hashrate(); tot != 0 {
		t.Error("expect the result should be zero")
	}

	api := &API{ecc}
	for i := 0; i < len(hashrate); i++ {
		if res := api.SubmitHashRate(hashrate[i], ids[i]); !res {
			t.Error("remote miner submit hashrate failed")
		}
		expect += uint64(hashrate[i])
	}
	if tot := ecc.Hashrate(); tot != float64(expect) {
		t.Error("expect total hashrate should be same")
	}
}
