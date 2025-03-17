// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package release

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/params"

	_ "embed"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

var (
	//go:embed cherrypicks
	cherryPicks  string
	lineFormatRE = regexp.MustCompile(`^([a-fA-F0-9]{40}) # (.*)$`)
)

type parsedLine struct {
	hash, commitMsg string
}

func parseCherryPicks(t *testing.T) (rawLines []string, lines []parsedLine) {
	t.Helper()
	for i, line := range strings.Split(cherryPicks, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch matches := lineFormatRE.FindStringSubmatch(line); len(matches) {
		case 3:
			rawLines = append(rawLines, line)
			lines = append(lines, parsedLine{
				hash:      matches[1],
				commitMsg: matches[2],
			})

		default:
			t.Errorf("Line %d is improperly formatted: %s", i, line)
		}
	}
	return rawLines, lines
}

func TestCherryPicksFormat(t *testing.T) {
	rawLines, lines := parseCherryPicks(t)
	if t.Failed() {
		t.Fatalf("Required line regexp: %s", lineFormatRE.String())
	}

	commits := make([]struct {
		obj  *object.Commit
		line parsedLine
	}, len(lines))

	repo := openGitRepo(t)
	for i, line := range lines {
		obj, err := repo.CommitObject(plumbing.NewHash(line.hash))
		require.NoErrorf(t, err, "%T.CommitObject(%q)", repo, line.hash)

		commits[i].obj = obj
		commits[i].line = line
	}
	sort.Slice(commits, func(i, j int) bool {
		ci, cj := commits[i].obj, commits[j].obj
		return ci.Committer.When.Before(cj.Committer.When)
	})

	var want []string
	for _, c := range commits {
		msg := strings.Split(c.obj.Message, "\n")[0]
		want = append(
			want,
			fmt.Sprintf("%s # %s", c.line.hash, msg),
		)
	}
	if diff := cmp.Diff(want, rawLines); diff != "" {
		t.Errorf("Commits in `cherrypicks` file out of order or have incorrect commit message(s);\n(-want +got):\n%s", diff)
		t.Logf("To fix, copy:\n%s", strings.Join(want, "\n"))
	}
}

const (
	defaultBranch       = "main"
	releaseBranchPrefix = "release/"
)

var triggerOrPRTargetBranch = flag.String(
	"target_branch",
	defaultBranch,
	"Target branch if triggered by a PR (github.base_ref), otherwise triggering branch (github.ref)",
)

func TestBranchProperties(t *testing.T) {
	branch := strings.TrimPrefix(*triggerOrPRTargetBranch, "refs/heads/")

	switch {
	case branch == defaultBranch:
		if rt := params.LibEVMReleaseType; rt.ForReleaseBranch() {
			t.Errorf("On default branch; params.LibEVMReleaseType = %q, which is reserved for release branches", rt)
		}

	case strings.HasPrefix(branch, releaseBranchPrefix):
		testReleaseBranch(t, branch)

	default:
		t.Logf("Branch %q is neither default nor release branch", branch)
	}
}

