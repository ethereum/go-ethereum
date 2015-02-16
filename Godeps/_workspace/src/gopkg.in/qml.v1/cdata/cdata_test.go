package cdata

import (
	"runtime"
	"sync"
	"testing"
)

type refPair struct {
	ref1, ref2 uintptr
}

func TestRef(t *testing.T) {
	const N = 10
	runtime.LockOSThread()
	exit := sync.WaitGroup{}
	exit.Add(1)
	defer exit.Done()
	wg := sync.WaitGroup{}
	wg.Add(N)
	ch := make(chan refPair)
	for i := 0; i < N; i++ {
		go func() {
			runtime.LockOSThread()
			wg.Done()
			ch <- refPair{Ref(), Ref()}
			exit.Wait()
		}()
	}
	wg.Wait()
	refs := make(map[uintptr]bool)
	for i := 0; i < N; i++ {
		pair := <-ch
		if pair.ref1 != pair.ref2 {
			t.Fatalf("found inconsistent ref: %d != %d", pair.ref1, pair.ref2)
		}
		if refs[pair.ref1] {
			t.Fatalf("found duplicated ref: %d", pair.ref1)
		}
		refs[pair.ref1] = true
	}
}
