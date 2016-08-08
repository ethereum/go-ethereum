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

// +build none

/*
The ci command is called from Continuous Integration scripts.

Usage: go run ci.go <command> <command flags/arguments>

Available commands are:

   install    [ packages... ]                          -- builds packages and executables
   test       [ -coverage ] [ -vet ] [ packages... ]   -- runs the tests
   archive    [ -type zip|tar ]                        -- archives build artefacts
   importkeys                                          -- imports signing keys from env
   debsrc     [ -sign key-id ] [ -upload dest ]        -- creates a debian source package
   xgo        [ options ]                              -- cross builds according to options

For all commands, -n prevents execution of external programs (dry run mode).

*/
package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"../internal/build"
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
		executablePath("evm"),
		executablePath("geth"),
		executablePath("rlpdump"),
	}

	// A debian package is created for all executables listed here.
	debExecutables = []debExecutable{
		{
			Name:        "geth",
			Description: "Ethereum CLI client.",
		},
		{
			Name:        "rlpdump",
			Description: "Developer utility tool that prints RLP structures.",
		},
		{
			Name:        "evm",
			Description: "Developer utility version of the EVM (Ethereum Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode.",
		},
		{
			Name:        "abigen",
			Description: "Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages.",
		},
	}

	// Distros for which packages are created.
	// Note: vivid is unsupported because there is no golang-1.6 package for it.
	debDistros = []string{"trusty", "wily", "xenial", "yakkety"}
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
	case "archive":
		doArchive(os.Args[2:])
	case "debsrc":
		doDebianSource(os.Args[2:])
	case "travis-debsrc":
		doTravisDebianSource(os.Args[2:])
	case "xgo":
		doXgo(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

// Compiling

func doInstall(cmdline []string) {
	commitHash := flag.String("gitcommit", "", "Git commit hash embedded into binary.")
	flag.CommandLine.Parse(cmdline)

	// Check Go version. People regularly open issues about compilation
	// failure with outdated Go. This should save them the trouble.
	if runtime.Version() < "go1.4" && !strings.HasPrefix(runtime.Version(), "devel") {
		log.Println("You have Go version", runtime.Version())
		log.Println("go-ethereum requires at least Go version 1.4 and cannot")
		log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
		os.Exit(1)
	}

	// Compile packages given as arguments, or everything if there are no arguments.
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}

	goinstall := goTool("install", makeBuildFlags(*commitHash)...)
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, packages...)
	build.MustRun(goinstall)
}

func makeBuildFlags(commitHash string) (flags []string) {
	// Since Go 1.5, the separator char for link time assignments
	// is '=' and using ' ' prints a warning. However, Go < 1.5 does
	// not support using '='.
	sep := " "
	if runtime.Version() > "go1.5" || strings.Contains(runtime.Version(), "devel") {
		sep = "="
	}

	if os.Getenv("GO_OPENCL") != "" {
		flags = append(flags, "-tags", "opencl")
	}

	// Set gitCommit constant via link-time assignment. If this is a git checkout, we can
	// just get the current commit hash through git. Otherwise we fall back to the hash
	// that was passed as -gitcommit.
	//
	// -gitcommit is required for Debian package builds. The source package doesn't
	// contain .git but we still want to embed the commit hash into the packaged binary.
	// The hash is rendered into the debian/rules build script when the source package is
	// created.
	if _, err := os.Stat(filepath.Join(".git", "HEAD")); !os.IsNotExist(err) {
		if c := build.GitCommit(); c != "" {
			commitHash = c
		}
	}
	if commitHash != "" {
		flags = append(flags, "-ldflags", "-X main.gitCommit"+sep+commitHash)
	}
	return flags
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	gocmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(gocmd, subcmd)
	cmd.Args = append(cmd.Args, args...)
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

// Running The Tests
//
// "tests" also includes static analysis tools such as vet.

func doTest(cmdline []string) {
	var (
		vet      = flag.Bool("vet", false, "Whether to run go vet")
		coverage = flag.Bool("coverage", false, "Whether to record code coverage")
	)
	flag.CommandLine.Parse(cmdline)
	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}

	// Run analysis tools before the tests.
	if *vet {
		build.MustRun(goTool("vet", packages...))
	}

	// Run the actual tests.
	gotest := goTool("test")
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover")
	}
	gotest.Args = append(gotest.Args, packages...)
	build.MustRun(gotest)
}

