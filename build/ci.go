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

	install    [ -arch architecture ] [ -cc compiler ] [ packages... ]                          -- builds packages and executables
	test       [ -coverage ] [ packages... ]                                                    -- runs the tests
	lint                                                                                        -- runs certain pre-selected linters
	archive    [ -arch architecture ] [ -type zip|tar ] [ -signer key-envvar ] [ -signify key-envvar ] [ -upload dest ] -- archives build artifacts
	importkeys                                                                                  -- imports signing keys from env
	debsrc     [ -signer key-id ] [ -upload dest ]                                              -- creates a debian source package
	nsis                                                                                        -- creates a Windows NSIS installer
	purge      [ -store blobstore ] [ -days threshold ]                                         -- purges old archives from the blobstore

For all commands, -n prevents execution of external programs (dry run mode).
*/
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/cp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/signify"
	"github.com/ethereum/go-ethereum/internal/build"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// Files that end up in the geth*.zip archive.
	gethArchiveFiles = []string{
		"COPYING",
		executablePath("geth"),
	}

	// Files that end up in the geth-alltools*.zip archive.
	allToolsArchiveFiles = []string{
		"COPYING",
		executablePath("abigen"),
		executablePath("bootnode"),
		executablePath("evm"),
		executablePath("geth"),
		executablePath("rlpdump"),
		executablePath("clef"),
	}

	// A debian package is created for all executables listed here.
	debExecutables = []debExecutable{
		{
			BinaryName:  "abigen",
			Description: "Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages.",
		},
		{
			BinaryName:  "bootnode",
			Description: "Ethereum bootnode.",
		},
		{
			BinaryName:  "evm",
			Description: "Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode.",
		},
		{
			BinaryName:  "geth",
			Description: "Ethereum CLI client.",
		},
		{
			BinaryName:  "rlpdump",
			Description: "Developer utility tool that prints RLP structures.",
		},
		{
			BinaryName:  "clef",
			Description: "Ethereum account management tool.",
		},
	}

	// A debian package is created for all executables listed here.
	debEthereum = debPackage{
		Name:        "ethereum",
		Version:     params.Version,
		Executables: debExecutables,
	}

	// Debian meta packages to build and push to Ubuntu PPA
	debPackages = []debPackage{
		debEthereum,
	}

	// Distros for which packages are created
	debDistros = []string{
		"xenial", // 16.04, EOL: 04/2026
		"bionic", // 18.04, EOL: 04/2028
		"focal",  // 20.04, EOL: 04/2030
		"jammy",  // 22.04, EOL: 04/2032
		"noble",  // 24.04, EOL: 04/2034
	}

	// This is where the tests should be unpacked.
	executionSpecTestsDir = "tests/spec-tests"
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
	case "archive":
		doArchive(os.Args[2:])
	case "docker":
		doDocker(os.Args[2:])
	case "debsrc":
		doDebianSource(os.Args[2:])
	case "nsis":
		doWindowsInstaller(os.Args[2:])
	case "purge":
		doPurge(os.Args[2:])
	case "sanitycheck":
		doSanityCheck()
	case "generate":
		doGenerate()
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
		csdb := build.MustLoadChecksums("build/checksums.txt")
		tc.Root = build.DownloadGo(csdb)
	}
	// Disable CLI markdown doc generation in release builds.
	buildTags := []string{"urfave_cli_no_docs"}

	// Enable linking the CKZG library since we can make it work with additional flags.
	if env.UbuntuVersion != "trusty" {
		buildTags = append(buildTags, "ckzg")
	}

	// Configure the build.
	gobuild := tc.Go("build", buildFlags(env, *staticlink, buildTags)...)

	// arm64 CI builders are memory-constrained and can't handle concurrent builds,
	// better disable it. This check isn't the best, it should probably
	// check for something in env instead.
	if env.CI && runtime.GOARCH == "arm64" {
		gobuild.Args = append(gobuild.Args, "-p", "1")
	}
	// We use -trimpath to avoid leaking local paths into the built executables.
	gobuild.Args = append(gobuild.Args, "-trimpath")

	// Show packages during build.
	gobuild.Args = append(gobuild.Args, "-v")

	// Now we choose what we're even building.
	// Default: collect all 'main' packages in cmd/ and build those.
	packages := flag.Args()
	if len(packages) == 0 {
		packages = build.FindMainPackages("./cmd")
	}

	// Do the build!
	for _, pkg := range packages {
		args := make([]string, len(gobuild.Args))
		copy(args, gobuild.Args)
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
		ld = append(ld, "-X", "github.com/ethereum/go-ethereum/internal/version.gitCommit="+env.Commit)
		ld = append(ld, "-X", "github.com/ethereum/go-ethereum/internal/version.gitDate="+env.Date)
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
		// making build-id reproducably absent.
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
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching downloads")
	)
	flag.CommandLine.Parse(cmdline)

	// Get test fixtures.
	csdb := build.MustLoadChecksums("build/checksums.txt")
	downloadSpecTestFixtures(csdb, *cachedir)

	// Configure the toolchain.
	tc := build.GoToolchain{GOARCH: *arch, CC: *cc}
	if *dlgo {
		tc.Root = build.DownloadGo(csdb)
	}
	gotest := tc.Go("test")

	// CI needs a bit more time for the statetests (default 10m).
	gotest.Args = append(gotest.Args, "-timeout=30m")

	// Enable CKZG backend in CI.
	gotest.Args = append(gotest.Args, "-tags=ckzg")

	// Enable integration-tests
	gotest.Args = append(gotest.Args, "-tags=integrationtests")

	// Test a single package at a time. CI builders are slow
	// and some tests run into timeouts under load.
	gotest.Args = append(gotest.Args, "-p", "1")
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover")
	}
	if *verbose {
		gotest.Args = append(gotest.Args, "-v")
	}
	if *race {
		gotest.Args = append(gotest.Args, "-race")
	}
	if *short {
		gotest.Args = append(gotest.Args, "-short")
	}

	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}
	gotest.Args = append(gotest.Args, packages...)
	build.MustRun(gotest)
}

