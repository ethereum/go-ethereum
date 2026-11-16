// Copyright 2026 The go-ethereum Authors
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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalEnvGit tests that LocalEnv finds the commit, the branch and the tag
// in a clone with packed refs and in a worktree, the commit on a detached head,
// and nothing in a directory that isn't the top of a working tree. Without git,
// it reads .git as before.
func TestLocalEnvGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	// Keep the user's git configuration (signing, hooks) and environment
	// (GIT_DIR and the like) out of the test.
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(strings.ToUpper(name), "GIT_") {
			t.Setenv(name, "") // restores it after the test
			os.Unsetenv(name)
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	sub := filepath.Join(repo, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	git(repo, "init", "-q", "-b", "main")
	git(repo, "add", ".")
	git(repo, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "first")
	git(repo, "tag", "main") // named like the branch
	commit := git(repo, "rev-parse", "HEAD")

	check := func(dir, wantCommit, wantBranch, wantTag string) {
		t.Helper()
		t.Chdir(dir)
		env := LocalEnv()
		if env.Commit != wantCommit || env.Branch != wantBranch || env.Tag != wantTag {
			t.Errorf("in %s: commit %q, branch %q, tag %q; want %q, %q, %q", dir, env.Commit, env.Branch, env.Tag, wantCommit, wantBranch, wantTag)
		}
	}
	path := os.Getenv("PATH")
	t.Setenv("PATH", "")
	check(repo, commit, "main", "") // no git: .git is read, and there's no tag lookup
	t.Setenv("PATH", path)

	git(repo, "pack-refs", "--all")
	check(repo, commit, "main", "main") // the branch's ref is packed
	worktree := filepath.Join(tmp, "worktree")
	git(repo, "worktree", "add", "-q", "-b", "feature", worktree)
	check(worktree, commit, "feature", "main") // .git is a file
	git(worktree, "checkout", "-q", "--detach")
	check(worktree, commit, "", "") // a detached head in a worktree
	check(sub, "", "", "")          // not the top of a working tree, as an unpacked source archive

	// On a detached head, a commit given by flag isn't looked up, so it needn't
	// be in the repository.
	*GitCommitFlag = strings.Repeat("1", 40)
	t.Cleanup(func() { *GitCommitFlag = "" })
	check(worktree, *GitCommitFlag, "", "")
	*GitCommitFlag = ""
}
