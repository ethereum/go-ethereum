package dasigners

import (
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bn254util"
	"github.com/ethereum/go-ethereum/crypto"
)

func PubkeyRegistrationHash(operatorAddress common.Address, chainId *big.Int) *bn254.G1Affine {
	toHash := make([]byte, 0)
	toHash = append(toHash, operatorAddress.Bytes()...)
	// make sure chainId is 32 bytes
	toHash = append(toHash, common.LeftPadBytes(chainId.Bytes(), 32)...)
	toHash = append(toHash, []byte("0G_BN254_Pubkey_Registration")...)

	msgHash := crypto.Keccak256(toHash)
	// convert to [32]byte
	var msgHash32 [32]byte
	copy(msgHash32[:], msgHash)

	// hash to G1
	return bn254util.MapToCurve(msgHash32)
}

func EpochRegistrationHash(operatorAddress common.Address, epoch uint64, chainId *big.Int) *bn254.G1Affine {
	toHash := make([]byte, 0)
	toHash = append(toHash, operatorAddress.Bytes()...)
	toHash = append(toHash, common.Uint64ToBytes(epoch)...)
	toHash = append(toHash, common.LeftPadBytes(chainId.Bytes(), 32)...)

	msgHash := crypto.Keccak256(toHash)
	// convert to [32]byte
	var msgHash32 [32]byte
	copy(msgHash32[:], msgHash)

	// hash to G1
	return bn254util.MapToCurve(msgHash32)
}

func ValidateSignature(s IDASignersSignerDetail, hash *bn254.G1Affine, signature *bn254.G1Affine) bool {
	pubkeyG1 := bn254util.DeserializeG1(SerializeG1(s.PkG1))
	pubkeyG2 := bn254util.DeserializeG2(SerializeG2(s.PkG2))
	gamma := bn254util.Gamma(hash, signature, pubkeyG1, pubkeyG2)

	// pairing
	P := [2]bn254.G1Affine{
		*new(bn254.G1Affine).Add(signature, new(bn254.G1Affine).ScalarMultiplication(pubkeyG1, gamma)),
		*new(bn254.G1Affine).Add(hash, new(bn254.G1Affine).ScalarMultiplication(bn254util.GetG1Generator(), gamma)),
	}
	Q := [2]bn254.G2Affine{
		*new(bn254.G2Affine).Neg(bn254util.GetG2Generator()),
		*pubkeyG2,
	}

	ok, err := bn254.PairingCheck(P[:], Q[:])
	if err != nil {
		return false
	}
	return ok
}