// testReleaseBranch asserts invariant properties of release branches:
//
//  1. They are named release/v${libevm-version};
//  2. The libevm version's [params.ReleaseType] is appropriate for a release
//     branch; and
//  3. The commit history is a "linear fork" off the default branch, with only
//     certain allowable commits.
//
// We define a "linear fork" as there being a single ancestral commit at which
// the release branch diverged from the default branch, with no merge commits
// after this divergence:
//
//	______________ main
//	    \___       release/*
//
// The commits in the release branch that are not in the default branch MUST be:
//
//  1. The cherry-pick commits embedded as [cherryPicks], in order; then
//  2. A single, final commit to change the libevm version.
//
// testReleaseBranch assumes that the git HEAD currently points at either
// `targetBranch` itself, or at a candidate (i.e. PR source) for said branch.
func testReleaseBranch(t *testing.T, targetBranch string) {
	t.Run("branch_name", func(t *testing.T) {
		want := fmt.Sprintf("%sv%s", releaseBranchPrefix, params.LibEVMVersion) // prefix already includes /
		assert.Equal(t, want, targetBranch)

		if rt := params.LibEVMReleaseType; !rt.ForReleaseBranch() {
			t.Errorf("On release branch; params.LibEVMReleaseType = %q, which is unsuitable for release branches", rt)
		}
	})

	t.Run("commit_history", func(t *testing.T) {
		repo := openGitRepo(t)
		headRef, err := repo.Head()
		require.NoErrorf(t, err, "%T.Head()", repo)

		head := commitFromRef(t, repo, headRef)
		main := commitFromBranchName(t, repo, defaultBranch)

		closestCommonAncestors, err := head.MergeBase(main)
		require.NoError(t, err)
		require.Lenf(t, closestCommonAncestors, 1, `number of "closest common ancestors" of HEAD (%v) and %q (%v)`, head.Hash, defaultBranch, main.Hash)
		// Not to be confused with the GitHub concept of a (repo) fork.
		fork := closestCommonAncestors[0]
		t.Logf("Forked from %q at commit %v (%s)", defaultBranch, fork.Hash, commitMsgFirstLine(fork))

		history, err := repo.Log(&git.LogOptions{
			Order: git.LogOrderDFS,
		})
		require.NoErrorf(t, err, "%T.Log()", repo)
		newCommits := linearCommitsSince(t, history, fork)
		logCommits(t, "History since fork from default branch", newCommits)

		t.Run("cherry_picked_commits", func(t *testing.T) {
			_, cherryPick := parseCherryPicks(t)
			wantCommits := commitsFromHashes(t, repo, cherryPick, fork)
			logCommits(t, "Expected cherry-picks", wantCommits)
			if got, want := len(newCommits), len(wantCommits)+1; got != want {
				t.Fatalf("Got %d commits since fork from default; want number to be cherry-picked plus one (%d)", got, want)
			}

			opt := compareCherryPickedCommits()
			if diff := cmp.Diff(wantCommits, newCommits[:len(wantCommits)], opt); diff != "" {
				t.Fatalf("Cherry-picked commits for release branch (-want +got):\n%s", diff)
			}
		})

		t.Run("final_commit", func(t *testing.T) {
			n := len(newCommits)
			last := newCommits[n-1]
			penultimate := fork
			if n >= 2 {
				penultimate = newCommits[n-2]
			}

			lastCommitDiffs, err := object.DiffTree(
				treeFromCommit(t, last),
				treeFromCommit(t, penultimate),
			)
			require.NoErrorf(t, err, "object.DiffTree(commits = [%v, %v])", last.Hash, penultimate.Hash)

			allowedFileModifications := map[string]bool{
				"version.libevm.go":      true,
				"version.libevm_test.go": true,
			}
			testFinalCommitChanges(t, lastCommitDiffs, allowedFileModifications)
		})
	})
}

func openGitRepo(t *testing.T) *git.Repository {
	t.Helper()

	opts := &git.PlainOpenOptions{DetectDotGit: true}
	repo, err := git.PlainOpenWithOptions("./", opts)
	require.NoErrorf(t, err, "git.PlainOpenWithOptions(./, %+v", opts)

	fetch := &git.FetchOptions{
		RemoteURL: "https://github.com/ethereum/go-ethereum.git",
	}
	err = repo.Fetch(fetch)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		t.Fatalf("%T.Fetch(%+v) error %v", repo, fetch, err)
	}

	return repo
}

func commitFromRef(t *testing.T, repo *git.Repository, ref *plumbing.Reference) *object.Commit {
	t.Helper()
	c, err := repo.CommitObject(ref.Hash())
	require.NoErrorf(t, err, "%T.CommitObject(%v)", repo, ref.Hash())
	return c
}

