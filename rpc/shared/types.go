package shared

import "encoding/json"

// RPC request
type Request struct {
	Id      interface{}     `json:"id"`
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// RPC response
type Response struct {
	Id      interface{} `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
}

// RPC success response
type SuccessResponse struct {
	Id      interface{} `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
}

// RPC error response
type ErrorResponse struct {
	Id      interface{}  `json:"id"`
	Jsonrpc string       `json:"jsonrpc"`
	Error   *ErrorObject `json:"error"`
}

// RPC error response details
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	// Data    interface{} `json:"data"`
}