// Release Packaging

func doArchive(cmdline []string) {
	var (
		atype = flag.String("type", "zip", "Type of archive to write (zip|tar)")
		ext   string
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
	base := makeArchiveBasename()
	if err := build.WriteArchive("geth-"+base, ext, gethArchiveFiles); err != nil {
		log.Fatal(err)
	}
	if err := build.WriteArchive("geth-alltools-"+base, ext, allToolsArchiveFiles); err != nil {
		log.Fatal(err)
	}
}

func makeArchiveBasename() string {
	// date := time.Now().UTC().Format("200601021504")
	platform := runtime.GOOS + "-" + runtime.GOARCH
	archive := platform + "-" + build.VERSION()
	if commit := build.GitCommit(); commit != "" {
		archive += "-" + commit[:8]
	}
	return archive
}

// Debian Packaging

// CLI entry point for Travis CI.
func doTravisDebianSource(cmdline []string) {
	flag.CommandLine.Parse(cmdline)

	// Package only whitelisted branches.
	switch {
	case os.Getenv("TRAVIS_REPO_SLUG") != "ethereum/go-ethereum":
		log.Printf("skipping because this is a fork build")
		return
	case os.Getenv("TRAVIS_PULL_REQUEST") != "false":
		log.Printf("skipping because this is a PR build")
		return
	case os.Getenv("TRAVIS_BRANCH") != "develop" && !strings.HasPrefix(os.Getenv("TRAVIS_TAG"), "v1."):
		log.Printf("skipping because branch %q tag %q is not on the whitelist",
			os.Getenv("TRAVIS_BRANCH"),
			os.Getenv("TRAVIS_TAG"))
		return
	}

	// Import the signing key.
	if b64key := os.Getenv("PPA_SIGNING_KEY"); b64key != "" {
		key, err := base64.StdEncoding.DecodeString(b64key)
		if err != nil {
			log.Fatal("invalid base64 PPA_SIGNING_KEY")
		}
		gpg := exec.Command("gpg", "--import")
		gpg.Stdin = bytes.NewReader(key)
		build.MustRun(gpg)
	}

	// Assign unstable status to non-tag builds.
	unstable := "true"
	if os.Getenv("TRAVIS_BRANCH") != "develop" && os.Getenv("TRAVIS_TAG") != "" {
		unstable = "false"
	}

	doDebianSource([]string{
		"-signer", "Felix Lange (Geth CI Testing Key) <fjl@twurst.com>",
		"-buildnum", os.Getenv("TRAVIS_BUILD_NUMBER"),
		"-upload", "ppa:lp-fjl/geth-ci-testing",
		"-unstable", unstable,
	})
}

// CLI entry point for doing packaging locally.
func doDebianSource(cmdline []string) {
	var (
		signer   = flag.String("signer", "", `Signing key name, also used as package author`)
		upload   = flag.String("upload", "", `Where to upload the source package (usually "ppa:ethereum/ethereum")`)
		buildnum = flag.String("buildnum", "", `Build number (included in version)`)
		unstable = flag.Bool("unstable", false, `Use package name suffix "-unstable"`)
		now      = time.Now()
	)
	flag.CommandLine.Parse(cmdline)

	// Create the debian worktree in /tmp.
	tmpdir, err := ioutil.TempDir("", "eth-deb-build-")
	if err != nil {
		log.Fatal(err)
	}

	for _, distro := range debDistros {
		meta := newDebMetadata(distro, *signer, *buildnum, *unstable, now)
		pkgdir := stageDebianSource(tmpdir, meta)
		debuild := exec.Command("debuild", "-S", "-sa", "-us", "-uc")
		debuild.Dir = pkgdir
		build.MustRun(debuild)

		changes := fmt.Sprintf("%s_%s_source.changes", meta.Name(), meta.VersionString())
		changes = filepath.Join(tmpdir, changes)
		if *signer != "" {
			build.MustRunCommand("debsign", changes)
		}
		if *upload != "" {
			build.MustRunCommand("dput", *upload, changes)
		}
	}
}

type debExecutable struct {
	Name, Description string
}

type debMetadata struct {
	// go-ethereum version being built. Note that this
	// is not the debian package version. The package version
	// is constructed by VersionString.
	Version string

	Author               string // "name <email>", also selects signing key
	Buildnum             string // build number
	Distro, Commit, Time string
	Executables          []debExecutable
	Unstable             bool
}

func newDebMetadata(distro, author, buildnum string, unstable bool, t time.Time) debMetadata {
	if author == "" {
		// No signing key, use default author.
		author = "Ethereum Builds <fjl@ethereum.org>"
	}
	return debMetadata{
		Unstable:    unstable,
		Author:      author,
		Distro:      distro,
		Commit:      build.GitCommit(),
		Version:     build.VERSION(),
		Buildnum:    buildnum,
		Time:        t.Format(time.RFC1123Z),
		Executables: debExecutables,
	}
}

// Name returns the name of the metapackage that depends
// on all executable packages.
func (meta debMetadata) Name() string {
	if meta.Unstable {
		return "ethereum-unstable"
	}
	return "ethereum"
}

// VersionString returns the debian version of the packages.
func (meta debMetadata) VersionString() string {
	vsn := meta.Version
	if meta.Buildnum != "" {
		vsn += "+build" + meta.Buildnum
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
	if meta.Unstable {
		return exe.Name + "-unstable"
	}
	return exe.Name
}

// ExeConflicts returns the content of the Conflicts field
// for executable packages.
func (meta debMetadata) ExeConflicts(exe debExecutable) string {
	if meta.Unstable {
		// Set up the conflicts list so that the *-unstable packages
		// cannot be installed alongside the regular version.
		//
		// https://www.debian.org/doc/debian-policy/ch-relationships.html
		// is very explicit about Conflicts: and says that Breaks: should
		// be preferred and the conflicting files should be handled via
		// alternates. We might do this eventually but using a conflict is
		// easier now.
		return "ethereum, " + exe.Name
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
	build.Render("build/deb.rules", filepath.Join(debian, "rules"), 0755, meta)
	build.Render("build/deb.changelog", filepath.Join(debian, "changelog"), 0644, meta)
	build.Render("build/deb.control", filepath.Join(debian, "control"), 0644, meta)
	build.Render("build/deb.copyright", filepath.Join(debian, "copyright"), 0644, meta)
	build.RenderString("8\n", filepath.Join(debian, "compat"), 0644, meta)
	build.RenderString("3.0 (native)\n", filepath.Join(debian, "source/format"), 0644, meta)
	for _, exe := range meta.Executables {
		install := filepath.Join(debian, exe.Name+".install")
		docs := filepath.Join(debian, exe.Name+".docs")
		build.Render("build/deb.install", install, 0644, exe)
		build.Render("build/deb.docs", docs, 0644, exe)
	}

	return pkgdir
}

// Cross compilation

func doXgo(cmdline []string) {
	// Make sure xgo is available for cross compilation
	gogetxgo := goTool("get", "github.com/karalabe/xgo")
	build.MustRun(gogetxgo)

	// Execute the actual cross compilation
	pkg := cmdline[len(cmdline)-1]
	args := append(cmdline[:len(cmdline)-1], makeBuildFlags("")...)

	build.MustRun(xgoTool(append(args, pkg)...))
}

func xgoTool(args ...string) *exec.Cmd {
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