func commitFromBranchName(t *testing.T, repo *git.Repository, name string) *object.Commit {
	t.Helper()

	branch, err := repo.Branch(name)
	require.NoErrorf(t, err, "%T.Branch(%q)", repo, name)
	ref, err := repo.Reference(branch.Merge, false)
	require.NoErrorf(t, err, "%T.Reference(%v)", repo, branch.Merge)
	return commitFromRef(t, repo, ref)
}

func linearCommitsSince(t *testing.T, iter object.CommitIter, since *object.Commit) []*object.Commit {
	t.Helper()

	var commits []*object.Commit
	errReachedSince := fmt.Errorf("%T reached terminal commit %v", iter, since.Hash)

	err := iter.ForEach(func(c *object.Commit) error {
		if c.Hash == since.Hash {
			return errReachedSince
		}
		if n := len(c.ParentHashes); n != 1 {
			return fmt.Errorf("Non-linear history; commit %v has %d parents", c.Hash, n)
		}
		commits = append(commits, c)
		return nil
	})
	require.ErrorIs(t, err, errReachedSince)

	slices.Reverse(commits)
	return commits
}

func commitsFromHashes(t *testing.T, repo *git.Repository, lines []parsedLine, skipAncestorsOf *object.Commit) []*object.Commit {
	t.Helper()

	var commits []*object.Commit
	for _, l := range lines {
		c, err := repo.CommitObject(plumbing.NewHash(l.hash))
		require.NoError(t, err)

		skip, err := c.IsAncestor(skipAncestorsOf)
		require.NoError(t, err)
		if skip || c.Hash == skipAncestorsOf.Hash {
			continue
		}
		commits = append(commits, c)
	}

	return commits
}

func commitMsgFirstLine(c *object.Commit) string {
	return strings.Split(c.Message, "\n")[0]
}

func logCommits(t *testing.T, header string, commits []*object.Commit) {
	t.Logf("### %s (%d commits):", header, len(commits))
	for _, c := range commits {
		t.Logf("%s by %s", commitMsgFirstLine(c), c.Author.String())
	}
}

// compareCherryPickedCommits returns a [cmp.Transformer] that converts
// [object.Commit] instances into structs carrying only the pertinent commit
// properties that remain stable when cherry-picked. Note, however, that this
// does not include the actual diffs induced by cherry-picking.
func compareCherryPickedCommits() cmp.Option {
	type comparableCommit struct {
		MessageFirstLine, Author string
		Authored                 time.Time
	}

	return cmp.Transformer("gitCommit", func(c *object.Commit) comparableCommit {
		return comparableCommit{
			MessageFirstLine: commitMsgFirstLine(c),
			Author:           c.Author.String(),
			Authored:         c.Author.When,
		}
	})
}

func treeFromCommit(t *testing.T, c *object.Commit) *object.Tree {
	t.Helper()
	tree, err := c.Tree()
	require.NoErrorf(t, err, "%T.Tree()", c)
	return tree
}

func testFinalCommitChanges(t *testing.T, changes object.Changes, allowed map[string]bool) {
	for _, c := range changes {
		from, to, err := c.Files()
		require.NoErrorf(t, err, "%T.Files()", c)
		// We have a guarantee that at most one of `from` or `to` is nil,
		// but not both. Usage of `x.Name` MUST be guarded by the if
		// statement to avoid a panic.
		switch {
		case from == nil:
			t.Errorf("Created %q", to.Name)
		case to == nil:
			t.Errorf("Deleted %q", from.Name)
		case from.Name != to.Name:
			t.Errorf("Renamed %q to %q", from.Name, to.Name)
		case !allowed[filepath.Base(from.Name)]:
			// [object.File.Name] is documented as being either the name or a path,
			// depending on how it was generated. We only need to protect against
			// accidental changes to the wrong files, so it's sufficient to just
			// check the names.
			t.Errorf("Modified disallowed file %q", filepath.Base(from.Name))
		}
	}
}
