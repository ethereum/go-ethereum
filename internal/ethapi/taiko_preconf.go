package ethapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/log"
)

type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Jsonrpc string           `json:"jsonrpc"`
	ID      int              `json:"id"`
	Result  *json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error,omitempty"`
}

func forward[T any](forwardURL string, method string, params []interface{}) (*T, error) {
	rpcReq := rpcRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", forwardURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to forward transaction, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp rpcResponse

	// Unmarshal the response into the struct
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, err
	}

	// Check for errors in the response
	if rpcResp.Error != nil {
		err := fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)

		log.Error("forwarded request error", "err", err, "method", method, "params", params)

		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		log.Warn("forwarded request result is nil", "method", method)
		return nil, nil
	}

	// Unmarshal the Result into the desired type
	var result T
	if err := json.Unmarshal(*rpcResp.Result, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
