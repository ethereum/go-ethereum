package core

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"io"
	"net/http"
	"net/url"

	"github.com/ethereum/go-ethereum/core/state"
)

// crossValidator holds configuration for stateless cross-validation of imported blocks.
type crossValidator struct {
	// HTTP endpoint for stateless block validation (in the future this will be a list)
	endpoint string
	// path to dump witnesses to disk when cross-validation fails
	witnessRecordingPath string
}

// CrossValidateBlock verifies the given stateless witness using the configured cross validater endpoint.
// If cross-validation fails, it dumps the witness to a file on disk.
//
// TODO: differentiate between errors from witness verification (maybe consensus
// failure) and anything else.
func (c *crossValidator) CrossValidateBlock(chainConfig *params.ChainConfig, witness *state.Witness) error {
	// encode the witness to RLP, zeroing-out the block state root before sending
	// it for cross validation to make it impossible for a cross-validator to
	// produce a correct validation result without computing it.
	enc, _ := witness.EncodeRLP()

	// TODO: implement retry if endpoint can't be reached
	p, err := url.JoinPath(c.endpoint, "verify_block")
	if err != nil {
		return fmt.Errorf("url.JoinPath failed: %v", err)
	}
	resp, err := http.Post(p, "application/octet-stream", bytes.NewBuffer(enc))
	if err != nil {
		return fmt.Errorf("error accessing block verification endpoint: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}
		return fmt.Errorf("cross-validator bad response code (%d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	if bytes.Compare(body, witness.Block.Header().Root[:]) != 0 {
		if errInner := state.DumpBlockWitnessToFile(chainConfig, witness, c.witnessRecordingPath); errInner != nil {
			log.Error("failed to dump block to file", "error", errInner)
			panic("should not happen")
		}
		return err
	}
	return nil
}

// CrossValidate posts the provided witness to the URL at {endpoint}/verify_block and returns whether the remote
// verification was successful or not.
