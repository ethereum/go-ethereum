package pprof

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// Profile generates a pprof.Profile report for the given profile name.
func Profile(profile string, debug, gc int) ([]byte, map[string]string, error) {
	p := pprof.Lookup(profile)
	if p == nil {
		return nil, nil, fmt.Errorf("profile '%s' not found", profile)
	}

	if profile == "heap" && gc > 0 {
		runtime.GC()
	}

	var buf bytes.Buffer
	if err := p.WriteTo(&buf, debug); err != nil {
		return nil, nil, err
	}

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
	}
	if debug != 0 {
		headers["Content-Type"] = "text/plain; charset=utf-8"
	} else {
		headers["Content-Type"] = "application/octet-stream"
		headers["Content-Disposition"] = fmt.Sprintf(`attachment; filename="%s"`, profile)
	}
	return buf.Bytes(), headers, nil
}

// CPUProfile generates a CPU Profile for a given duration
func CPUProfile(ctx context.Context, sec int) ([]byte, map[string]string, error) {
	if sec <= 0 {
		sec = 1
	}

	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		return nil, nil, err
	}

	sleep(ctx, time.Duration(sec)*time.Second)

	pprof.StopCPUProfile()

	return buf.Bytes(),
		map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "application/octet-stream",
			"Content-Disposition":    `attachment; filename="profile"`,
		}, nil
}

// CPUProfile generates a CPU Profile for a given duration
func CPUProfileWithChannel(done chan bool) ([]byte, map[string]string, error) {
	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		return nil, nil, err
	}

	select {
	case <-done:
	case <-time.After(30 * time.Second):
	}

	pprof.StopCPUProfile()

	return buf.Bytes(),
		map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "application/octet-stream",
			"Content-Disposition":    `attachment; filename="profile"`,
		}, nil
}

// Trace runs a trace profile for a given duration
func Trace(ctx context.Context, sec int) ([]byte, map[string]string, error) {
	if sec <= 0 {
		sec = 1
	}

	var buf bytes.Buffer
	if err := trace.Start(&buf); err != nil {
		return nil, nil, err
	}

	sleep(ctx, time.Duration(sec)*time.Second)

	trace.Stop()

	return buf.Bytes(),
		map[string]string{
			"X-Content-Type-Options": "nosniff",
			"Content-Type":           "application/octet-stream",
			"Content-Disposition":    `attachment; filename="trace"`,
		}, nil
}

func sleep(ctx context.Context, d time.Duration) {
	// Sleep until duration is met or ctx is cancelled
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}

func SetMemProfileRate(rate int) {
	runtime.MemProfileRate = rate
}

func SetSetBlockProfileRate(rate int) {
	runtime.SetBlockProfileRate(rate)
}

func StartPProf(address string) {
	log.Info("Starting pprof server", "addr", fmt.Sprintf("http://%s/debug/pprof", address))

	go func() {
		// nolint: gosec
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Error("Failure in running pprof server", "err", err)
		}
	}()
}
