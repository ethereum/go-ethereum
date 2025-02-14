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
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	_ "embed"
)

var (
	//go:embed cherrypicks
	cherryPicks  string
	lineFormatRE = regexp.MustCompile(`^([a-fA-F0-9]{40}) # (.*)$`)
)

func TestCherryPicksFormat(t *testing.T) {
	type parsedLine struct {
		hash, commitMsg string
	}
	var (
		rawLines []string
		lines    []parsedLine
	)

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
	if t.Failed() {
		t.Fatalf("Required line regexp: %s", lineFormatRE.String())
	}

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

	commits := make([]struct {
		obj  *object.Commit
		line parsedLine
	}, len(lines))

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
