package health

import (
	"context"
	"errors"
	"fmt"
)

var (
	errNotEnoughPeers = errors.New("not enough peers")
)

// checkMinPeers returns 'errNotEnoughPeers' if the current peer count its lower than 'minPeerCount'
func checkMinPeers(ec ethClient, minPeerCount uint64) error {
	peerCount, err := ec.PeerCount(context.TODO())
	if err != nil {
		return err
	}
	if peerCount < minPeerCount {
		return fmt.Errorf("%w: %d (minimum %d)", errNotEnoughPeers, peerCount, minPeerCount)
	}
	return nil
}
