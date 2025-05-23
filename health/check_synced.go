package health

import (
	"context"
	"errors"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

var (
	errNotSynced = errors.New("not synced")
)

// checkSynced returns 'errNotSynced' if the node is in the syncing state.
func checkSynced(ec ethClient, r *http.Request) error {
	i, err := ec.SyncProgress(context.TODO())
	if err != nil {
		log.Root().Warn("Unable to check sync status for healthcheck", "err", err.Error())
		return err
	}
	if i == nil {
		return nil
	}

	return errNotSynced
}
