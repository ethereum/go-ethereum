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

package build

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// These flags override values in build env.
	GitCommitFlag     = flag.String("git-commit", "", `Overrides git commit hash embedded into executables`)
	GitBranchFlag     = flag.String("git-branch", "", `Overrides git branch being built`)
	GitTagFlag        = flag.String("git-tag", "", `Overrides git tag being built`)
	BuildnumFlag      = flag.String("buildnum", "", `Overrides CI build number`)
	PullRequestFlag   = flag.Bool("pull-request", false, `Overrides pull request status of the build`)
	CronJobFlag       = flag.Bool("cron-job", false, `Overrides cron job status of the build`)
	UbuntuVersionFlag = flag.String("ubuntu", "", `Sets the ubuntu version being built for`)
)

// Environment contains metadata provided by the build environment.
type Environment struct {
	CI                        bool
	Name                      string // name of the environment
	Repo                      string // name of GitHub repo
	Commit, Date, Branch, Tag string // Git info
	Buildnum                  string
	UbuntuVersion             string // Ubuntu version being built for
	IsPullRequest             bool
	IsCronJob                 bool
}

func (env Environment) String() string {
	return fmt.Sprintf("%s env (commit:%s date:%s branch:%s tag:%s buildnum:%s pr:%t)",
		env.Name, env.Commit, env.Date, env.Branch, env.Tag, env.Buildnum, env.IsPullRequest)
}

// Env returns metadata about the current CI environment, falling back to LocalEnv
// if not running on CI.
func Env() Environment {
	switch {
	case os.Getenv("CI") == "true" && os.Getenv("TRAVIS") == "true":
		commit := os.Getenv("TRAVIS_PULL_REQUEST_SHA")
		if commit == "" {
			commit = os.Getenv("TRAVIS_COMMIT")
		}
		return Environment{
			CI:            true,
			Name:          "travis",
			Repo:          os.Getenv("TRAVIS_REPO_SLUG"),
			Commit:        commit,
			Date:          getDate(commit),
			Branch:        os.Getenv("TRAVIS_BRANCH"),
			Tag:           os.Getenv("TRAVIS_TAG"),
			Buildnum:      os.Getenv("TRAVIS_BUILD_NUMBER"),
			IsPullRequest: os.Getenv("TRAVIS_PULL_REQUEST") != "false",
			IsCronJob:     os.Getenv("TRAVIS_EVENT_TYPE") == "cron",
		}
	case os.Getenv("CI") == "True" && os.Getenv("APPVEYOR") == "True":
		commit := os.Getenv("APPVEYOR_PULL_REQUEST_HEAD_COMMIT")
		if commit == "" {
			commit = os.Getenv("APPVEYOR_REPO_COMMIT")
		}
		return Environment{
			CI:            true,
			Name:          "appveyor",
			Repo:          os.Getenv("APPVEYOR_REPO_NAME"),
			Commit:        commit,
			Date:          getDate(commit),
			Branch:        os.Getenv("APPVEYOR_REPO_BRANCH"),
			Tag:           os.Getenv("APPVEYOR_REPO_TAG_NAME"),
			Buildnum:      os.Getenv("APPVEYOR_BUILD_NUMBER"),
			IsPullRequest: os.Getenv("APPVEYOR_PULL_REQUEST_NUMBER") != "",
			IsCronJob:     os.Getenv("APPVEYOR_SCHEDULED_BUILD") == "True",
		}
	case os.Getenv("CI") == "true" && os.Getenv("GITHUB_ACTIONS") == "true":
		commit := os.Getenv("GITHUB_SHA")
		reftype := os.Getenv("GITHUB_REF_TYPE")
		isPR := os.Getenv("GITHUB_HEAD_REF") != ""
		tag := ""
		branch := ""
		switch {
		case isPR:
			branch = os.Getenv("GITHUB_BASE_REF")
		case reftype == "branch":
			branch = os.Getenv("GITHUB_REF_NAME")
		case reftype == "tag":
			tag = os.Getenv("GITHUB_REF_NAME")
		}
		return Environment{
			CI:            true,
			Name:          "github-actions",
			Repo:          os.Getenv("GITHUB_REPOSITORY"),
			Commit:        commit,
			Date:          getDate(commit),
			Branch:        branch,
			Tag:           tag,
			IsPullRequest: isPR,
			Buildnum:      os.Getenv("GITHUB_RUN_ID"),
			IsCronJob:     os.Getenv("GITHUB_EVENT_NAME") == "schedule",
		}

	default:
		return LocalEnv()
	}
}

