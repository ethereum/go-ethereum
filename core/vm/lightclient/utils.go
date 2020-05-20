package lightclient

import (
	"fmt"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
	tmtypes "github.com/tendermint/tendermint/types"
)

func GetInitConsensusState(node rpcclient.Client, height int64) (*ConsensusState, error) {
	status, err := node.Status()
	if err != nil {
		return nil, err
	}

	nextValHeight := height + 1
	nextValidatorSet, err := node.Validators(&nextValHeight)
	if err != nil {
		return nil, err
	}

	header, err := node.Block(&height)
	if err != nil {
		return nil, err
	}

	appHash := header.BlockMeta.Header.AppHash
	curValidatorSetHash := header.BlockMeta.Header.ValidatorsHash

	cs := &ConsensusState{
		ChainID:             status.NodeInfo.Network,
		Height:              uint64(height),
		AppHash:             appHash,
		CurValidatorSetHash: curValidatorSetHash,
		NextValidatorSet: &tmtypes.ValidatorSet{
			Validators: nextValidatorSet.Validators,
		},
	}
	return cs, nil
}

func QueryTendermintHeader(node rpcclient.Client, height int64) (*Header, error) {
	nextHeight := height + 1

	commit, err := node.Commit(&height)
	if err != nil {
		return nil, err
	}

	validators, err := node.Validators(&height)
	if err != nil {
		return nil, err
	}

	nextvalidators, err := node.Validators(&nextHeight)
	if err != nil {
		return nil, err
	}

	header := &Header{
		SignedHeader:     commit.SignedHeader,
		ValidatorSet:     tmtypes.NewValidatorSet(validators.Validators),
		NextValidatorSet: tmtypes.NewValidatorSet(nextvalidators.Validators),
	}

	return header, nil
}

func QueryKeyWithProof(node rpcclient.Client, key []byte, storeName string, height int64) ([]byte, []byte, []byte, error) {
	opts := rpcclient.ABCIQueryOptions{
		Height: height,
		Prove:  true,
	}

	path := fmt.Sprintf("/store/%s/%s", storeName, "key")
	result, err := node.ABCIQueryWithOptions(path, key, opts)
	if err != nil {
		return nil, nil, nil, err
	}
	proofBytes, err := result.Response.Proof.Marshal()
	if err != nil {
		return nil, nil, nil, err
	}

	return key, result.Response.Value, proofBytes, nil
}
