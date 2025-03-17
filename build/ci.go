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

	lint           -- runs certain pre-selected linters
	check_tidy     -- verifies that everything is 'go mod tidy'-ed
	check_generate -- verifies that everything is 'go generate'-ed

	install    [ -arch architecture ] [ -cc compiler ] [ packages... ]                          -- builds packages and executables
	test       [ -coverage ] [ packages... ]                                                    -- runs the tests
	importkeys                                                                                  -- imports signing keys from env
	xgo        [ -alltools ] [ options ]                                                        -- cross builds according to options

For all commands, -n prevents execution of external programs (dry run mode).
*/
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/internal/build"
)

var (
	// Files that end up in the geth-alltools*.zip archive.
	allToolsArchiveFiles = []string{
		"COPYING",
		executablePath("abigen"),
		executablePath("bootnode"),
		executablePath("evm"),
		executablePath("geth"),
		executablePath("puppeth"),
		executablePath("rlpdump"),
		executablePath("swarm"),
		executablePath("wnode"),
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

	if _, err := os.Stat(filepath.Join("build", "ci.go")); os.IsNotExist(err) {
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
	case "check_tidy":
		doCheckTidy()
	case "check_generate":
		doCheckGenerate()
	case "xgo":
		doXgo(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

// Compiling

func doInstall(cmdline []string) {
	var (
		arch = flag.String("arch", "", "Architecture to cross build for")
		cc   = flag.String("cc", "", "C compiler to cross build with")
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Check Go version. People regularly open issues about compilation
	// failure with outdated Go. This should save them the trouble.
	if !strings.Contains(runtime.Version(), "devel") {
		// Figure out the minor version number since we can't textually compare (1.10 < 1.9)
		var minor int
		fmt.Sscanf(strings.TrimPrefix(runtime.Version(), "go1."), "%d", &minor)

		if minor < 23 {
			log.Println("You have Go version", runtime.Version())
			log.Println("XDC requires at least Go version 1.23 and cannot")
			log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
			os.Exit(1)
		}
	}
	// Compile packages given as arguments, or everything if there are no arguments.
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}
	// packages = build.ExpandPackagesNoVendor(packages)

	if *arch == "" || *arch == runtime.GOARCH {
		goinstall := goTool("install", buildFlags(env)...)
		goinstall.Args = append(goinstall.Args, "-v")
		goinstall.Args = append(goinstall.Args, packages...)
		build.MustRun(goinstall)
		return
	}
	// If we are cross compiling to ARMv5 ARMv6 or ARMv7, clean any previous builds
	if *arch == "arm" {
		os.RemoveAll(filepath.Join(runtime.GOROOT(), "pkg", runtime.GOOS+"_arm"))
		for _, path := range filepath.SplitList(build.GOPATH()) {
			os.RemoveAll(filepath.Join(path, "pkg", runtime.GOOS+"_arm"))
		}
	}
	// Seems we are cross compiling, work around forbidden GOBIN
	goinstall := goToolArch(*arch, *cc, "install", buildFlags(env)...)
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, []string{"-buildmode", "archive"}...)
	goinstall.Args = append(goinstall.Args, packages...)
	build.MustRun(goinstall)

	if cmds, err := os.ReadDir("cmd"); err == nil {
		for _, cmd := range cmds {
			pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join(".", "cmd", cmd.Name()), nil, parser.PackageClauseOnly)
			if err != nil {
				log.Fatal(err)
			}
			for name := range pkgs {
				if name == "main" {
					gobuild := goToolArch(*arch, *cc, "build", buildFlags(env)...)
					gobuild.Args = append(gobuild.Args, "-v")
					gobuild.Args = append(gobuild.Args, []string{"-o", executablePath(cmd.Name())}...)
					gobuild.Args = append(gobuild.Args, "."+string(filepath.Separator)+filepath.Join("cmd", cmd.Name()))
					build.MustRun(gobuild)
					break
				}
			}
		}
	}
}

func buildFlags(env build.Environment) (flags []string) {
	var ld []string
	if env.Commit != "" {
		ld = append(ld, "-X", "main.gitCommit="+env.Commit)
	}
	if runtime.GOOS == "darwin" {
		ld = append(ld, "-s")
	}

	if len(ld) > 0 {
		flags = append(flags, "-ldflags", strings.Join(ld, " "))
	}
	return flags
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	return goToolArch(runtime.GOARCH, os.Getenv("CC"), subcmd, args...)
}

func goToolArch(arch string, cc string, subcmd string, args ...string) *exec.Cmd {
	cmd := build.GoTool(subcmd, args...)
	cmd.Env = []string{"GOPATH=" + build.GOPATH()}
	if arch == "" || arch == runtime.GOARCH {
		cmd.Env = append(cmd.Env, "GOBIN="+GOBIN)
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		cmd.Env = append(cmd.Env, "GOARCH="+arch)
	}
	if cc != "" {
		cmd.Env = append(cmd.Env, "CC="+cc)
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") || strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

// Running The Tests
//
// "tests" also includes static analysis tools such as vet.
func doTest(cmdline []string) {
	coverage := flag.Bool("coverage", false, "Whether to record code coverage")
	verbose := flag.Bool("v", false, "Whether to log verbosely")
	quick := flag.Bool("quick", false, "Whether to skip long time test")
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	packages := []string{"./..."} // if a package has no test files, the lines in that package are not added to the total line count
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	} else {
		// added all files in all packages (except vendor) to coverage report files count, even there is no test file in the package
		packages = build.ExpandPackages(packages, *quick)
	}

	// Run analysis tools before the tests.
	// build.MustRun(goTool("vet", packages...))

	// Run the actual tests.
	// gotest := goTool("test", buildFlags(env)...)
	// Test a single package at a time. CI builders are slow
	// and some tests run into timeouts under load.
	gotest := goTool("test", buildFlags(env)...)
	gotest.Args = append(gotest.Args, "-p", "1")
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover", "-coverprofile=coverage.txt")
	}
	if *verbose {
		gotest.Args = append(gotest.Args, "-v")
	}

	gotest.Args = append(gotest.Args, packages...)
	build.MustRun(gotest)
}

// doCheckTidy assets that the Go modules files are tidied already.
func doCheckTidy() {
	targets := []string{"go.mod", "go.sum"}

	hashes, err := build.HashFiles(targets)
	if err != nil {
		log.Fatalf("failed to hash go.mod/go.sum: %v", err)
	}
	build.MustRun(new(build.GoToolchain).Go("mod", "tidy"))

	tidied, err := build.HashFiles(targets)
	if err != nil {
		log.Fatalf("failed to rehash go.mod/go.sum: %v", err)
	}
	if updates := build.DiffHashes(hashes, tidied); len(updates) > 0 {
		log.Fatalf("files changed on running 'go mod tidy': %v", updates)
	}
	fmt.Println("No untidy module files detected.")
}

// doCheckGenerate ensures that re-generating generated files does not cause
// any mutations in the source file tree.
func doCheckGenerate() {
	var (
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching binaries.")
	)
	// Compute the origin hashes of all the files
	var hashes map[string][32]byte

	var err error
	hashes, err = build.HashFolder(".", []string{"tests/testdata", "build/cache"})
	if err != nil {
		log.Fatal("Error computing hashes", "err", err)
	}
	// Run any go generate steps we might be missing
	var (
		protocPath      = downloadProtoc(*cachedir)
		protocGenGoPath = downloadProtocGenGo(*cachedir)
	)
	c := new(build.GoToolchain).Go("generate", "./...")
	pathList := []string{filepath.Join(protocPath, "bin"), protocGenGoPath, os.Getenv("PATH")}
	c.Env = append(c.Env, "PATH="+strings.Join(pathList, string(os.PathListSeparator)))
	build.MustRun(c)

	// Check if generate file hashes have changed
	generated, err := build.HashFolder(".", []string{"tests/testdata", "build/cache"})
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
	fmt.Println("No stale files detected.")
}

// doLint runs golangci-lint on requested packages.
func doLint(cmdline []string) {
	var (
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching golangci-lint binary.")
	)
	flag.CommandLine.Parse(cmdline)
	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}

	linter := downloadLinter(*cachedir)
	lflags := []string{"run", "--config", ".golangci.yml"}
	build.MustRunCommandWithOutput(linter, append(lflags, packages...)...)
	fmt.Println("You have achieved perfection.")
}

// downloadLinter downloads and unpacks golangci-lint.
func downloadLinter(cachedir string) string {
	csdb := build.MustLoadChecksums("build/checksums.txt")
	version, err := build.Version(csdb, "golangci")
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
	url := fmt.Sprintf("https://github.com/golangci/golangci-lint/releases/download/v%s/%s%s", version, base, ext)
	archivePath := filepath.Join(cachedir, base+ext)
	if err := csdb.DownloadFile(url, archivePath); err != nil {
		log.Fatal(err)
	}
	if err := build.ExtractArchive(archivePath, cachedir); err != nil {
		log.Fatal(err)
	}
	return filepath.Join(cachedir, base, "golangci-lint")
}

// downloadProtocGenGo downloads protoc-gen-go, which is used by protoc
// in the generate command.  It returns the full path of the directory
// containing the 'protoc-gen-go' executable.
func downloadProtocGenGo(cachedir string) string {
	csdb := build.MustLoadChecksums("build/checksums.txt")
	version, err := build.Version(csdb, "protoc-gen-go")
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

	url := fmt.Sprintf("https://github.com/protocolbuffers/protobuf-go/releases/download/v%s/%s", version, archiveName)

	archivePath := path.Join(cachedir, archiveName)
	if err := csdb.DownloadFile(url, archivePath); err != nil {
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

// downloadProtoc downloads the prebuilt protoc binary used to lint generated
// files as a CI step.  It returns the full path to the directory containing
// the protoc executable.
func downloadProtoc(cachedir string) string {
	csdb := build.MustLoadChecksums("build/checksums.txt")
	version, err := build.Version(csdb, "protoc")
	if err != nil {
		log.Fatal(err)
	}
	baseName, err := protocArchiveBaseName()
	if err != nil {
		log.Fatal(err)
	}

	fileName := fmt.Sprintf("protoc-%s-%s", version, baseName)
	archiveFileName := fileName + ".zip"
	url := fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/%s", version, archiveFileName)
	archivePath := filepath.Join(cachedir, archiveFileName)

	if err := csdb.DownloadFile(url, archivePath); err != nil {
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

// Cross compilation
func doXgo(cmdline []string) {
	var (
		alltools = flag.Bool("alltools", false, `Flag whether we're building all known tools, or only on in particular`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Make sure xgo is available for cross compilation
	gogetxgo := goTool("get", "github.com/karalabe/xgo")
	build.MustRun(gogetxgo)

	// If all tools building is requested, build everything the builder wants
	args := append(buildFlags(env), flag.Args()...)

	if *alltools {
		args = append(args, []string{"--dest", GOBIN}...)
		for _, res := range allToolsArchiveFiles {
			if strings.HasPrefix(res, GOBIN) {
				// Binary tool found, cross build it explicitly
				args = append(args, "./"+filepath.Join("cmd", filepath.Base(res)))
				xgo := xgoTool(args)
				build.MustRun(xgo)
				args = args[:len(args)-1]
			}
		}
		return
	}
	// Otherwise xxecute the explicit cross compilation
	path := args[len(args)-1]
	args = append(args[:len(args)-1], []string{"--dest", GOBIN, path}...)

	xgo := xgoTool(args)
	build.MustRun(xgo)
}

func xgoTool(args []string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(GOBIN, "xgo"), args...)
	cmd.Env = []string{
		"GOPATH=" + build.GOPATH(),
		"GOBIN=" + GOBIN,
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") || strings.HasPrefix(e, "GOBIN=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}
