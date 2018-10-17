package ethgraphql

import (
	"testing"
)

func TestBuildSchema(t *testing.T) {
	// Make sure the schema can be parsed and matched up to the object model.
	_, err := NewHandler(nil)
	if err != nil {
		t.Errorf("Could not construct GraphQL handler: %v", err)
	}
}
