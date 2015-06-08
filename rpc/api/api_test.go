package api

import (
	"testing"

	"github.com/ethereum/go-ethereum/rpc/codec"
)

func TestParseApiString(t *testing.T) {
	apis, err := ParseApiString("", codec.JSON, nil, nil)
	if err == nil {
		t.Errorf("Expected an err from parsing empty API string but got nil")
	}

	if len(apis) != 0 {
		t.Errorf("Expected 0 apis from empty API string")
	}

	apis, err = ParseApiString("eth", codec.JSON, nil, nil)
	if err != nil {
		t.Errorf("Expected nil err from parsing empty API string but got %v", err)
	}

	if len(apis) != 1 {
		t.Errorf("Expected 1 apis but got %d - %v", apis, apis)
	}

	apis, err = ParseApiString("eth,eth", codec.JSON, nil, nil)
	if err != nil {
		t.Errorf("Expected nil err from parsing empty API string but got \"%v\"", err)
	}

	if len(apis) != 2 {
		t.Errorf("Expected 2 apis but got %d - %v", apis, apis)
	}

	apis, err = ParseApiString("eth,invalid", codec.JSON, nil, nil)
	if err == nil {
		t.Errorf("Expected an err but got no err")
	}

}
