package poseidon

import (
	"fmt"
	"testing"
)

func TestPoseidonCodeHash(t *testing.T) {
	// nil
	got := fmt.Sprintf("%s", CodeHash(nil))
	want := "0x2098f5fb9e239eab3ceac3f27b81e481dc3124d55ffed523a839ee8446b64864"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	// single byte
	got = fmt.Sprintf("%s", CodeHash([]byte{0}))
	want = "0x29f94b67ee4e78b2bb08da025f9943c1201a7af025a27600c2dd0a2e71c7cf8b"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	got = fmt.Sprintf("%s", CodeHash([]byte{1}))
	want = "0x246d3c06960643350a3e2d587fa16315c381635eb5ac1ac4501e195423dbf78e"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	// 32 bytes
	bytes := make([]byte, 32)
	for i := range bytes {
		bytes[i] = 1
	}
	got = fmt.Sprintf("%s", CodeHash(bytes))
	want = "0x0b46d156183dffdbed8e6c6b0af139b95c058e735878ca7f4dca334e0ea8bd20"
	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
