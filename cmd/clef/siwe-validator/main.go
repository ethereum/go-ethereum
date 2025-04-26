package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read stdin:", err)
		os.Exit(1)
	}

	lines := strings.Split(string(raw), "\n")

	if len(lines) < 8 {
		fmt.Fprintln(os.Stderr, "invalid message: not enough lines")
		os.Exit(1)
	}

	domainLine := lines[0]
	addressLine := lines[1]
	uriLine := lines[3]
	versionLine := lines[4]
	chainIDLine := lines[5]
	nonceLine := lines[6]
	issuedAtLine := lines[7]

	if !strings.Contains(domainLine, "localhost:3000") {
		fmt.Fprintf(os.Stderr, "domain mismatch: %s\n", domainLine)
		os.Exit(1)
	}

	if !strings.HasPrefix(addressLine, "0x") {
		fmt.Fprintf(os.Stderr, "invalid address: %s\n", addressLine)
		os.Exit(1)
	}

	if !strings.HasPrefix(uriLine, "URI: https://localhost:3000") {
		fmt.Fprintf(os.Stderr, "uri mismatch: %s\n", uriLine)
		os.Exit(1)
	}

	if !strings.Contains(versionLine, "Version: 1") {
		fmt.Fprintf(os.Stderr, "version mismatch: %s\n", versionLine)
		os.Exit(1)
	}

	if !strings.Contains(chainIDLine, "ChainID: 1") {
		fmt.Fprintf(os.Stderr, "chainID mismatch: %s\n", chainIDLine)
		os.Exit(1)
	}

	if !strings.Contains(nonceLine, "Nonce:") {
		fmt.Fprintf(os.Stderr, "nonce missing: %s\n", nonceLine)
		os.Exit(1)
	}

	if !strings.Contains(issuedAtLine, "Issued At:") {
		fmt.Fprintf(os.Stderr, "issued at missing: %s\n", issuedAtLine)
		os.Exit(1)
	}

	// 全部檢查通過
	os.Exit(0)
}
