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
	"slices"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/params"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
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
//  3. The commit history since the default branch is only a single commit, to
//     change the version.
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
		t.Logf("HEAD is %v (%s)", head.Hash, commitMsgFirstLine(head))
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

		t.Run("final_commit", func(t *testing.T) {
			require.Len(t, newCommits, 1, "Single commit off default branch")
			last := newCommits[0]
			penultimate := fork

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

func commitMsgFirstLine(c *object.Commit) string {
	return strings.Split(c.Message, "\n")[0]
}

func logCommits(t *testing.T, header string, commits []*object.Commit) {
	t.Logf("### %s (%d commits):", header, len(commits))
	for _, c := range commits {
		t.Logf("%s by %s", commitMsgFirstLine(c), c.Author.String())
	}
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
