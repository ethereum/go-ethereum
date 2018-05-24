package bind

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestUnkownEventName(t *testing.T) {
	errorPrefix := "abi: event"
	c := &BoundContract{}
	c.abi.Events = make(map[string]abi.Event)
	c.abi.Events["event1"] = abi.Event{}
	if _, _, err := c.FilterLogs(nil, "event_not_exist"); err == nil || !strings.HasPrefix(err.Error(), errorPrefix) {
		t.Fatal("should report error if event not found")
	}

	if _, _, err := c.WatchLogs(nil, "event_not_exist"); err == nil || !strings.HasPrefix(err.Error(), errorPrefix) {
		t.Fatal("should report error if event not found")
	}

	var v interface{}
	elog := types.Log{}
	if err := c.UnpackLog(v, "event_not_exist", elog); err == nil || !strings.HasPrefix(err.Error(), errorPrefix) {
		t.Fatal("should report error if event not found")
	}
}
