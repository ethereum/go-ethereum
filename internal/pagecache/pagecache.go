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

// Package pagecache keeps a bulk job's files, each written once and read
// once, from disturbing a node running beside it. Left to the kernel, their
// written data piles up as gigabytes of dirty pages flushed in bursts, which
// stall every read meanwhile, and both written and read data crowd the node's
// own pages out of the page cache. On Linux a Writer sends each window to
// disk as soon as it is written and a Reader drops each window behind it, a
// consuming Reader freeing it on disk as well; elsewhere both are the file
// itself.
package pagecache
