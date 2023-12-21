package txpool

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type CustomValidationOptions struct {
}

type CustomValidator interface {
	Validate(tx *types.Transaction, head *types.Header, signer types.Signer, opts *CustomValidationOptions) error
}

type CustomValidatorHypernative struct {
	Config *CustomValidatorConfigHypernative
}

type CustomValidatorConfigHypernative struct {
	BannedAddresses []common.Address
}

func NewHypernativeValidator(config *CustomValidatorConfigHypernative) CustomValidator {
	return &CustomValidatorHypernative{
		Config: config,
	}
}

func (v *CustomValidatorHypernative) Validate(tx *types.Transaction, head *types.Header, signer types.Signer, opts *CustomValidationOptions) error {
	// ban certain senders just for the fun of it
	if address, err := signer.Sender(tx); err == nil {
		for _, bannedAddress := range v.Config.BannedAddresses {
			if address == bannedAddress {
				return ErrCustomValidationFailed
			}
		}
	} else {
		return err
	}

	return nil
}
