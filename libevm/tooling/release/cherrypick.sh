#!/usr/bin/env bash

#  Copyright 2025 the libevm authors.
#
#  The libevm additions to go-ethereum are free software: you can redistribute
#  them and/or modify them under the terms of the GNU Lesser General Public License
#  as published by the Free Software Foundation, either version 3 of the License,
#  or (at your option) any later version.
#
#  The libevm additions are distributed in the hope that they will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
#  General Public License for more details.
#
#  You should have received a copy of the GNU Lesser General Public License
#  along with the go-ethereum library. If not, see
#  <http://www.gnu.org/licenses/>.

# Usage: run `./cherrypick.sh` on a branch intended to become a release.
#
# Reads the contents of ./cherrypicks, filters out commits that are already
# ancestors of HEAD, and calls `git cherry-pick` with the remaining commit
# hashes.

set -eu;
set -o pipefail;

SELF_DIR=$(dirname "${0}")
# The format of the `cherrypicks` file is guaranteed by a test so we can use simple parsing here.
CHERRY_PICKS=$(< "${SELF_DIR}/cherrypicks" grep -Pv "^#" | awk '{print $1}')

commits=()
for commit in ${CHERRY_PICKS}; do
    git merge-base --is-ancestor "${commit}" HEAD && \
        echo "Skipping ${commit} already in history" && \
        continue;

    echo "Cherry-picking ${commit}";
    commits+=("${commit}");
done

if [[ -z "${commits[*]// }" ]]; then # $x// removes whitespace
    echo "No commits to cherry-pick";
    exit 0;
fi

git cherry-pick "${commits[@]}";
