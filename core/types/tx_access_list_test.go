package types

import (
	"crypto/ecdsa"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAccessListWithoutAuthorizations(t *testing.T) {
	generateKey := func(t *testing.T) *ecdsa.PrivateKey {
		key, err := crypto.GenerateKey()
		require.NoError(t, err)
		return key
	}

	prepareAuth := func(t *testing.T, auth SetCodeAuthorization, key *ecdsa.PrivateKey) SetCodeAuthorization {
		signature, err := crypto.Sign(auth.sigHash().Bytes(), key)
		require.NoError(t, err)
		auth.R.SetBytes(signature[0:32])
		auth.S.SetBytes(signature[32:64])
		auth.V = signature[64]
		return auth
	}

	targetChainID := uint256.NewInt(1)
	key1, key2, key3, key4, key5 := generateKey(t), generateKey(t), generateKey(t), generateKey(t), generateKey(t)

	authorizationWithInvalidSignature := SetCodeAuthorization{ChainID: *targetChainID}
	authorizationWithInvalidChainID := prepareAuth(t, SetCodeAuthorization{ChainID: *uint256.NewInt(2)}, key1)
	authorizationWithNonceOverflow := prepareAuth(t, SetCodeAuthorization{ChainID: *targetChainID, Nonce: math.MaxUint64}, key2)
	validAuthorizationForAddressWithStorage := prepareAuth(t, SetCodeAuthorization{ChainID: *targetChainID}, key3)
	validAuthorizationForAddressWithoutStorage1 := prepareAuth(t, SetCodeAuthorization{}, key4)
	validAuthorizationForAddressWithoutStorage2 := prepareAuth(t, SetCodeAuthorization{ChainID: *targetChainID}, key5)

	accessTuple1 := AccessTuple{Address: crypto.PubkeyToAddress(key1.PublicKey)}
	accessTuple2 := AccessTuple{Address: crypto.PubkeyToAddress(key2.PublicKey)}
	accessTuple3 := AccessTuple{Address: crypto.PubkeyToAddress(key3.PublicKey), StorageKeys: []common.Hash{{}}}
	accessTuple4 := AccessTuple{Address: crypto.PubkeyToAddress(key4.PublicKey), StorageKeys: []common.Hash{}} // should be deleted
	accessTuple5 := AccessTuple{Address: crypto.PubkeyToAddress(key5.PublicKey)}                               // should be deleted
	accessTuple6 := AccessTuple{Address: common.Address{1}}

	al := AccessList{accessTuple1, accessTuple2, accessTuple3, accessTuple4, accessTuple5, accessTuple6}

	require.Equal(t, al, al.WithoutAuthorizations(targetChainID.ToBig(), nil), "AccessList should not be modified without authorizations")
	require.Equal(t, AccessList{accessTuple1, accessTuple2, accessTuple3, accessTuple6}, al.WithoutAuthorizations(targetChainID.ToBig(), []SetCodeAuthorization{
		authorizationWithInvalidSignature,
		authorizationWithInvalidChainID,
		authorizationWithNonceOverflow,
		validAuthorizationForAddressWithStorage,
		validAuthorizationForAddressWithoutStorage1,
		validAuthorizationForAddressWithoutStorage2,
	}))
}
