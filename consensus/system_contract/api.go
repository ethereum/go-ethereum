package system_contract

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-authority scheme.
type API struct {
	system_contract *SystemContract
}

// GetSigners retrieves the list of authorized signers at the specified block.
func (api *API) GetLocalSigner() (common.Address, error) {
	return api.system_contract.localSignerAddress(), nil
}

// GetSigners retrieves the list of authorized signers at the specified block.
func (api *API) GetAuthorizedSigner() (common.Address, error) {
	return api.system_contract.currentSignerAddressL1(), nil
}