// downloadSpecTestFixtures downloads and extracts the execution-spec-tests fixtures.
func downloadSpecTestFixtures(csdb *build.ChecksumDB, cachedir string) string {
	executionSpecTestsVersion, err := build.Version(csdb, "spec-tests")
	if err != nil {
		log.Fatal(err)
	}
	ext := ".tar.gz"
	base := "fixtures_develop" // TODO(MariusVanDerWijden) rename once the version becomes part of the filename
	url := fmt.Sprintf("https://github.com/ethereum/execution-spec-tests/releases/download/v%s/%s%s", executionSpecTestsVersion, base, ext)
	archivePath := filepath.Join(cachedir, base+ext)
	if err := csdb.DownloadFile(url, archivePath); err != nil {
		log.Fatal(err)
	}
	if err := build.ExtractArchive(archivePath, executionSpecTestsDir); err != nil {
		log.Fatal(err)
	}
	return filepath.Join(cachedir, base)
}

// hashAllSourceFiles iterates all files under the top-level project directory
// computing the hash of each file (excluding files within the tests
// subrepo)
func hashAllSourceFiles() (map[string]common.Hash, error) {
	res := make(map[string]common.Hash)
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if strings.HasPrefix(path, filepath.FromSlash("tests/testdata")) {
			return filepath.SkipDir
		}
		if !d.Type().IsRegular() {
			return nil
		}
		// open the file and hash it
		f, err := os.OpenFile(path, os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}
		res[path] = common.Hash(hasher.Sum(nil))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// hashSourceFiles iterates the provided set of filepaths (relative to the top-level geth project directory)
// computing the hash of each file.
func hashSourceFiles(files []string) (map[string]common.Hash, error) {
	res := make(map[string]common.Hash)
	for _, filePath := range files {
		f, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
		if err != nil {
			return nil, err
		}
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return nil, err
		}
		res[filePath] = common.Hash(hasher.Sum(nil))
	}
	return res, nil
}

// compareHashedFilesets compares two maps (key is relative file path to top-level geth directory, value is its hash)
// and returns the list of file paths whose hashes differed.
func compareHashedFilesets(preHashes map[string]common.Hash, postHashes map[string]common.Hash) []string {
	updates := []string{}
	for path, postHash := range postHashes {
		preHash, ok := preHashes[path]
		if !ok || preHash != postHash {
			updates = append(updates, path)
		}
	}
	return updates
}

func doGoModTidy() {
	targetFiles := []string{"go.mod", "go.sum"}
	preHashes, err := hashSourceFiles(targetFiles)
	if err != nil {
		log.Fatal("failed to hash go.mod/go.sum", "err", err)
	}
	tc := new(build.GoToolchain)
	c := tc.Go("mod", "tidy")
	build.MustRun(c)
	postHashes, err := hashSourceFiles(targetFiles)
	updates := compareHashedFilesets(preHashes, postHashes)
	for _, updatedFile := range updates {
		fmt.Fprintf(os.Stderr, "changed file %s\n", updatedFile)
	}
	if len(updates) != 0 {
		log.Fatal("go.sum and/or go.mod were updated by running 'go mod tidy'")
	}
}

