package openrpc_test

import (
	"testing"

	"github.com/ubiq/go-ubiq/internal/openrpc"
	"github.com/ubiq/go-ubiq/rpc"
)

func TestDefaultSchema(t *testing.T) {
	if err := rpc.SetDefaultOpenRPCSchemaRaw(openrpc.OpenRPCSchema); err != nil {
		t.Fatal(err)
	}
}
