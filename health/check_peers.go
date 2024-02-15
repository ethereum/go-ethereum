package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	errNotEnoughPeers = errors.New("not enough peers")
)

func checkMinPeers(ec *ethclient.Client, minPeerCount uint) error {
	peerCount, err := ec.PeerCount(context.TODO())
	if err != nil {
		return err
	}
	if uint64(peerCount) < uint64(minPeerCount) {
		return fmt.Errorf("%w: %d (minimum %d)", errNotEnoughPeers, peerCount, minPeerCount)
	}
	return nil
}
