package pprof

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
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
