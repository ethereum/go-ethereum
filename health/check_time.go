package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

var (
	errTimestampTooOld = errors.New("timestamp too old")
)

// checkTime fetches the timestamp of the most recent block and returns an error if it is earlier than 'minTimestamp'.
func checkTime(
	ec ethClient,
	r *http.Request,
	minTimestamp int,
) error {
	i, err := ec.BlockByNumber(context.TODO(), nil)
	if err != nil {
		return err
	}
	timestamp := i.Time()
	if timestamp < uint64(minTimestamp) {
		return fmt.Errorf("%w: got ts: %d, need: %d", errTimestampTooOld, timestamp, minTimestamp)
	}

	return nil
}
