// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

//go:build none
// +build none

/*
The ci command is called from Continuous Integration scripts.

Usage: go run build/ci.go <command> <command flags/arguments>

Available commands are:

	lint      -- runs certain pre-selected linters
	tidy      -- verifies that everything is 'go mod tidy'-ed
	generate  -- verifies that everything is 'go generate'-ed
	baddeps   -- verifies that certain dependencies are avoided

	install    [ -arch architecture ] [ -cc compiler ] [ packages... ]  -- builds packages and executables
	test       [ -coverage ] [ packages... ]                            -- runs the tests
	importkeys                                                          -- imports signing keys from env
	xgo        [ -alltools ] [ options ]                                -- cross builds according to options

For all commands, -n prevents execution of external programs (dry run mode).
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/internal/build"
	"github.com/XinFinOrg/XDPoSChain/internal/download"
)

var (
	goModules = []string{
		".",
	}

	// Files that end up in the geth-alltools*.zip archive.
	allToolsArchiveFiles = []string{
		"COPYING",
		executablePath("abigen"),
		executablePath("bootnode"),
		executablePath("ethkey"),
		executablePath("evm"),
		executablePath("p2psim"),
		executablePath("puppeth"),
		executablePath("rlpdump"),
		executablePath("XDC"),
	}
)

var GOBIN, _ = filepath.Abs(filepath.Join("build", "bin"))

func executablePath(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(GOBIN, name)
}

func main() {
	log.SetFlags(log.Lshortfile)

	if !common.FileExist(filepath.Join("build", "ci.go")) {
		log.Fatal("this script must be run from the root of the repository")
	}
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}
	switch os.Args[1] {
	case "install":
		doInstall(os.Args[2:])
	case "test":
		doTest(os.Args[2:])
	case "lint":
		doLint(os.Args[2:])
	case "tidy":
		doTidy()
	case "generate":
		doGenerate()
	case "baddeps":
		doBadDeps()
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

// Compiling

func doInstall(cmdline []string) {
	var (
		dlgo       = flag.Bool("dlgo", false, "Download Go and build with it")
		arch       = flag.String("arch", "", "Architecture to cross build for")
		cc         = flag.String("cc", "", "C compiler to cross build with")
		staticlink = flag.Bool("static", false, "Create statically-linked executable")
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Configure the toolchain.
	tc := build.GoToolchain{GOARCH: *arch, CC: *cc}
	if *dlgo {
		csdb := download.MustLoadChecksums("build/checksums.txt")
		tc.Root = build.DownloadGo(csdb)
	}
	// Disable CLI markdown doc generation in release builds.
	buildTags := []string{"urfave_cli_no_docs"}

	// Configure the build.
	gobuild := tc.Go("build", buildFlags(env, *staticlink, buildTags)...)

	// Show packages during build.
	gobuild.Args = append(gobuild.Args, "-v")

	// Now we choose what we're even building.
	// Default: collect all 'main' packages in cmd/ and build those.
	packages := flag.Args()
	if len(packages) == 0 {
		// NOTE: to collect all main packages, use:
		// packages = build.FindMainPackages(&tc, "./...")
		packages = build.FindMainPackages(&tc, "./cmd/...")
	}

	// Do the build!
	for _, pkg := range packages {
		args := slices.Clone(gobuild.Args)
		args = append(args, "-o", executablePath(path.Base(pkg)))
		args = append(args, pkg)
		build.MustRun(&exec.Cmd{Path: gobuild.Path, Args: args, Env: gobuild.Env})
	}
}

// buildFlags returns the go tool flags for building.
func buildFlags(env build.Environment, staticLinking bool, buildTags []string) (flags []string) {
	var ld []string
	// See https://github.com/golang/go/issues/33772#issuecomment-528176001
	// We need to set --buildid to the linker here, and also pass --build-id to the
	// cgo-linker further down.
	ld = append(ld, "--buildid=none")
	if env.Commit != "" {
		ld = append(ld, "-X", "github.com/XinFinOrg/XDPoSChain/internal/version.gitCommit="+env.Commit)
		ld = append(ld, "-X", "github.com/XinFinOrg/XDPoSChain/internal/version.gitDate="+env.Date)
	}
	// Strip DWARF on darwin. This used to be required for certain things,
	// and there is no downside to this, so we just keep doing it.
	if runtime.GOOS == "darwin" {
		ld = append(ld, "-s")
	}
	if runtime.GOOS == "linux" {
		// Enforce the stacksize to 8M, which is the case on most platforms apart from
		// alpine Linux.
		// See https://sourceware.org/binutils/docs-2.23.1/ld/Options.html#Options
		// regarding the options --build-id=none and --strip-all. It is needed for
		// reproducible builds; removing references to temporary files in C-land, and
		// making build-id reproducibly absent.
		extld := []string{"-Wl,-z,stack-size=0x800000,--build-id=none,--strip-all"}
		if staticLinking {
			extld = append(extld, "-static")
			// Under static linking, use of certain glibc features must be
			// disabled to avoid shared library dependencies.
			buildTags = append(buildTags, "osusergo", "netgo")
		}
		ld = append(ld, "-extldflags", "'"+strings.Join(extld, " ")+"'")
	}
	if len(ld) > 0 {
		flags = append(flags, "-ldflags", strings.Join(ld, " "))
	}
	if len(buildTags) > 0 {
		flags = append(flags, "-tags", strings.Join(buildTags, ","))
	}
	// We use -trimpath to avoid leaking local paths into the built executables.
	flags = append(flags, "-trimpath")
	return flags
}

// Running The Tests
//
// "tests" also includes static analysis tools such as vet.

func doTest(cmdline []string) {
	var (
		dlgo     = flag.Bool("dlgo", false, "Download Go and build with it")
		arch     = flag.String("arch", "", "Run tests for given architecture")
		cc       = flag.String("cc", "", "Sets C compiler binary")
		coverage = flag.Bool("coverage", false, "Whether to record code coverage")
		verbose  = flag.Bool("v", false, "Whether to log verbosely")
		race     = flag.Bool("race", false, "Execute the race detector")
		short    = flag.Bool("short", false, "Pass the 'short'-flag to go test")
		threads  = flag.Int("p", 1, "Number of CPU threads to use for testing")
		quick    = flag.Bool("quick", false, "Whether to skip long time test")
		failfast = flag.Bool("failfast", false, "Do not start new tests after the first test failure")
	)
	flag.CommandLine.Parse(cmdline)

	// Load checksums file (needed for both spec tests and dlgo)
	csdb := download.MustLoadChecksums("build/checksums.txt")

	// Configure the toolchain.
	tc := build.GoToolchain{GOARCH: *arch, CC: *cc}
	if *dlgo {
		tc.Root = build.DownloadGo(csdb)
	}

	gotest := tc.Go("test")

	// CI needs a bit more time for the statetests (default 45m).
	gotest.Args = append(gotest.Args, "-timeout=45m")

	// Enable integration-tests
	gotest.Args = append(gotest.Args, "-tags=integrationtests")

	// Test a single package at a time. CI builders are slow
	// and some tests run into timeouts under load.
	gotest.Args = append(gotest.Args, "-p", fmt.Sprintf("%d", *threads))
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover")
	}
	if *verbose {
		gotest.Args = append(gotest.Args, "-v")
	}
	if *failfast {
		gotest.Args = append(gotest.Args, "-failfast")
	}
	if *race {
		gotest.Args = append(gotest.Args, "-race")
	}
	if *short {
		gotest.Args = append(gotest.Args, "-short")
	}

	packages := flag.CommandLine.Args()
	if len(packages) > 0 {
		if *quick {
			packages = filterPackages(packages)
		}
		gotest.Args = append(gotest.Args, packages...)
		build.MustRun(gotest)
		return
	}

	// No packages specified, run all tests for all modules.
	if *quick {
		packages = filterPackages(build.FindAllPackages(&tc))
	} else {
		packages = []string{"./..."}
	}
	gotest.Args = append(gotest.Args, packages...)
	for _, mod := range goModules {
		test := *gotest
		test.Dir = mod
		build.MustRun(&test)
	}
}

// filterPackages removes time-consuming packages.
func filterPackages(packages []string) []string {
	var filtered []string

	for _, pkg := range packages {
		if strings.Contains(pkg, "/consensus/tests/engine_v2_tests") {
			continue
		}
		filtered = append(filtered, pkg)
	}

	return filtered
}

// doTidy runs go mod tidy check.
func doTidy() {
	var tc = new(build.GoToolchain)

	for _, mod := range goModules {
		tidy := tc.Go("mod", "tidy", "-diff")
		tidy.Dir = mod
		build.MustRun(tidy)
	}
	fmt.Println("No untidy module files detected.")
}

// doGenerate ensures that re-generating generated files does not cause
// any mutations in the source file tree.
func doGenerate() {
	var (
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching binaries.")
		tc       = new(build.GoToolchain)
	)

	// Run any go generate steps we might be missing
	var (
		protocPath      = downloadProtoc(*cachedir)
		protocGenGoPath = downloadProtocGenGo(*cachedir)
	)
	pathList := []string{filepath.Join(protocPath, "bin"), protocGenGoPath, os.Getenv("PATH")}

	excludes := []string{"tests/testdata", "build/cache", ".git"}
	for i := range excludes {
		excludes[i] = filepath.FromSlash(excludes[i])
	}

	for _, mod := range goModules {
		// Compute the origin hashes of all the files
		hashes, err := build.HashFolder(mod, excludes)
		if err != nil {
			log.Fatal("Error computing hashes", "err", err)
		}

		c := tc.Go("generate", "./...")
		c.Env = append(c.Env, "PATH="+strings.Join(pathList, string(os.PathListSeparator)))
		c.Dir = mod
		build.MustRun(c)
		// Check if generate file hashes have changed
		generated, err := build.HashFolder(mod, excludes)
		if err != nil {
			log.Fatalf("Error re-computing hashes: %v", err)
		}
		updates := build.DiffHashes(hashes, generated)
		for _, file := range updates {
			log.Printf("File changed: %s", file)
		}
		if len(updates) != 0 {
			log.Fatal("One or more generated files were updated by running 'go generate ./...'")
		}
	}
	fmt.Println("No stale files detected.")
}

// doBadDeps verifies whether certain unintended dependencies between some
// packages leak into the codebase due to a refactor. This is not an exhaustive
// list, rather something we build up over time at sensitive places.
func doBadDeps() {
	baddeps := [][2]string{
		// Rawdb tends to be a dumping ground for db utils, sometimes leaking the db itself
		{"github.com/XinFinOrg/XDPoSChain/core/rawdb", "github.com/XinFinOrg/XDPoSChain/ethdb/leveldb"},
		{"github.com/XinFinOrg/XDPoSChain/core/rawdb", "github.com/XinFinOrg/XDPoSChain/ethdb/pebbledb"},
	}
	tc := new(build.GoToolchain)

	var failed bool
	for _, rule := range baddeps {
		out, err := tc.Go("list", "-deps", rule[0]).CombinedOutput()
		if err != nil {
			log.Fatalf("Failed to list '%s' dependencies: %v", rule[0], err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.TrimSpace(line) == rule[1] {
				log.Printf("Found bad dependency '%s' -> '%s'", rule[0], rule[1])
				failed = true
			}
		}
	}
	if failed {
		log.Fatalf("Bad dependencies detected.")
	}
	fmt.Println("No bad dependencies detected.")
}

// doLint runs golangci-lint on requested packages.
func doLint(cmdline []string) {
	var (
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching golangci-lint binary.")
	)
	flag.CommandLine.Parse(cmdline)

	linter := downloadLinter(*cachedir)
	linter, err := filepath.Abs(linter)
	if err != nil {
		log.Fatal(err)
	}
	config, err := filepath.Abs(".golangci.yml")
	if err != nil {
		log.Fatal(err)
	}

	lflags := []string{"run", "--config", config}
	packages := flag.CommandLine.Args()
	if len(packages) > 0 {
		build.MustRunCommandWithOutput(linter, append(lflags, packages...)...)
	} else {
		// Run for all modules in workspace.
		for _, mod := range goModules {
			args := append(lflags, "./...")
			lintcmd := exec.Command(linter, args...)
			lintcmd.Dir = mod
			build.MustRunWithOutput(lintcmd)
		}
	}
	fmt.Println("You have achieved perfection.")
}

// downloadLinter downloads and unpacks golangci-lint.
func downloadLinter(cachedir string) string {
	csdb := download.MustLoadChecksums("build/checksums.txt")
	version, err := csdb.FindVersion("golangci")
	if err != nil {
		log.Fatal(err)
	}
	arch := runtime.GOARCH
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	if arch == "arm" {
		arch += "v" + os.Getenv("GOARM")
	}
	base := fmt.Sprintf("golangci-lint-%s-%s-%s", version, runtime.GOOS, arch)
	archivePath := filepath.Join(cachedir, base+ext)
	if err := csdb.DownloadFileFromKnownURL(archivePath); err != nil {
		log.Fatal(err)
	}
	if err := build.ExtractArchive(archivePath, cachedir); err != nil {
		log.Fatal(err)
	}
	return filepath.Join(cachedir, base, "golangci-lint")
}

// protocArchiveBaseName returns the name of the protoc archive file for
// the current system, stripped of version and file suffix.
func protocArchiveBaseName() (string, error) {
	switch runtime.GOOS + "-" + runtime.GOARCH {
	case "windows-amd64":
		return "win64", nil
	case "windows-386":
		return "win32", nil
	case "linux-arm64":
		return "linux-aarch_64", nil
	case "linux-386":
		return "linux-x86_32", nil
	case "linux-amd64":
		return "linux-x86_64", nil
	case "darwin-arm64":
		return "osx-aarch_64", nil
	case "darwin-amd64":
		return "osx-x86_64", nil
	default:
		return "", fmt.Errorf("no prebuilt release of protoc available for this system (os: %s, arch: %s)", runtime.GOOS, runtime.GOARCH)
	}
}

// downloadProtocGenGo downloads protoc-gen-go, which is used by protoc
// in the generate command.  It returns the full path of the directory
// containing the 'protoc-gen-go' executable.
func downloadProtocGenGo(cachedir string) string {
	csdb := download.MustLoadChecksums("build/checksums.txt")
	version, err := csdb.FindVersion("protoc-gen-go")
	if err != nil {
		log.Fatal(err)
	}
	baseName := fmt.Sprintf("protoc-gen-go.v%s.%s.%s", version, runtime.GOOS, runtime.GOARCH)
	archiveName := baseName
	if runtime.GOOS == "windows" {
		archiveName += ".zip"
	} else {
		archiveName += ".tar.gz"
	}

	archivePath := path.Join(cachedir, archiveName)
	if err := csdb.DownloadFileFromKnownURL(archivePath); err != nil {
		log.Fatal(err)
	}
	extractDest := filepath.Join(cachedir, baseName)
	if err := build.ExtractArchive(archivePath, extractDest); err != nil {
		log.Fatal(err)
	}
	extractDest, err = filepath.Abs(extractDest)
	if err != nil {
		log.Fatal("error resolving absolute path for protoc", "err", err)
	}
	return extractDest
}

// downloadProtoc downloads the prebuilt protoc binary used to lint generated
// files as a CI step.  It returns the full path to the directory containing
// the protoc executable.
func downloadProtoc(cachedir string) string {
	csdb := download.MustLoadChecksums("build/checksums.txt")
	version, err := csdb.FindVersion("protoc")
	if err != nil {
		log.Fatal(err)
	}
	baseName, err := protocArchiveBaseName()
	if err != nil {
		log.Fatal(err)
	}

	fileName := fmt.Sprintf("protoc-%s-%s", version, baseName)
	archiveFileName := fileName + ".zip"
	archivePath := filepath.Join(cachedir, archiveFileName)
	if err := csdb.DownloadFileFromKnownURL(archivePath); err != nil {
		log.Fatal(err)
	}
	extractDest := filepath.Join(cachedir, fileName)
	if err := build.ExtractArchive(archivePath, extractDest); err != nil {
		log.Fatal(err)
	}
	extractDest, err = filepath.Abs(extractDest)
	if err != nil {
		log.Fatal("error resolving absolute path for protoc", "err", err)
	}
	return extractDest
}
