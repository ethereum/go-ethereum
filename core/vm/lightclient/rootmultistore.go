package lightclient

import (
	"fmt"

	"github.com/tendermint/tendermint/crypto/merkle"
	"github.com/tendermint/tendermint/crypto/tmhash"
)

//----------------------------------------
// CommitID

// CommitID contains the tree version number and its merkle root.
type CommitID struct {
	Version int64
	Hash    []byte
}

func (cid CommitID) IsZero() bool { //nolint
	return cid.Version == 0 && len(cid.Hash) == 0
}

func (cid CommitID) String() string {
	return fmt.Sprintf("CommitID{%v:%X}", cid.Hash, cid.Version)
}

//----------------------------------------
// CommitInfo

// NOTE: Keep CommitInfo a simple immutable struct.
type CommitInfo struct {

	// Version
	Version int64

	// Store info for
	StoreInfos []StoreInfo
}

// Hash returns the simple merkle root hash of the stores sorted by name.
func (ci CommitInfo) Hash() []byte {
	// TODO cache to ci.hash []byte
	m := make(map[string][]byte, len(ci.StoreInfos))
	for _, storeInfo := range ci.StoreInfos {
		m[storeInfo.Name] = storeInfo.Hash()
	}
	return merkle.SimpleHashFromMap(m)
}

func (ci CommitInfo) CommitID() CommitID {
	return CommitID{
		Version: ci.Version,
		Hash:    ci.Hash(),
	}
}

//----------------------------------------
// StoreInfo

// StoreInfo contains the name and core reference for an
// underlying store.  It is the leaf of the rootMultiStores top
// level simple merkle tree.
type StoreInfo struct {
	Name string
	Core StoreCore
}

type StoreCore struct {
	// StoreType StoreType
	CommitID CommitID
	// ... maybe add more state
}

// Implements merkle.Hasher.
func (si StoreInfo) Hash() []byte {
	// Doesn't write Name, since merkle.SimpleHashFromMap() will
	// include them via the keys.
	bz, _ := Cdc.MarshalBinaryLengthPrefixed(si.Core) // Does not error
	hasher := tmhash.New()
	_, err := hasher.Write(bz)
	if err != nil {
		// TODO: Handle with #870
		panic(err)
	}
	return hasher.Sum(nil)
}
