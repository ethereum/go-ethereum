package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	errTimestampTooOld = errors.New("timestamp too old")
)

func checkTime(
	ec *ethclient.Client,
	r *http.Request,
	seconds int,
) error {
	i, err := ec.BlockByNumber(context.TODO(), nil)
	if err != nil {
		return err
	}
	timestamp := i.Time()
	if timestamp < uint64(seconds) {
		return fmt.Errorf("%w: got ts: %d, need: %d", errTimestampTooOld, timestamp, seconds)
	}

	return nil
}
