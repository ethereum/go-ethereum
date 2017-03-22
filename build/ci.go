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

   install    [ -arch architecture ] [ packages... ]                                           -- builds packages and executables
   test       [ -coverage ] [ -vet ] [ -misspell ] [ packages... ]                             -- runs the tests
   archive    [ -arch architecture ] [ -type zip|tar ] [ -signer key-envvar ] [ -upload dest ] -- archives build artefacts
   importkeys                                                                                  -- imports signing keys from env
   debsrc     [ -signer key-id ] [ -upload dest ]                                              -- creates a debian source package
   nsis                                                                                        -- creates a Windows NSIS installer
   aar        [ -local ] [ -sign key-id ] [-deploy repo] [ -upload dest ]                      -- creates an Android archive
   xcode      [ -local ] [ -sign key-id ] [-deploy repo] [ -upload dest ]                      -- creates an iOS XCode framework
   xgo        [ -alltools ] [ options ]                                                        -- cross builds according to options

For all commands, -n prevents execution of external programs (dry run mode).

*/
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/internal/build"
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
		executablePath("swarm"),
		executablePath("rlpdump"),
	}

	// A debian package is created for all executables listed here.
	debExecutables = []debExecutable{
		{
			Name:        "geth",
			Description: "Ethereum CLI client.",
		},
		{
			Name:        "bootnode",
			Description: "Ethereum bootnode.",
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
			Name:        "swarm",
			Description: "Ethereum Swarm daemon and tools",
		},
		{
			Name:        "abigen",
			Description: "Source code generator to convert Ethereum contract definitions into easy to use, compile-time type-safe Go packages.",
		},
	}

	// Distros for which packages are created.
	// Note: vivid is unsupported because there is no golang-1.6 package for it.
	// Note: wily is unsupported because it was officially deprecated on lanchpad.
	debDistros = []string{"trusty", "xenial", "yakkety"}
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
	case "nsis":
		doWindowsInstaller(os.Args[2:])
	case "aar":
		doAndroidArchive(os.Args[2:])
	case "xcode":
		doXCodeFramework(os.Args[2:])
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
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Check Go version. People regularly open issues about compilation
	// failure with outdated Go. This should save them the trouble.
	if runtime.Version() < "go1.7" && !strings.HasPrefix(runtime.Version(), "devel") {
		log.Println("You have Go version", runtime.Version())
		log.Println("go-ethereum requires at least Go version 1.7 and cannot")
		log.Println("be compiled with an earlier version. Please upgrade your Go installation.")
		os.Exit(1)
	}
	// Compile packages given as arguments, or everything if there are no arguments.
	packages := []string{"./..."}
	if flag.NArg() > 0 {
		packages = flag.Args()
	}
	if *arch == "" || *arch == runtime.GOARCH {
		goinstall := goTool("install", buildFlags(env)...)
		goinstall.Args = append(goinstall.Args, "-v")
		goinstall.Args = append(goinstall.Args, packages...)
		build.MustRun(goinstall)
		return
	}
	// If we are cross compiling to ARMv5 ARMv6 or ARMv7, clean any prvious builds
	if *arch == "arm" {
		os.RemoveAll(filepath.Join(runtime.GOROOT(), "pkg", runtime.GOOS+"_arm"))
		for _, path := range filepath.SplitList(build.GOPATH()) {
			os.RemoveAll(filepath.Join(path, "pkg", runtime.GOOS+"_arm"))
		}
	}
	// Seems we are cross compiling, work around forbidden GOBIN
	goinstall := goToolArch(*arch, "install", buildFlags(env)...)
	goinstall.Args = append(goinstall.Args, "-v")
	goinstall.Args = append(goinstall.Args, []string{"-buildmode", "archive"}...)
	goinstall.Args = append(goinstall.Args, packages...)
	build.MustRun(goinstall)

	if cmds, err := ioutil.ReadDir("cmd"); err == nil {
		for _, cmd := range cmds {
			pkgs, err := parser.ParseDir(token.NewFileSet(), filepath.Join(".", "cmd", cmd.Name()), nil, parser.PackageClauseOnly)
			if err != nil {
				log.Fatal(err)
			}
			for name := range pkgs {
				if name == "main" {
					gobuild := goToolArch(*arch, "build", buildFlags(env)...)
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
	// Set gitCommit constant via link-time assignment.
	if env.Commit != "" {
		flags = append(flags, "-ldflags", "-X main.gitCommit="+env.Commit)
	}
	return flags
}

func goTool(subcmd string, args ...string) *exec.Cmd {
	return goToolArch(runtime.GOARCH, subcmd, args...)
}

func goToolArch(arch string, subcmd string, args ...string) *exec.Cmd {
	gocmd := filepath.Join(runtime.GOROOT(), "bin", "go")
	cmd := exec.Command(gocmd, subcmd)
	cmd.Args = append(cmd.Args, args...)

	if subcmd == "build" || subcmd == "install" || subcmd == "test" {
		// Go CGO has a Windows linker error prior to 1.8 (https://github.com/golang/go/issues/8756).
		// Work around issue by allowing multiple definitions for <1.8 builds.
		if runtime.GOOS == "windows" && runtime.Version() < "go1.8" {
			cmd.Args = append(cmd.Args, []string{"-ldflags", "-extldflags -Wl,--allow-multiple-definition"}...)
		}
	}
	cmd.Env = []string{"GOPATH=" + build.GOPATH()}
	if arch == "" || arch == runtime.GOARCH {
		cmd.Env = append(cmd.Env, "GOBIN="+GOBIN)
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		cmd.Env = append(cmd.Env, "GOARCH="+arch)
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
		misspell = flag.Bool("misspell", false, "Whether to run the spell checker")
		coverage = flag.Bool("coverage", false, "Whether to record code coverage")
	)
	flag.CommandLine.Parse(cmdline)

	packages := []string{"./..."}
	if len(flag.CommandLine.Args()) > 0 {
		packages = flag.CommandLine.Args()
	}
	if len(packages) == 1 && packages[0] == "./..." {
		// Resolve ./... manually since go vet will fail on vendored stuff
		out, err := goTool("list", "./...").CombinedOutput()
		if err != nil {
			log.Fatalf("package listing failed: %v\n%s", err, string(out))
		}
		packages = []string{}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "vendor") {
				packages = append(packages, strings.TrimSpace(line))
			}
		}
	}
	// Run analysis tools before the tests.
	if *vet {
		build.MustRun(goTool("vet", packages...))
	}
	if *misspell {
		spellcheck(packages)
	}
	// Run the actual tests.
	gotest := goTool("test")
	// Test a single package at a time. CI builders are slow
	// and some tests run into timeouts under load.
	gotest.Args = append(gotest.Args, "-p", "1")
	if *coverage {
		gotest.Args = append(gotest.Args, "-covermode=atomic", "-cover")
	}
	gotest.Args = append(gotest.Args, packages...)
	build.MustRun(gotest)
}

// spellcheck runs the client9/misspell spellchecker package on all Go, Cgo and
// test files in the requested packages.
func spellcheck(packages []string) {
	// Ensure the spellchecker is available
	build.MustRun(goTool("get", "github.com/client9/misspell/cmd/misspell"))

	// Windows chokes on long argument lists, check packages individually
	for _, pkg := range packages {
		// The spell checker doesn't work on packages, gather all .go files for it
		out, err := goTool("list", "-f", "{{.Dir}}{{range .GoFiles}}\n{{.}}{{end}}{{range .CgoFiles}}\n{{.}}{{end}}{{range .TestGoFiles}}\n{{.}}{{end}}", pkg).CombinedOutput()
		if err != nil {
			log.Fatalf("source file listing failed: %v\n%s", err, string(out))
		}
		// Retrieve the folder and assemble the source list
		lines := strings.Split(string(out), "\n")
		root := lines[0]

		sources := make([]string, 0, len(lines)-1)
		for _, line := range lines[1:] {
			if line = strings.TrimSpace(line); line != "" {
				sources = append(sources, filepath.Join(root, line))
			}
		}
		// Run the spell checker for this particular package
		build.MustRunCommand(filepath.Join(GOBIN, "misspell"), append([]string{"-error"}, sources...)...)
	}
}

// Release Packaging

func doArchive(cmdline []string) {
	var (
		arch   = flag.String("arch", runtime.GOARCH, "Architecture cross packaging")
		atype  = flag.String("type", "zip", "Type of archive to write (zip|tar)")
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. LINUX_SIGNING_KEY)`)
		upload = flag.String("upload", "", `Destination to upload the archives (usually "gethstore/builds")`)
		ext    string
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
		base     = archiveBasename(*arch, env)
		geth     = "geth-" + base + ext
		alltools = "geth-alltools-" + base + ext
	)
	maybeSkipArchive(env)
	if err := build.WriteArchive(geth, gethArchiveFiles); err != nil {
		log.Fatal(err)
	}
	if err := build.WriteArchive(alltools, allToolsArchiveFiles); err != nil {
		log.Fatal(err)
	}
	for _, archive := range []string{geth, alltools} {
		if err := archiveUpload(archive, *upload, *signer); err != nil {
			log.Fatal(err)
		}
	}
}

func archiveBasename(arch string, env build.Environment) string {
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
	return platform + "-" + archiveVersion(env)
}

func archiveVersion(env build.Environment) string {
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-unstable"
	}
	if env.Commit != "" {
		version += "-" + env.Commit[:8]
	}
	return version
}

func archiveUpload(archive string, blobstore string, signer string) error {
	// If signing was requested, generate the signature files
	if signer != "" {
		pgpkey, err := base64.StdEncoding.DecodeString(os.Getenv(signer))
		if err != nil {
			return fmt.Errorf("invalid base64 %s", signer)
		}
		if err := build.PGPSignFile(archive, archive+".asc", string(pgpkey)); err != nil {
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
	}
	return nil
}

// skips archiving for some build configurations.
func maybeSkipArchive(env build.Environment) {
	if env.IsPullRequest {
		log.Printf("skipping because this is a PR build")
		os.Exit(0)
	}
	if env.Branch != "master" && !strings.HasPrefix(env.Tag, "v1.") {
		log.Printf("skipping because branch %q, tag %q is not on the whitelist", env.Branch, env.Tag)
		os.Exit(0)
	}
}

// Debian Packaging

func doDebianSource(cmdline []string) {
	var (
		signer  = flag.String("signer", "", `Signing key name, also used as package author`)
		upload  = flag.String("upload", "", `Where to upload the source package (usually "ppa:ethereum/ethereum")`)
		workdir = flag.String("workdir", "", `Output directory for packages (uses temp dir if unset)`)
		now     = time.Now()
	)
	flag.CommandLine.Parse(cmdline)
	*workdir = makeWorkdir(*workdir)
	env := build.Env()
	maybeSkipArchive(env)

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

	// Create the packages.
	for _, distro := range debDistros {
		meta := newDebMetadata(distro, *signer, env, now)
		pkgdir := stageDebianSource(*workdir, meta)
		debuild := exec.Command("debuild", "-S", "-sa", "-us", "-uc")
		debuild.Dir = pkgdir
		build.MustRun(debuild)

		changes := fmt.Sprintf("%s_%s_source.changes", meta.Name(), meta.VersionString())
		changes = filepath.Join(*workdir, changes)
		if *signer != "" {
			build.MustRunCommand("debsign", changes)
		}
		if *upload != "" {
			build.MustRunCommand("dput", *upload, changes)
		}
	}
}

func makeWorkdir(wdflag string) string {
	var err error
	if wdflag != "" {
		err = os.MkdirAll(wdflag, 0744)
	} else {
		wdflag, err = ioutil.TempDir("", "geth-build-")
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

type debMetadata struct {
	Env build.Environment

	// go-ethereum version being built. Note that this
	// is not the debian package version. The package version
	// is constructed by VersionString.
	Version string

	Author       string // "name <email>", also selects signing key
	Distro, Time string
	Executables  []debExecutable
}

type debExecutable struct {
	Name, Description string
}

func newDebMetadata(distro, author string, env build.Environment, t time.Time) debMetadata {
	if author == "" {
		// No signing key, use default author.
		author = "Ethereum Builds <fjl@ethereum.org>"
	}
	return debMetadata{
		Env:         env,
		Author:      author,
		Distro:      distro,
		Version:     build.VERSION(),
		Time:        t.Format(time.RFC1123Z),
		Executables: debExecutables,
	}
}

// Name returns the name of the metapackage that depends
// on all executable packages.
func (meta debMetadata) Name() string {
	if isUnstableBuild(meta.Env) {
		return "ethereum-unstable"
	}
	return "ethereum"
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
		return exe.Name + "-unstable"
	}
	return exe.Name
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
		install := filepath.Join(debian, meta.ExeName(exe)+".install")
		docs := filepath.Join(debian, meta.ExeName(exe)+".docs")
		build.Render("build/deb.install", install, 0644, exe)
		build.Render("build/deb.docs", docs, 0644, exe)
	}

	return pkgdir
}

// Windows installer

func doWindowsInstaller(cmdline []string) {
	// Parse the flags and make skip installer generation on PRs
	var (
		arch    = flag.String("arch", runtime.GOARCH, "Architecture for cross build packaging")
		signer  = flag.String("signer", "", `Environment variable holding the signing key (e.g. WINDOWS_SIGNING_KEY)`)
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
	build.CopyFile(filepath.Join(*workdir, "SimpleFC.dll"), "build/nsis.simplefc.dll", 0755)
	build.CopyFile(filepath.Join(*workdir, "COPYING"), "COPYING", 0755)

	// Build the installer. This assumes that all the needed files have been previously
	// built (don't mix building and packaging to keep cross compilation complexity to a
	// minimum).
	version := strings.Split(build.VERSION(), ".")
	if env.Commit != "" {
		version[2] += "-" + env.Commit[:8]
	}
	installer, _ := filepath.Abs("geth-" + archiveBasename(*arch, env) + ".exe")
	build.MustRunCommand("makensis.exe",
		"/DOUTPUTFILE="+installer,
		"/DMAJORVERSION="+version[0],
		"/DMINORVERSION="+version[1],
		"/DBUILDVERSION="+version[2],
		"/DARCH="+*arch,
		filepath.Join(*workdir, "geth.nsi"),
	)

	// Sign and publish installer.
	if err := archiveUpload(installer, *upload, *signer); err != nil {
		log.Fatal(err)
	}
}

// Android archives

func doAndroidArchive(cmdline []string) {
	var (
		local  = flag.Bool("local", false, `Flag whether we're only doing a local build (skip Maven artifacts)`)
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. ANDROID_SIGNING_KEY)`)
		deploy = flag.String("deploy", "", `Destination to deploy the archive (usually "https://oss.sonatype.org")`)
		upload = flag.String("upload", "", `Destination to upload the archive (usually "gethstore/builds")`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Sanity check that the SDK and NDK are installed and set
	if os.Getenv("ANDROID_HOME") == "" {
		log.Fatal("Please ensure ANDROID_HOME points to your Android SDK")
	}
	if os.Getenv("ANDROID_NDK") == "" {
		log.Fatal("Please ensure ANDROID_NDK points to your Android NDK")
	}
	// Build the Android archive and Maven resources
	build.MustRun(goTool("get", "golang.org/x/mobile/cmd/gomobile"))
	build.MustRun(gomobileTool("init", "--ndk", os.Getenv("ANDROID_NDK")))
	build.MustRun(gomobileTool("bind", "--target", "android", "--javapkg", "org.ethereum", "-v", "github.com/ethereum/go-ethereum/mobile"))

	if *local {
		// If we're building locally, copy bundle to build dir and skip Maven
		os.Rename("geth.aar", filepath.Join(GOBIN, "geth.aar"))
		return
	}
	meta := newMavenMetadata(env)
	build.Render("build/mvn.pom", meta.Package+".pom", 0755, meta)

	// Skip Maven deploy and Azure upload for PR builds
	maybeSkipArchive(env)

	// Sign and upload the archive to Azure
	archive := "geth-" + archiveBasename("android", env) + ".aar"
	os.Rename("geth.aar", archive)

	if err := archiveUpload(archive, *upload, *signer); err != nil {
		log.Fatal(err)
	}
	// Sign and upload all the artifacts to Maven Central
	os.Rename(archive, meta.Package+".aar")
	if *signer != "" && *deploy != "" {
		// Import the signing key into the local GPG instance
		if b64key := os.Getenv(*signer); b64key != "" {
			key, err := base64.StdEncoding.DecodeString(b64key)
			if err != nil {
				log.Fatalf("invalid base64 %s", *signer)
			}
			gpg := exec.Command("gpg", "--import")
			gpg.Stdin = bytes.NewReader(key)
			build.MustRun(gpg)
		}
		// Upload the artifacts to Sonatype and/or Maven Central
		repo := *deploy + "/service/local/staging/deploy/maven2"
		if meta.Develop {
			repo = *deploy + "/content/repositories/snapshots"
		}
		build.MustRunCommand("mvn", "gpg:sign-and-deploy-file",
			"-settings=build/mvn.settings", "-Durl="+repo, "-DrepositoryId=ossrh",
			"-DpomFile="+meta.Package+".pom", "-Dfile="+meta.Package+".aar")
	}
}

func gomobileTool(subcmd string, args ...string) *exec.Cmd {
	cmd := exec.Command(filepath.Join(GOBIN, "gomobile"), subcmd)
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = []string{
		"GOPATH=" + build.GOPATH(),
	}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOPATH=") {
			continue
		}
		cmd.Env = append(cmd.Env, e)
	}
	return cmd
}

type mavenMetadata struct {
	Version      string
	Package      string
	Develop      bool
	Contributors []mavenContributor
}

type mavenContributor struct {
	Name  string
	Email string
}

func newMavenMetadata(env build.Environment) mavenMetadata {
	// Collect the list of authors from the repo root
	contribs := []mavenContributor{}
	if authors, err := os.Open("AUTHORS"); err == nil {
		defer authors.Close()

		scanner := bufio.NewScanner(authors)
		for scanner.Scan() {
			// Skip any whitespace from the authors list
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line[0] == '#' {
				continue
			}
			// Split the author and insert as a contributor
			re := regexp.MustCompile("([^<]+) <(.+)>")
			parts := re.FindStringSubmatch(line)
			if len(parts) == 3 {
				contribs = append(contribs, mavenContributor{Name: parts[1], Email: parts[2]})
			}
		}
	}
	// Render the version and package strings
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-SNAPSHOT"
	}
	return mavenMetadata{
		Version:      version,
		Package:      "geth-" + version,
		Develop:      isUnstableBuild(env),
		Contributors: contribs,
	}
}

// XCode frameworks

func doXCodeFramework(cmdline []string) {
	var (
		local  = flag.Bool("local", false, `Flag whether we're only doing a local build (skip Maven artifacts)`)
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. IOS_SIGNING_KEY)`)
		deploy = flag.String("deploy", "", `Destination to deploy the archive (usually "trunk")`)
		upload = flag.String("upload", "", `Destination to upload the archives (usually "gethstore/builds")`)
	)
	flag.CommandLine.Parse(cmdline)
	env := build.Env()

	// Build the iOS XCode framework
	build.MustRun(goTool("get", "golang.org/x/mobile/cmd/gomobile"))
	build.MustRun(gomobileTool("init"))
	bind := gomobileTool("bind", "--target", "ios", "--tags", "ios", "-v", "github.com/ethereum/go-ethereum/mobile")

	if *local {
		// If we're building locally, use the build folder and stop afterwards
		bind.Dir, _ = filepath.Abs(GOBIN)
		build.MustRun(bind)
		return
	}
	archive := "geth-" + archiveBasename("ios", env)
	if err := os.Mkdir(archive, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	bind.Dir, _ = filepath.Abs(archive)
	build.MustRun(bind)
	build.MustRunCommand("tar", "-zcvf", archive+".tar.gz", archive)

	// Skip CocoaPods deploy and Azure upload for PR builds
	maybeSkipArchive(env)

	// Sign and upload the framework to Azure
	if err := archiveUpload(archive+".tar.gz", *upload, *signer); err != nil {
		log.Fatal(err)
	}
	// Prepare and upload a PodSpec to CocoaPods
	if *deploy != "" {
		meta := newPodMetadata(env, archive)
		build.Render("build/pod.podspec", "Geth.podspec", 0755, meta)
		build.MustRunCommand("pod", *deploy, "push", "Geth.podspec", "--allow-warnings", "--verbose")
	}
}

type podMetadata struct {
	Version      string
	Commit       string
	Archive      string
	Contributors []podContributor
}

type podContributor struct {
	Name  string
	Email string
}

func newPodMetadata(env build.Environment, archive string) podMetadata {
	// Collect the list of authors from the repo root
	contribs := []podContributor{}
	if authors, err := os.Open("AUTHORS"); err == nil {
		defer authors.Close()

		scanner := bufio.NewScanner(authors)
		for scanner.Scan() {
			// Skip any whitespace from the authors list
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line[0] == '#' {
				continue
			}
			// Split the author and insert as a contributor
			re := regexp.MustCompile("([^<]+) <(.+)>")
			parts := re.FindStringSubmatch(line)
			if len(parts) == 3 {
				contribs = append(contribs, podContributor{Name: parts[1], Email: parts[2]})
			}
		}
	}
	version := build.VERSION()
	if isUnstableBuild(env) {
		version += "-unstable." + env.Buildnum
	}
	return podMetadata{
		Archive:      archive,
		Version:      version,
		Commit:       env.Commit,
		Contributors: contribs,
	}
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
