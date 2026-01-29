// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// Build script for XDC Network. This provides XDC-specific build targets
// in addition to the standard go-ethereum build system.

//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	// Build flags
	buildTags    string
	ldflags      string
	race         bool
	verbose      bool
	crossCompile string

	// XDC specific
	enableXDCx    bool
	enableLending bool
	staticLink    bool
)

func init() {
	flag.StringVar(&buildTags, "tags", "", "Build tags to use")
	flag.StringVar(&ldflags, "ldflags", "", "Additional ldflags")
	flag.BoolVar(&race, "race", false, "Enable race detector")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.StringVar(&crossCompile, "cross", "", "Cross compile target (e.g., linux/amd64)")
	flag.BoolVar(&enableXDCx, "xdcx", true, "Enable XDCx DEX support")
	flag.BoolVar(&enableLending, "lending", true, "Enable lending support")
	flag.BoolVar(&staticLink, "static", false, "Static linking")
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		fmt.Println("Usage: go run ci_xdc.go [flags] <target>")
		fmt.Println("\nTargets:")
		fmt.Println("  build     - Build XDC binary")
		fmt.Println("  test      - Run tests")
		fmt.Println("  lint      - Run linters")
		fmt.Println("  docker    - Build Docker image")
		fmt.Println("  clean     - Clean build artifacts")
		fmt.Println("  install   - Install XDC binary")
		os.Exit(1)
	}

	target := args[0]
	switch target {
	case "build":
		buildXDC()
	case "test":
		runTests()
	case "lint":
		runLint()
	case "docker":
		buildDocker()
	case "clean":
		clean()
	case "install":
		installXDC()
	default:
		fmt.Printf("Unknown target: %s\n", target)
		os.Exit(1)
	}
}

func buildXDC() {
	fmt.Println("Building XDC...")

	// Set up environment
	env := os.Environ()
	if crossCompile != "" {
		parts := strings.Split(crossCompile, "/")
		if len(parts) == 2 {
			env = append(env, "GOOS="+parts[0])
			env = append(env, "GOARCH="+parts[1])
		}
	}

	// Build ldflags
	ldf := buildLdflags()

	// Build tags
	tags := buildTags
	if enableXDCx {
		if tags != "" {
			tags += ","
		}
		tags += "xdcx"
	}
	if enableLending {
		if tags != "" {
			tags += ","
		}
		tags += "lending"
	}

	// Build command
	args := []string{"build", "-o", "build/bin/XDC"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	if ldf != "" {
		args = append(args, "-ldflags", ldf)
	}
	if race {
		args = append(args, "-race")
	}
	if verbose {
		args = append(args, "-v")
	}
	args = append(args, "./cmd/XDC")

	runCommand("go", args, env)
	fmt.Println("Build complete: build/bin/XDC")
}

func buildLdflags() string {
	var parts []string

	// Version info
	gitCommit := getGitCommit()
	gitDate := getGitDate()
	buildDate := time.Now().Format(time.RFC3339)

	parts = append(parts, fmt.Sprintf("-X main.gitCommit=%s", gitCommit))
	parts = append(parts, fmt.Sprintf("-X main.gitDate=%s", gitDate))
	parts = append(parts, fmt.Sprintf("-X main.buildDate=%s", buildDate))

	// Static linking
	if staticLink {
		parts = append(parts, "-linkmode external -extldflags -static")
	}

	if ldflags != "" {
		parts = append(parts, ldflags)
	}

	return strings.Join(parts, " ")
}

func runTests() {
	fmt.Println("Running tests...")

	args := []string{"test"}
	if race {
		args = append(args, "-race")
	}
	if verbose {
		args = append(args, "-v")
	}
	args = append(args, "-coverprofile=coverage.out")
	args = append(args, "./...")

	runCommand("go", args, nil)

	// Run XDPoS specific tests
	fmt.Println("\nRunning XDPoS consensus tests...")
	xdposArgs := []string{"test"}
	if verbose {
		xdposArgs = append(xdposArgs, "-v")
	}
	xdposArgs = append(xdposArgs, "./consensus/XDPoS/...")
	runCommand("go", xdposArgs, nil)
}

func runLint() {
	fmt.Println("Running linters...")

	// Check if golangci-lint is installed
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println("Installing golangci-lint...")
		runCommand("go", []string{"install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"}, nil)
	}

	runCommand("golangci-lint", []string{"run", "--timeout", "10m"}, nil)
}

func buildDocker() {
	fmt.Println("Building Docker image...")

	args := []string{"build", "-t", "xinfin/xdc:latest", "."}
	runCommand("docker", args, nil)
}

func clean() {
	fmt.Println("Cleaning build artifacts...")

	os.RemoveAll("build/bin")
	os.Remove("coverage.out")

	fmt.Println("Clean complete")
}

func installXDC() {
	fmt.Println("Installing XDC...")

	// Build first
	buildXDC()

	// Copy to GOPATH/bin
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}

	binDir := filepath.Join(gopath, "bin")
	os.MkdirAll(binDir, 0755)

	src := "build/bin/XDC"
	dst := filepath.Join(binDir, "XDC")
	if runtime.GOOS == "windows" {
		src += ".exe"
		dst += ".exe"
	}

	copyFile(src, dst)
	fmt.Printf("Installed to %s\n", dst)
}

func runCommand(name string, args []string, env []string) {
	cmd := exec.Command(name, args...)
	if env != nil {
		cmd.Env = env
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if verbose {
		fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Command failed: %v\n", err)
		os.Exit(1)
	}
}

func getGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))[:8]
}

func getGitDate() string {
	cmd := exec.Command("git", "log", "-1", "--format=%ci")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func copyFile(src, dst string) {
	data, err := os.ReadFile(src)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", src, err)
		os.Exit(1)
	}

	if err := os.WriteFile(dst, data, 0755); err != nil {
		fmt.Printf("Failed to write %s: %v\n", dst, err)
		os.Exit(1)
	}
}