// LocalEnv returns build environment metadata gathered from git.
func LocalEnv() Environment {
	env := applyEnvFlags(Environment{Name: "local", Repo: "ethereum/go-ethereum"})

	// Ask git instead of reading .git: in a worktree or a submodule, .git is a
	// file pointing to the repository, and refs may be packed.
	topInfo, err := os.Stat(gitOutput("rev-parse", "--show-toplevel"))
	if err != nil {
		// git is missing, refuses the repository or isn't in one, or gives a
		// path Go can't open (MSYS git on Windows): read .git as before.
		return localEnvFromFiles(env)
	}
	if wdInfo, err := os.Stat("."); err != nil || !os.SameFile(topInfo, wdInfo) {
		// Not the top of the working tree, as a source archive unpacked
		// inside another repository: don't take that repository's commit.
		return env
	}
	// Not symbolic-ref --short, which gives heads/master when a tag is
	// named master too.
	branch, ok := strings.CutPrefix(gitOutput("symbolic-ref", "-q", "HEAD"), "refs/heads/")
	if !ok {
		// In this case we are in "detached head" state
		// see: https://git-scm.com/docs/git-checkout#_detached_head
		if env.Commit == "" {
			env.Commit = gitOutput("rev-parse", "HEAD")
			env.Date = getDate(env.Commit)
		}
		return env
	}
	if env.Commit == "" {
		env.Commit = gitOutput("rev-parse", "HEAD")
	}
	env.Date = getDate(env.Commit)
	if env.Branch == "" {
		env.Branch = branch
	}
	if env.Tag == "" {
		env.Tag = firstLine(gitOutput("tag", "-l", "--points-at", "HEAD"))
	}
	return env
}

// localEnvFromFiles reads the commit and the branch from .git directly, for
// when git can't tell. It finds nothing in a worktree or a submodule, or when
// the branch's ref is packed.
func localEnvFromFiles(env Environment) Environment {
	head := readGitFile("HEAD")
	if fields := strings.Fields(head); len(fields) == 2 {
		head = fields[1]
	} else {
		// In this case we are in "detached head" state
		// see: https://git-scm.com/docs/git-checkout#_detached_head
		// Additional check required to verify, that file contains commit hash
		commitRe, _ := regexp.Compile("^([0-9a-f]{40})$")
		if commit := commitRe.FindString(head); commit != "" && env.Commit == "" {
			env.Commit = commit
			env.Date = getDate(env.Commit)
		}
		return env
	}
	if env.Commit == "" {
		env.Commit = readGitFile(head)
	}
	env.Date = getDate(env.Commit)
	if env.Branch == "" {
		if head != "HEAD" {
			env.Branch = strings.TrimPrefix(head, "refs/heads/")
		}
	}
	if info, err := os.Stat(".git/objects"); err == nil && info.IsDir() && env.Tag == "" {
		env.Tag = firstLine(RunGit("tag", "-l", "--points-at", "HEAD"))
	}
	return env
}

func firstLine(s string) string {
	return strings.Split(s, "\n")[0]
}

func getDate(commit string) string {
	if commit == "" {
		return ""
	}
	out := RunGit("show", "-s", "--format=%ct", commit)
	if out == "" {
		return ""
	}
	date, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("failed to parse git commit date: %v", err))
	}
	return time.Unix(date, 0).Format("20060102")
}

func applyEnvFlags(env Environment) Environment {
	if !flag.Parsed() {
		panic("you need to call flag.Parse before Env or LocalEnv")
	}
	if *GitCommitFlag != "" {
		env.Commit = *GitCommitFlag
	}
	if *GitBranchFlag != "" {
		env.Branch = *GitBranchFlag
	}
	if *GitTagFlag != "" {
		env.Tag = *GitTagFlag
	}
	if *BuildnumFlag != "" {
		env.Buildnum = *BuildnumFlag
	}
	if *PullRequestFlag {
		env.IsPullRequest = true
	}
	if *CronJobFlag {
		env.IsCronJob = true
	}
	if *UbuntuVersionFlag != "" {
		env.UbuntuVersion = *UbuntuVersionFlag
	}
	return env
}
