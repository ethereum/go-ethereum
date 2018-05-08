package adapters

import (
	"bytes"
	"testing"
	"time"
)

func TestFindWSAddr(t *testing.T) {
	line := `t=2018-05-02T19:00:45+0200 lvl=info msg="WebSocket endpoint opened"  node.id=26c65a606d1125a44695bc08573190d047152b6b9a776ccbbe593e90f91444d9c1ebdadac6a775ad9fdd0923468a1d698ed3a842c1fb89c1bc0f9d4801f8c39c url=ws://127.0.0.1:59975`
	buf := bytes.NewBufferString(line)
	got, err := findWSAddr(buf, 10*time.Second)
	if err != nil {
		t.Fatalf("Failed to find addr: %v", err)
	}
	expected := `ws://127.0.0.1:59975`

	if got != expected {
		t.Fatalf("Expected to get '%s', but got '%s'", expected, got)
	}
}