// doGenerate ensures that re-generating generated files does not cause
// any mutations in the source file tree:  i.e. all generated files were
// updated and committed.  Any stale generated files are updated.
func doGenerate() {
	var (
		tc       = new(build.GoToolchain)
		cachedir = flag.String("cachedir", "./build/cache", "directory for caching binaries.")
		verify   = flag.Bool("verify", false, "check whether any files are changed by go generate")
	)

	protocPath := downloadProtoc(*cachedir)
	protocGenGoPath := downloadProtocGenGo(*cachedir)

	var preHashes map[string]common.Hash
	if *verify {
		var err error
		preHashes, err = hashAllSourceFiles()
		if err != nil {
			log.Fatal("failed to compute map of source hashes", "err", err)
		}
	}

	c := tc.Go("generate", "./...")
	pathList := []string{filepath.Join(protocPath, "bin"), protocGenGoPath, os.Getenv("PATH")}
	c.Env = append(c.Env, "PATH="+strings.Join(pathList, string(os.PathListSeparator)))
	build.MustRun(c)

	if !*verify {
		return
	}
	// Check if files were changed.
	postHashes, err := hashAllSourceFiles()
	if err != nil {
		log.Fatal("error computing source tree file hashes", "err", err)
	}
	updates := compareHashedFilesets(preHashes, postHashes)
	for _, updatedFile := range updates {
		fmt.Fprintf(os.Stderr, "changed file %s\n", updatedFile)
	}
	if len(updates) != 0 {
		log.Fatal("One or more generated files were updated by running 'go generate ./...'")
	}
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

	doGoModTidy()
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

// Release Packaging
func doArchive(cmdline []string) {
	var (
		arch    = flag.String("arch", runtime.GOARCH, "Architecture cross packaging")
		atype   = flag.String("type", "zip", "Type of archive to write (zip|tar)")
		signer  = flag.String("signer", "", `Environment variable holding the signing key (e.g. LINUX_SIGNING_KEY)`)
		signify = flag.String("signify", "", `Environment variable holding the signify key (e.g. LINUX_SIGNIFY_KEY)`)
		upload  = flag.String("upload", "", `Destination to upload the archives (usually "gethstore/builds")`)
		ext     string
	)
	flag.CommandLine.Parse(cmdline)
	switch *atype {
	case "zip":
		ext = ".zip"
	case "tar":
		ext = ".tar.gz"
	default:
		log.Fatal("unknown archive type: ", atype)
	}

	var (
		env      = build.Env()
		basegeth = archiveBasename(*arch, params.ArchiveVersion(env.Commit))
		geth     = "geth-" + basegeth + ext
		alltools = "geth-alltools-" + basegeth + ext
	)
	maybeSkipArchive(env)
	if err := build.WriteArchive(geth, gethArchiveFiles); err != nil {
		log.Fatal(err)
	}
	if err := build.WriteArchive(alltools, allToolsArchiveFiles); err != nil {
		log.Fatal(err)
	}
	for _, archive := range []string{geth, alltools} {
		if err := archiveUpload(archive, *upload, *signer, *signify); err != nil {
			log.Fatal(err)
		}
	}
}

func archiveBasename(arch string, archiveVersion string) string {
	platform := runtime.GOOS + "-" + arch
	if arch == "arm" {
		platform += os.Getenv("GOARM")
	}
	if arch == "android" {
		platform = "android-all"
	}
	if arch == "ios" {
		platform = "ios-all"
	}
	return platform + "-" + archiveVersion
}

func archiveUpload(archive string, blobstore string, signer string, signifyVar string) error {
	// If signing was requested, generate the signature files
	if signer != "" {
		key := getenvBase64(signer)
		if err := build.PGPSignFile(archive, archive+".asc", string(key)); err != nil {
			return err
		}
	}
	if signifyVar != "" {
		key := os.Getenv(signifyVar)
		untrustedComment := "verify with geth-release.pub"
		trustedComment := fmt.Sprintf("%s (%s)", archive, time.Now().UTC().Format(time.RFC1123))
		if err := signify.SignFile(archive, archive+".sig", key, untrustedComment, trustedComment); err != nil {
			return err
		}
	}
	// If uploading to Azure was requested, push the archive possibly with its signature
	if blobstore != "" {
		auth := build.AzureBlobstoreConfig{
			Account:   strings.Split(blobstore, "/")[0],
			Token:     os.Getenv("AZURE_BLOBSTORE_TOKEN"),
			Container: strings.SplitN(blobstore, "/", 2)[1],
		}
		if err := build.AzureBlobstoreUpload(archive, filepath.Base(archive), auth); err != nil {
			return err
		}
		if signer != "" {
			if err := build.AzureBlobstoreUpload(archive+".asc", filepath.Base(archive+".asc"), auth); err != nil {
				return err
			}
		}
		if signifyVar != "" {
			if err := build.AzureBlobstoreUpload(archive+".sig", filepath.Base(archive+".sig"), auth); err != nil {
				return err
			}
		}
	}
	return nil
}

// skips archiving for some build configurations.
func maybeSkipArchive(env build.Environment) {
	if env.IsPullRequest {
		log.Printf("skipping archive creation because this is a PR build")
		os.Exit(0)
	}
	if env.Branch != "master" && !strings.HasPrefix(env.Tag, "v1.") {
		log.Printf("skipping archive creation because branch %q, tag %q is not on the inclusion list", env.Branch, env.Tag)
		os.Exit(0)
	}
}

// Builds the docker images and optionally uploads them to Docker Hub.
func doDocker(cmdline []string) {
	var (
		image    = flag.Bool("image", false, `Whether to build and push an arch specific docker image`)
		manifest = flag.String("manifest", "", `Push a multi-arch docker image for the specified architectures (usually "amd64,arm64")`)
		upload   = flag.String("upload", "", `Where to upload the docker image (usually "ethereum/client-go")`)
	)
	flag.CommandLine.Parse(cmdline)

	// Skip building and pushing docker images for PR builds
	env := build.Env()
	maybeSkipArchive(env)

	// Retrieve the upload credentials and authenticate
	user := getenvBase64("DOCKER_HUB_USERNAME")
	pass := getenvBase64("DOCKER_HUB_PASSWORD")

	if len(user) > 0 && len(pass) > 0 {
		auther := exec.Command("docker", "login", "-u", string(user), "--password-stdin")
		auther.Stdin = bytes.NewReader(pass)
		build.MustRun(auther)
	}
	// Retrieve the version infos to build and push to the following paths:
	//  - ethereum/client-go:latest                            - Pushes to the master branch, Geth only
	//  - ethereum/client-go:stable                            - Version tag publish on GitHub, Geth only
	//  - ethereum/client-go:alltools-latest                   - Pushes to the master branch, Geth & tools
	//  - ethereum/client-go:alltools-stable                   - Version tag publish on GitHub, Geth & tools
	//  - ethereum/client-go:release-<major>.<minor>           - Version tag publish on GitHub, Geth only
	//  - ethereum/client-go:alltools-release-<major>.<minor>  - Version tag publish on GitHub, Geth & tools
	//  - ethereum/client-go:v<major>.<minor>.<patch>          - Version tag publish on GitHub, Geth only
	//  - ethereum/client-go:alltools-v<major>.<minor>.<patch> - Version tag publish on GitHub, Geth & tools
	var tags []string

	switch {
	case env.Branch == "master":
		tags = []string{"latest"}
	case strings.HasPrefix(env.Tag, "v1."):
		tags = []string{"stable", fmt.Sprintf("release-1.%d", params.VersionMinor), "v" + params.Version}
	}
	// If architecture specific image builds are requested, build and push them
	if *image {
		build.MustRunCommand("docker", "build", "--build-arg", "COMMIT="+env.Commit, "--build-arg", "VERSION="+params.VersionWithMeta, "--build-arg", "BUILDNUM="+env.Buildnum, "--tag", fmt.Sprintf("%s:TAG", *upload), ".")
		build.MustRunCommand("docker", "build", "--build-arg", "COMMIT="+env.Commit, "--build-arg", "VERSION="+params.VersionWithMeta, "--build-arg", "BUILDNUM="+env.Buildnum, "--tag", fmt.Sprintf("%s:alltools-TAG", *upload), "-f", "Dockerfile.alltools", ".")

		// Tag and upload the images to Docker Hub
		for _, tag := range tags {
			gethImage := fmt.Sprintf("%s:%s-%s", *upload, tag, runtime.GOARCH)
			toolImage := fmt.Sprintf("%s:alltools-%s-%s", *upload, tag, runtime.GOARCH)

			// If the image already exists (non version tag), check the build
			// number to prevent overwriting a newer commit if concurrent builds
			// are running. This is still a tiny bit racey if two published are
			// done at the same time, but that's extremely unlikely even on the
			// master branch.
			for _, img := range []string{gethImage, toolImage} {
				if exec.Command("docker", "pull", img).Run() != nil {
					continue // Generally the only failure is a missing image, which is good
				}
				buildnum, err := exec.Command("docker", "inspect", "--format", "{{index .Config.Labels \"buildnum\"}}", img).CombinedOutput()
				if err != nil {
					log.Fatalf("Failed to inspect container: %v\nOutput: %s", err, string(buildnum))
				}
				buildnum = bytes.TrimSpace(buildnum)

				if len(buildnum) > 0 && len(env.Buildnum) > 0 {
					oldnum, err := strconv.Atoi(string(buildnum))
					if err != nil {
						log.Fatalf("Failed to parse old image build number: %v", err)
					}
					newnum, err := strconv.Atoi(env.Buildnum)
					if err != nil {
						log.Fatalf("Failed to parse current build number: %v", err)
					}
					if oldnum > newnum {
						log.Fatalf("Current build number %d not newer than existing %d", newnum, oldnum)
					} else {
						log.Printf("Updating %s from build %d to %d", img, oldnum, newnum)
					}
				}
			}
			build.MustRunCommand("docker", "image", "tag", fmt.Sprintf("%s:TAG", *upload), gethImage)
			build.MustRunCommand("docker", "image", "tag", fmt.Sprintf("%s:alltools-TAG", *upload), toolImage)
			build.MustRunCommand("docker", "push", gethImage)
			build.MustRunCommand("docker", "push", toolImage)
		}
	}
	// If multi-arch image manifest push is requested, assemble it
	if len(*manifest) != 0 {
		// Since different architectures are pushed by different builders, wait
		// until all required images are updated.
		var mismatch bool
		for i := 0; i < 2; i++ { // 2 attempts, second is race check
			mismatch = false // hope there's no mismatch now

			for _, tag := range tags {
				for _, arch := range strings.Split(*manifest, ",") {
					gethImage := fmt.Sprintf("%s:%s-%s", *upload, tag, arch)
					toolImage := fmt.Sprintf("%s:alltools-%s-%s", *upload, tag, arch)

					for _, img := range []string{gethImage, toolImage} {
						if out, err := exec.Command("docker", "pull", img).CombinedOutput(); err != nil {
							log.Printf("Required image %s unavailable: %v\nOutput: %s", img, err, out)
							mismatch = true
							break
						}
						buildnum, err := exec.Command("docker", "inspect", "--format", "{{index .Config.Labels \"buildnum\"}}", img).CombinedOutput()
						if err != nil {
							log.Fatalf("Failed to inspect container: %v\nOutput: %s", err, string(buildnum))
						}
						buildnum = bytes.TrimSpace(buildnum)

						if string(buildnum) != env.Buildnum {
							log.Printf("Build number mismatch on %s: want %s, have %s", img, env.Buildnum, buildnum)
							mismatch = true
							break
						}
					}
					if mismatch {
						break
					}
				}
				if mismatch {
					break
				}
			}
			if mismatch {
				// Build numbers mismatching, retry in a short time to
				// avoid concurrent fails in both publisher images. If
				// however the retry failed too, it means the concurrent
				// builder is still crunching, let that do the publish.
				if i == 0 {
					time.Sleep(30 * time.Second)
				}
				continue
			}
			break
		}
		if mismatch {
			log.Println("Relinquishing publish to other builder")
			return
		}
		// Assemble and push the Geth manifest image
		for _, tag := range tags {
			gethImage := fmt.Sprintf("%s:%s", *upload, tag)

			var gethSubImages []string
			for _, arch := range strings.Split(*manifest, ",") {
				gethSubImages = append(gethSubImages, gethImage+"-"+arch)
			}
			build.MustRunCommand("docker", append([]string{"manifest", "create", gethImage}, gethSubImages...)...)
			build.MustRunCommand("docker", "manifest", "push", gethImage)
		}
		// Assemble and push the alltools manifest image
		for _, tag := range tags {
			toolImage := fmt.Sprintf("%s:alltools-%s", *upload, tag)

			var toolSubImages []string
			for _, arch := range strings.Split(*manifest, ",") {
				toolSubImages = append(toolSubImages, toolImage+"-"+arch)
			}
			build.MustRunCommand("docker", append([]string{"manifest", "create", toolImage}, toolSubImages...)...)
			build.MustRunCommand("docker", "manifest", "push", toolImage)
		}
	}
}

// Debian Packaging
func doDebianSource(cmdline []string) {
	var (
		cachedir = flag.String("cachedir", "./build/cache", `Filesystem path to cache the downloaded Go bundles at`)
		signer   = flag.String("signer", "", `Signing key name, also used as package author`)
		upload   = flag.String("upload", "", `Where to upload the source package (usually "ethereum/ethereum")`)
		sshUser  = flag.String("sftp-user", "", `Username for SFTP upload (usually "geth-ci")`)
		workdir  = flag.String("workdir", "", `Output directory for packages (uses temp dir if unset)`)
		now      = time.Now()
	)
	flag.CommandLine.Parse(cmdline)
	*workdir = makeWorkdir(*workdir)
	env := build.Env()
	tc := new(build.GoToolchain)
	maybeSkipArchive(env)

	// Import the signing key.
	if key := getenvBase64("PPA_SIGNING_KEY"); len(key) > 0 {
		gpg := exec.Command("gpg", "--import")
		gpg.Stdin = bytes.NewReader(key)
		build.MustRun(gpg)
	}
	// Download and verify the Go source packages.
	var (
		gobootbundles = downloadGoBootstrapSources(*cachedir)
		gobundle      = downloadGoSources(*cachedir)
	)
	// Download all the dependencies needed to build the sources and run the ci script
	srcdepfetch := tc.Go("mod", "download")
	srcdepfetch.Env = append(srcdepfetch.Env, "GOPATH="+filepath.Join(*workdir, "modgopath"))
	build.MustRun(srcdepfetch)

	cidepfetch := tc.Go("run", "./build/ci.go")
	cidepfetch.Env = append(cidepfetch.Env, "GOPATH="+filepath.Join(*workdir, "modgopath"))
	cidepfetch.Run() // Command fails, don't care, we only need the deps to start it

	// Create Debian packages and upload them.
	for _, pkg := range debPackages {
		for _, distro := range debDistros {
			// Prepare the debian package with the go-ethereum sources.
			meta := newDebMetadata(distro, *signer, env, now, pkg.Name, pkg.Version, pkg.Executables)
			pkgdir := stageDebianSource(*workdir, meta)

			// Add bootstrapper Go source code
			for i, gobootbundle := range gobootbundles {
				if err := build.ExtractArchive(gobootbundle, pkgdir); err != nil {
					log.Fatalf("Failed to extract bootstrapper Go sources: %v", err)
				}
				if err := os.Rename(filepath.Join(pkgdir, "go"), filepath.Join(pkgdir, fmt.Sprintf(".goboot-%d", i+1))); err != nil {
					log.Fatalf("Failed to rename bootstrapper Go source folder: %v", err)
				}
			}
			// Add builder Go source code
			if err := build.ExtractArchive(gobundle, pkgdir); err != nil {
				log.Fatalf("Failed to extract builder Go sources: %v", err)
			}
			if err := os.Rename(filepath.Join(pkgdir, "go"), filepath.Join(pkgdir, ".go")); err != nil {
				log.Fatalf("Failed to rename builder Go source folder: %v", err)
			}
			// Add all dependency modules in compressed form
			os.MkdirAll(filepath.Join(pkgdir, ".mod", "cache"), 0755)
			if err := cp.CopyAll(filepath.Join(pkgdir, ".mod", "cache", "download"), filepath.Join(*workdir, "modgopath", "pkg", "mod", "cache", "download")); err != nil {
				log.Fatalf("Failed to copy Go module dependencies: %v", err)
			}
			// Run the packaging and upload to the PPA
			debuild := exec.Command("debuild", "-S", "-sa", "-us", "-uc", "-d", "-Zxz", "-nc")
			debuild.Dir = pkgdir
			build.MustRun(debuild)

			var (
				basename  = fmt.Sprintf("%s_%s", meta.Name(), meta.VersionString())
				source    = filepath.Join(*workdir, basename+".tar.xz")
				dsc       = filepath.Join(*workdir, basename+".dsc")
				changes   = filepath.Join(*workdir, basename+"_source.changes")
				buildinfo = filepath.Join(*workdir, basename+"_source.buildinfo")
			)
			if *signer != "" {
				build.MustRunCommand("debsign", changes)
			}
			if *upload != "" {
				ppaUpload(*workdir, *upload, *sshUser, []string{source, dsc, changes, buildinfo})
			}
		}
	}
}

// downloadGoBootstrapSources downloads the Go source tarball(s) that will be used
// to bootstrap the builder Go.
func downloadGoBootstrapSources(cachedir string) []string {
	csdb := build.MustLoadChecksums("build/checksums.txt")

	var bundles []string
	for _, booter := range []string{"ppa-builder-1", "ppa-builder-2"} {
		gobootVersion, err := build.Version(csdb, booter)
		if err != nil {
			log.Fatal(err)
		}
		file := fmt.Sprintf("go%s.src.tar.gz", gobootVersion)
		url := "https://dl.google.com/go/" + file
		dst := filepath.Join(cachedir, file)
		if err := csdb.DownloadFile(url, dst); err != nil {
			log.Fatal(err)
		}
		bundles = append(bundles, dst)
	}
	return bundles
}

// downloadGoSources downloads the Go source tarball.
func downloadGoSources(cachedir string) string {
	csdb := build.MustLoadChecksums("build/checksums.txt")
	dlgoVersion, err := build.Version(csdb, "golang")
	if err != nil {
		log.Fatal(err)
	}
	file := fmt.Sprintf("go%s.src.tar.gz", dlgoVersion)
	url := "https://dl.google.com/go/" + file
	dst := filepath.Join(cachedir, file)
	if err := csdb.DownloadFile(url, dst); err != nil {
		log.Fatal(err)
	}
	return dst
}

func ppaUpload(workdir, ppa, sshUser string, files []string) {
	p := strings.Split(ppa, "/")
	if len(p) != 2 {
		log.Fatal("-upload PPA name must contain single /")
	}
	if sshUser == "" {
		sshUser = p[0]
	}
	incomingDir := fmt.Sprintf("~%s/ubuntu/%s", p[0], p[1])
	// Create the SSH identity file if it doesn't exist.
	var idfile string
	if sshkey := getenvBase64("PPA_SSH_KEY"); len(sshkey) > 0 {
		idfile = filepath.Join(workdir, "sshkey")
		if !common.FileExist(idfile) {
			os.WriteFile(idfile, sshkey, 0600)
		}
	}
	// Upload
	dest := sshUser + "@ppa.launchpad.net"
	if err := build.UploadSFTP(idfile, dest, incomingDir, files); err != nil {
		log.Fatal(err)
	}
}

func getenvBase64(variable string) []byte {
	dec, err := base64.StdEncoding.DecodeString(os.Getenv(variable))
	if err != nil {
		log.Fatal("invalid base64 " + variable)
	}
	return []byte(dec)
}

func makeWorkdir(wdflag string) string {
	var err error
	if wdflag != "" {
		err = os.MkdirAll(wdflag, 0744)
	} else {
		wdflag, err = os.MkdirTemp("", "geth-build-")
	}
	if err != nil {
		log.Fatal(err)
	}
	return wdflag
}

func isUnstableBuild(env build.Environment) bool {
	if env.Tag != "" {
		return false
	}
	return true
}

type debPackage struct {
	Name        string          // the name of the Debian package to produce, e.g. "ethereum"
	Version     string          // the clean version of the debPackage, e.g. 1.8.12, without any metadata
	Executables []debExecutable // executables to be included in the package
}

type debMetadata struct {
	Env         build.Environment
	PackageName string

	// go-ethereum version being built. Note that this
	// is not the debian package version. The package version
	// is constructed by VersionString.
	Version string

	Author       string // "name <email>", also selects signing key
	Distro, Time string
	Executables  []debExecutable
}

type debExecutable struct {
	PackageName string
	BinaryName  string
	Description string
}

// Package returns the name of the package if present, or
// fallbacks to BinaryName
func (d debExecutable) Package() string {
	if d.PackageName != "" {
		return d.PackageName
	}
	return d.BinaryName
}

func newDebMetadata(distro, author string, env build.Environment, t time.Time, name string, version string, exes []debExecutable) debMetadata {
	if author == "" {
		// No signing key, use default author.
		author = "Ethereum Builds <fjl@ethereum.org>"
	}
	return debMetadata{
		PackageName: name,
		Env:         env,
		Author:      author,
		Distro:      distro,
		Version:     version,
		Time:        t.Format(time.RFC1123Z),
		Executables: exes,
	}
}

// Name returns the name of the metapackage that depends
// on all executable packages.
func (meta debMetadata) Name() string {
	if isUnstableBuild(meta.Env) {
		return meta.PackageName + "-unstable"
	}
	return meta.PackageName
}

// VersionString returns the debian version of the packages.
func (meta debMetadata) VersionString() string {
	vsn := meta.Version
	if meta.Env.Buildnum != "" {
		vsn += "+build" + meta.Env.Buildnum
	}
	if meta.Distro != "" {
		vsn += "+" + meta.Distro
	}
	return vsn
}

// ExeList returns the list of all executable packages.
func (meta debMetadata) ExeList() string {
	names := make([]string, len(meta.Executables))
	for i, e := range meta.Executables {
		names[i] = meta.ExeName(e)
	}
	return strings.Join(names, ", ")
}

// ExeName returns the package name of an executable package.
func (meta debMetadata) ExeName(exe debExecutable) string {
	if isUnstableBuild(meta.Env) {
		return exe.Package() + "-unstable"
	}
	return exe.Package()
}

// ExeConflicts returns the content of the Conflicts field
// for executable packages.
func (meta debMetadata) ExeConflicts(exe debExecutable) string {
	if isUnstableBuild(meta.Env) {
		// Set up the conflicts list so that the *-unstable packages
		// cannot be installed alongside the regular version.
		//
		// https://www.debian.org/doc/debian-policy/ch-relationships.html
		// is very explicit about Conflicts: and says that Breaks: should
		// be preferred and the conflicting files should be handled via
		// alternates. We might do this eventually but using a conflict is
		// easier now.
		return "ethereum, " + exe.Package()
	}
	return ""
}

func stageDebianSource(tmpdir string, meta debMetadata) (pkgdir string) {
	pkg := meta.Name() + "-" + meta.VersionString()
	pkgdir = filepath.Join(tmpdir, pkg)
	if err := os.Mkdir(pkgdir, 0755); err != nil {
		log.Fatal(err)
	}
	// Copy the source code.
	build.MustRunCommand("git", "checkout-index", "-a", "--prefix", pkgdir+string(filepath.Separator))

	// Put the debian build files in place.
	debian := filepath.Join(pkgdir, "debian")
	build.Render("build/deb/"+meta.PackageName+"/deb.rules", filepath.Join(debian, "rules"), 0755, meta)
	build.Render("build/deb/"+meta.PackageName+"/deb.changelog", filepath.Join(debian, "changelog"), 0644, meta)
	build.Render("build/deb/"+meta.PackageName+"/deb.control", filepath.Join(debian, "control"), 0644, meta)
	build.Render("build/deb/"+meta.PackageName+"/deb.copyright", filepath.Join(debian, "copyright"), 0644, meta)
	build.RenderString("8\n", filepath.Join(debian, "compat"), 0644, meta)
	build.RenderString("3.0 (native)\n", filepath.Join(debian, "source/format"), 0644, meta)
	for _, exe := range meta.Executables {
		install := filepath.Join(debian, meta.ExeName(exe)+".install")
		docs := filepath.Join(debian, meta.ExeName(exe)+".docs")
		build.Render("build/deb/"+meta.PackageName+"/deb.install", install, 0644, exe)
		build.Render("build/deb/"+meta.PackageName+"/deb.docs", docs, 0644, exe)
	}
	return pkgdir
}

// Windows installer
func doWindowsInstaller(cmdline []string) {
	// Parse the flags and make skip installer generation on PRs
	var (
		arch    = flag.String("arch", runtime.GOARCH, "Architecture for cross build packaging")
		signer  = flag.String("signer", "", `Environment variable holding the signing key (e.g. WINDOWS_SIGNING_KEY)`)
		signify = flag.String("signify key", "", `Environment variable holding the signify signing key (e.g. WINDOWS_SIGNIFY_KEY)`)
		upload  = flag.String("upload", "", `Destination to upload the archives (usually "gethstore/builds")`)
		workdir = flag.String("workdir", "", `Output directory for packages (uses temp dir if unset)`)
	)
	flag.CommandLine.Parse(cmdline)
	*workdir = makeWorkdir(*workdir)
	env := build.Env()
	maybeSkipArchive(env)

	// Aggregate binaries that are included in the installer
	var (
		devTools []string
		allTools []string
		gethTool string
	)
	for _, file := range allToolsArchiveFiles {
		if file == "COPYING" { // license, copied later
			continue
		}
		allTools = append(allTools, filepath.Base(file))
		if filepath.Base(file) == "geth.exe" {
			gethTool = file
		} else {
			devTools = append(devTools, file)
		}
	}

	// Render NSIS scripts: Installer NSIS contains two installer sections,
	// first section contains the geth binary, second section holds the dev tools.
	templateData := map[string]interface{}{
		"License":  "COPYING",
		"Geth":     gethTool,
		"DevTools": devTools,
	}
	build.Render("build/nsis.geth.nsi", filepath.Join(*workdir, "geth.nsi"), 0644, nil)
	build.Render("build/nsis.install.nsh", filepath.Join(*workdir, "install.nsh"), 0644, templateData)
	build.Render("build/nsis.uninstall.nsh", filepath.Join(*workdir, "uninstall.nsh"), 0644, allTools)
	build.Render("build/nsis.pathupdate.nsh", filepath.Join(*workdir, "PathUpdate.nsh"), 0644, nil)
	build.Render("build/nsis.envvarupdate.nsh", filepath.Join(*workdir, "EnvVarUpdate.nsh"), 0644, nil)
	if err := cp.CopyFile(filepath.Join(*workdir, "SimpleFC.dll"), "build/nsis.simplefc.dll"); err != nil {
		log.Fatalf("Failed to copy SimpleFC.dll: %v", err)
	}
	if err := cp.CopyFile(filepath.Join(*workdir, "COPYING"), "COPYING"); err != nil {
		log.Fatalf("Failed to copy copyright note: %v", err)
	}
	// Build the installer. This assumes that all the needed files have been previously
	// built (don't mix building and packaging to keep cross compilation complexity to a
	// minimum).
	version := strings.Split(params.Version, ".")
	if env.Commit != "" {
		version[2] += "-" + env.Commit[:8]
	}
	installer, err := filepath.Abs("geth-" + archiveBasename(*arch, params.ArchiveVersion(env.Commit)) + ".exe")
	if err != nil {
		log.Fatalf("Failed to convert installer file path: %v", err)
	}
	build.MustRunCommand("makensis.exe",
		"/DOUTPUTFILE="+installer,
		"/DMAJORVERSION="+version[0],
		"/DMINORVERSION="+version[1],
		"/DBUILDVERSION="+version[2],
		"/DARCH="+*arch,
		filepath.Join(*workdir, "geth.nsi"),
	)
	// Sign and publish installer.
	if err := archiveUpload(installer, *upload, *signer, *signify); err != nil {
		log.Fatal(err)
	}
}

// Binary distribution cleanups

func doPurge(cmdline []string) {
	var (
		store = flag.String("store", "", `Destination from where to purge archives (usually "gethstore/builds")`)
		limit = flag.Int("days", 30, `Age threshold above which to delete unstable archives`)
	)
	flag.CommandLine.Parse(cmdline)

	if env := build.Env(); !env.IsCronJob {
		log.Printf("skipping because not a cron job")
		os.Exit(0)
	}
	// Create the azure authentication and list the current archives
	auth := build.AzureBlobstoreConfig{
		Account:   strings.Split(*store, "/")[0],
		Token:     os.Getenv("AZURE_BLOBSTORE_TOKEN"),
		Container: strings.SplitN(*store, "/", 2)[1],
	}
	blobs, err := build.AzureBlobstoreList(auth)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d blobs\n", len(blobs))

	// Iterate over the blobs, collect and sort all unstable builds
	for i := 0; i < len(blobs); i++ {
		if !strings.Contains(*blobs[i].Name, "unstable") {
			blobs = append(blobs[:i], blobs[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(blobs); i++ {
		for j := i + 1; j < len(blobs); j++ {
			if blobs[i].Properties.LastModified.After(*blobs[j].Properties.LastModified) {
				blobs[i], blobs[j] = blobs[j], blobs[i]
			}
		}
	}
	// Filter out all archives more recent that the given threshold
	for i, blob := range blobs {
		if time.Since(*blob.Properties.LastModified) < time.Duration(*limit)*24*time.Hour {
			blobs = blobs[:i]
			break
		}
	}
	fmt.Printf("Deleting %d blobs\n", len(blobs))
	// Delete all marked as such and return
	if err := build.AzureBlobstoreDelete(auth, blobs); err != nil {
		log.Fatal(err)
	}
}

func doSanityCheck() {
	build.DownloadAndVerifyChecksums(build.MustLoadChecksums("build/checksums.txt"))
}
