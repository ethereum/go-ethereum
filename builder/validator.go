package builder

import (
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
)

type ValidatorPrivateData struct {
	sk *bls.SecretKey
	Pk hexutil.Bytes
}

func NewRandomValidator() *ValidatorPrivateData {
	sk, pk, err := bls.GenerateNewKeypair()
	if err != nil {
		return nil
	}
	return &ValidatorPrivateData{sk, pk.Compress()}
}

func (v *ValidatorPrivateData) Sign(msg boostTypes.HashTreeRoot, d boostTypes.Domain) (boostTypes.Signature, error) {
	return boostTypes.SignMessage(msg, d, v.sk)
}

func (v *ValidatorPrivateData) PrepareRegistrationMessage(feeRecipientHex string) (boostTypes.SignedValidatorRegistration, error) {
	address, err := boostTypes.HexToAddress(feeRecipientHex)
	if err != nil {
		return boostTypes.SignedValidatorRegistration{}, err
	}

	pubkey := boostTypes.PublicKey{}
	pubkey.FromSlice(v.Pk)

	msg := &boostTypes.RegisterValidatorRequestMessage{
		FeeRecipient: address,
		GasLimit:     1000,
		Timestamp:    uint64(time.Now().UnixMilli()),
		Pubkey:       pubkey,
	}
	signature, err := v.Sign(msg, boostTypes.DomainBuilder)
	if err != nil {
		return boostTypes.SignedValidatorRegistration{}, err
	}
	return boostTypes.SignedValidatorRegistration{Message: msg, Signature: signature}, nil
}
