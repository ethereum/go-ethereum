package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const (
	healthHeader     = "X-GETH-HEALTHCHECK"
	query            = "query"
	synced           = "synced"
	minPeerCount     = "min_peer_count"
	checkBlock       = "check_block"
	maxSecondsBehind = "max_seconds_behind"
)

var (
	errCheckDisabled = errors.New("error check disabled")
	errInvalidValue  = errors.New("invalid value provided")
)

type requestBody struct {
	Synced           *bool    `json:"synced"`
	MinPeerCount     *uint    `json:"min_peer_count"`
	CheckBlock       *big.Int `json:"check_block"`
	MaxSecondsBehind *int     `json:"max_seconds_behind"`
}

// processFromHeaders handles requests when 'X-GETH-HEALTHCHECK' header labels are present.
func processFromHeaders(ec ethClient, headers []string, w http.ResponseWriter, r *http.Request) {
	var (
		errCheckSynced  = errCheckDisabled
		errCheckPeer    = errCheckDisabled
		errCheckBlock   = errCheckDisabled
		errCheckSeconds = errCheckDisabled
	)

	for _, header := range headers {
		lHeader := strings.ToLower(header)
		switch {
		case lHeader == synced:
			errCheckSynced = checkSynced(ec, r)
		case strings.HasPrefix(lHeader, minPeerCount):
			peers, err := strconv.Atoi(strings.TrimPrefix(lHeader, minPeerCount))
			if err != nil {
				errCheckPeer = err
				break
			}
			errCheckPeer = checkMinPeers(ec, uint64(peers))
		case strings.HasPrefix(lHeader, checkBlock):
			block, err := strconv.Atoi(strings.TrimPrefix(lHeader, checkBlock))
			if err != nil {
				errCheckBlock = err
				break
			}
			errCheckBlock = checkBlockNumber(ec, big.NewInt(int64(block)))
		case strings.HasPrefix(lHeader, maxSecondsBehind):
			seconds, err := strconv.Atoi(strings.TrimPrefix(lHeader, maxSecondsBehind))
			if err != nil {
				errCheckSeconds = err
				break
			}
			if seconds < 0 {
				errCheckSeconds = errInvalidValue
				break
			}
			now := time.Now().Unix()
			errCheckSeconds = checkTime(ec, r, int(now)-seconds)
		}
	}

	reportHealth(nil, errCheckSynced, errCheckPeer, errCheckBlock, errCheckSeconds, w)
}

// processFromBody handles requests when 'X-GETH-HEALTHCHECK' headers are not present.
func processFromBody(ec ethClient, w http.ResponseWriter, r *http.Request) {
	body, errParse := parseHealthCheckBody(r.Body)
	defer r.Body.Close()

	var (
		errCheckSynced  = errCheckDisabled
		errCheckPeer    = errCheckDisabled
		errCheckBlock   = errCheckDisabled
		errCheckSeconds = errCheckDisabled
	)

	if errParse != nil {
		log.Root().Warn("Unable to process healthcheck request", "err", errParse)
	} else {
		if body.Synced != nil {
			errCheckSynced = checkSynced(ec, r)
		}

		if body.MinPeerCount != nil {
			errCheckPeer = checkMinPeers(ec, uint64(*body.MinPeerCount))
		}

		if body.CheckBlock != nil {
			errCheckBlock = checkBlockNumber(ec, body.CheckBlock)
		}

		if body.MaxSecondsBehind != nil {
			seconds := *body.MaxSecondsBehind
			if seconds < 0 {
				errCheckSeconds = errInvalidValue
			} else {
				now := time.Now().Unix()
				errCheckSeconds = checkTime(ec, r, int(now)-seconds)
			}
		}
	}

	err := reportHealth(errParse, errCheckSynced, errCheckPeer, errCheckBlock, errCheckSeconds, w)
	if err != nil {
		log.Root().Warn("Unable to process healthcheck request", "err", err)
	}
}

// reportHealth builds the response body, sets the status code and calls for it to be written.
func reportHealth(errParse, errCheckSynced, errCheckPeer, errCheckBlock, errCheckSeconds error, w http.ResponseWriter) error {
	statusCode := http.StatusOK
	errs := make(map[string]string)

	if shouldChangeStatusCode(errParse) {
		statusCode = http.StatusInternalServerError
	}
	errs[query] = errorStringOrOK(errParse)

	if shouldChangeStatusCode(errCheckSynced) {
		statusCode = http.StatusInternalServerError
	}
	errs[synced] = errorStringOrOK(errCheckSynced)

	if shouldChangeStatusCode(errCheckPeer) {
		statusCode = http.StatusInternalServerError
	}
	errs[minPeerCount] = errorStringOrOK(errCheckPeer)

	if shouldChangeStatusCode(errCheckBlock) {
		statusCode = http.StatusInternalServerError
	}
	errs[checkBlock] = errorStringOrOK(errCheckBlock)

	if shouldChangeStatusCode(errCheckSeconds) {
		statusCode = http.StatusInternalServerError
	}
	errs[maxSecondsBehind] = errorStringOrOK(errCheckSeconds)

	return writeResponse(w, errs, statusCode)
}

// parseHealthCheckBody parses and type checks the request body when 'X-GETH-HEALTHCHECK' headers are not present.
func parseHealthCheckBody(reader io.Reader) (requestBody, error) {
	var body requestBody

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return body, err
	}

	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		return body, err
	}

	return body, nil
}

// writeResponse delivers the status and body to the response writer.
func writeResponse(w http.ResponseWriter, errs map[string]string, statusCode int) error {
	w.WriteHeader(statusCode)

	bodyJson, err := json.Marshal(errs)
	if err != nil {
		return err
	}

	_, err = w.Write(bodyJson)
	if err != nil {
		return err
	}

	return nil
}

// shouldChangeStatusCode returns 'true' if an error exists and is not 'errCheckDisabled'.
func shouldChangeStatusCode(err error) bool {
	return err != nil && !errors.Is(err, errCheckDisabled)
}

// errorStringOrOK returns "OK", "DISABLED" or the error message based on the output of the check.
func errorStringOrOK(err error) string {
	if err == nil {
		return "OK"
	}

	if errors.Is(err, errCheckDisabled) {
		return "DISABLED"
	}

	return fmt.Sprintf("ERROR: %v", err)
}
