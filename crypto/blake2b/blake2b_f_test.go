package blake2b

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestF(t *testing.T) {
	for i, test := range testVectorsF {
		t.Run(fmt.Sprintf("test vector %v", i), func(t *testing.T) {
			//toEthereumTestCase(test)

			h := test.hIn
			F(&h, test.m, test.c, test.f, test.rounds)

			if !reflect.DeepEqual(test.hOut, h) {
				t.Errorf("Unexpected result\nExpected: [%#x]\nActual:   [%#x]\n", test.hOut, h)
			}
		})
	}
}

type testVector struct {
	hIn    [8]uint64
	m      [16]uint64
	c      [2]uint64
	f      bool
	rounds uint32
	hOut   [8]uint64
}

// https://tools.ietf.org/html/rfc7693#appendix-A
func randomF(r *rand.Rand) (h [8]uint64, m [16]uint64, c [2]uint64, final bool) {
	for i := range h {
		h[i] = r.Uint64()
	}
	for i := range m {
		m[i] = r.Uint64()
	}
	c[0], c[1] = r.Uint64(), r.Uint64()
	return h, m, c, r.Intn(2) == 0
}

func TestFChunkedMatchesGeneric(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	rounds := []uint32{
		maxAsmRounds - 1, maxAsmRounds, maxAsmRounds + 1,
		2*maxAsmRounds - 1, 2 * maxAsmRounds, 2*maxAsmRounds + 1,
		3*maxAsmRounds + 7, 100003,
	}
	for _, n := range rounds {
		for trial := range 4 {
			h, m, c, final := randomF(r)
			var flag uint64
			if final {
				flag = 0xFFFFFFFFFFFFFFFF
			}
			want := h
			fGeneric(&want, &m, c[0], c[1], flag, uint64(n))

			got := h
			F(&got, m, c, final, n)

			if got != want {
				t.Fatalf("rounds=%d trial=%d: got %#x, want %#x", n, trial, got, want)
			}
		}
	}
}

// gcWait reports the longest a GC had to wait while work ran in another
// goroutine. Work the runtime cannot preempt shows up here directly.
func gcWait(work func()) time.Duration {
	done := make(chan struct{})
	go func() { defer close(done); work() }()
	var worst time.Duration
	for {
		select {
		case <-done:
			return worst
		default:
		}
		start := time.Now()
		runtime.GC()
		worst = max(worst, time.Since(start))
	}
}

// The rounds argument of the F precompile comes from calldata and is priced at
// one gas per round, so a single transaction can ask for tens of millions of
// rounds. Assembly is never preemptible, so an unchunked call blocks every
// stop-the-world for as long as it runs.
//
// The budget is relative to fGeneric, which computes the same thing in Go and is
// always preemptible: a slow or loaded runner moves both numbers together and
// only a real regression separates them.
func TestFLongRoundsIsPreemptible(t *testing.T) {
	if testing.Short() || runtime.GOMAXPROCS(0) < 2 {
		t.Skip("runs millions of rounds, and needs GOMAXPROCS >= 2 to see a stalled GC")
	}
	if !useAVX2 && !useAVX && !useSSE4 {
		t.Skip("no assembly on this machine, so the round loop is already preemptible Go")
	}

	var h [8]uint64
	var m [16]uint64
	var c [2]uint64
	const rounds = 8_000_000

	floor := gcWait(func() { fGeneric(&h, &m, c[0], c[1], 0, rounds) })
	got := gcWait(func() { F(&h, m, c, false, rounds) })

	t.Logf("worst GC wait: F %v, pure-Go fGeneric %v", got, floor)
	if got > 5*time.Millisecond && got > 4*floor {
		t.Fatalf("a GC waited %v on F against %v on the pure-Go path: "+
			"the assembly round loop is running unbounded", got, floor)
	}
}

var testVectorsF = []testVector{
	{
		hIn: [8]uint64{
			0x6a09e667f2bdc948, 0xbb67ae8584caa73b,
			0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
			0x510e527fade682d1, 0x9b05688c2b3e6c1f,
			0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
		},
		m: [16]uint64{
			0x0000000000636261, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000, 0x0000000000000000, 0x0000000000000000,
			0x0000000000000000,
		},
		c:      [2]uint64{3, 0},
		f:      true,
		rounds: 12,
		hOut: [8]uint64{
			0x0D4D1C983FA580BA, 0xE9F6129FB697276A, 0xB7C45A68142F214C,
			0xD1A2FFDB6FBB124B, 0x2D79AB2A39C5877D, 0x95CC3345DED552C2,
			0x5A92F1DBA88AD318, 0x239900D4ED8623B9,
		},
	},
}
