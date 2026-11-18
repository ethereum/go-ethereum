// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

var cloudflareTestTree = map[string]string{
	"n":                            "enrtree-root:v1 e=JWXYDBPXYWG6FX3GMDIBFA6CJ4 l=C7HRFPF3BLGF3YR4DY5KX3SMBE seq=1 sig=o908WmNp7LibOfPsr4btQwatZJ5URBr2ZAuxvK4UWHlsB9sUOTJQaGAlLPVAhM__XJesCHxLISo94z5Z2a463gA",
	"C7HRFPF3BLGF3YR4DY5KX3SMBE.n": "enrtree://AM5FCQLWIZX2QFPNJAP7VUERCCRNGRHWZG3YYHIUV7BVDQ5FDPRT2@morenodes.example.org",
	"JWXYDBPXYWG6FX3GMDIBFA6CJ4.n": "enrtree-branch:2XS2367YHAXJFGLZHVAWLQD4ZY,H4FHT4B454P6UXFD7JCYQ5PWDY,MHTDO6TMUBRIA2XWG5LUDACK24",
	"2XS2367YHAXJFGLZHVAWLQD4ZY.n": "enr:-HW4QOFzoVLaFJnNhbgMoDXPnOvcdVuj7pDpqRvh6BRDO68aVi5ZcjB3vzQRZH2IcLBGHzo8uUN3snqmgTiE56CH3AMBgmlkgnY0iXNlY3AyNTZrMaECC2_24YYkYHEgdzxlSNKQEnHhuNAbNlMlWJxrJxbAFvA",
	"H4FHT4B454P6UXFD7JCYQ5PWDY.n": "enr:-HW4QAggRauloj2SDLtIHN1XBkvhFZ1vtf1raYQp9TBW2RD5EEawDzbtSmlXUfnaHcvwOizhVYLtr7e6vw7NAf6mTuoCgmlkgnY0iXNlY3AyNTZrMaECjrXI8TLNXU0f8cthpAMxEshUyQlK-AM0PW2wfrnacNI",
	"MHTDO6TMUBRIA2XWG5LUDACK24.n": "enr:-HW4QLAYqmrwllBEnzWWs7I5Ev2IAs7x_dZlbYdRdMUx5EyKHDXp7AV5CkuPGUPdvbv1_Ms1CPfhcGCvSElSosZmyoqAgmlkgnY0iXNlY3AyNTZrMaECriawHKWdDRk2xeZkrOXBQ0dfMFLHY4eENZwdufn1S1o",
}

// summarize reduces changes to "action path" strings for comparison.
func summarize(changes []cloudflareChange) []string {
	var s []string
	for _, ch := range changes {
		s = append(s, ch.action+" "+ch.path)
	}
	return s
}

// This test checks that changes are ordered leaf-created -> root-changed ->
// stale-deleted, and that records of the currently published tree are retained.
func TestCloudflareChanges(t *testing.T) {
	t.Parallel()
	existing := map[string]cloudflare.DNSRecord{
		// Currently published tree.
		"n":                            {ID: "1", Content: "enrtree-root:v1 e=2KFJOGVXDQTXXUGBH7GS7NAAAI l=FDXN3SN67NA5DKA4J2GOK7BVQI seq=0 sig=v_-J_q_9ICQg5ztExFvLQhDBGMb0lZPJLhe3ts9LAcgqhOhtT3YFJsl8BWNDSwGtamUdR-9xl88_w-X42SVpjwE"},
		"2kfjogvxdqtxxugbh7gs7naaai.n": {ID: "2", Content: "enr:-HW4QO1ml1DdXLeZLsUxewnthhUy8eROqkDyoMTyavfks9JlYQIlMFEUoM78PovJDPQrAkrb3LRJ-vtrymDguKCOIAWAgmlkgnY0iXNlY3AyNTZrMaEDffaGfJzgGhUif1JqFruZlYmA31HzathLSWxfbq_QoQ4"},
		"fdxn3sn67na5dka4j2gok7bvqi.n": {ID: "3", Content: "enrtree-branch:"},
		// Also part of the new tree, unchanged.
		"2xs2367yhaxjfglzhvawlqd4zy.n": {ID: "4", Content: cloudflareTestTree["2XS2367YHAXJFGLZHVAWLQD4ZY.n"]},
		// Left over from an earlier deployment.
		"qhnwgu5dmt3erpthy7p2lu4bkm.n": {ID: "5", Content: "enrtree-branch:"},
	}
	want := []string{
		"create c7hrfpf3blgf3yr4dy5kx3smbe.n",
		"create h4fht4b454p6uxfd7jcyq5pwdy.n",
		"create jwxydbpxywg6fx3gmdibfa6cj4.n",
		"create mhtdo6tmubria2xwg5ludack24.n",
		"update n",
		"delete qhnwgu5dmt3erpthy7p2lu4bkm.n",
	}
	changes := computeCloudflareChanges("n", cloudflareTestTree, existing)
	if got := summarize(changes); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong changes:\n got  %q\n want %q", got, want)
	}
	if changes[4].old.ID != "1" || changes[4].value != cloudflareTestTree["n"] {
		t.Errorf("wrong root update: %+v", changes[4])
	}
	if changes[5].old.ID != "5" {
		t.Errorf("wrong record deleted: %+v", changes[5])
	}
}

// This test checks that on a fresh deployment the root is created last.
func TestCloudflareChangesNewDomain(t *testing.T) {
	t.Parallel()
	changes := computeCloudflareChanges("n", cloudflareTestTree, nil)
	if len(changes) != len(cloudflareTestTree) {
		t.Fatalf("wrong number of changes: got %d, want %d", len(changes), len(cloudflareTestTree))
	}
	for i, ch := range changes {
		if ch.action != "create" {
			t.Errorf("change %d: wrong action %q", i, ch.action)
		}
		isRoot := ch.path == "n"
		if isRoot != (i == len(changes)-1) {
			t.Errorf("change %d: root must be created last, got %q", i, ch.path)
		}
		wantTTL := treeNodeTTLCloudflare
		if isRoot {
			wantTTL = rootTTL
		}
		if ch.ttl != wantTTL {
			t.Errorf("change %d (%s): wrong TTL %d, want %d", i, ch.path, ch.ttl, wantTTL)
		}
	}
}

func TestPreviousTreeRecords(t *testing.T) {
	t.Parallel()
	existing := map[string]string{
		"n":                            "enrtree-root:v1 e=AAAAAAAAAAAAAAAAAAAAAAAAAA l=BBBBBBBBBBBBBBBBBBBBBBBBBB seq=3 sig=xxx",
		"aaaaaaaaaaaaaaaaaaaaaaaaaa.n": "enrtree-branch:CCCCCCCCCCCCCCCCCCCCCCCCCC,DDDDDDDDDDDDDDDDDDDDDDDDDD",
		"bbbbbbbbbbbbbbbbbbbbbbbbbb.n": "enrtree-branch:",
		"cccccccccccccccccccccccccc.n": "enrtree-branch:EEEEEEEEEEEEEEEEEEEEEEEEEE,AAAAAAAAAAAAAAAAAAAAAAAAAA", // includes a cycle
		"dddddddddddddddddddddddddd.n": "enr:-xyz",
		"eeeeeeeeeeeeeeeeeeeeeeeeee.n": "enr:-abc",
		"ffffffffffffffffffffffffff.n": "enr:-unreachable",
	}
	want := map[string]bool{
		"n":                            true,
		"aaaaaaaaaaaaaaaaaaaaaaaaaa.n": true,
		"bbbbbbbbbbbbbbbbbbbbbbbbbb.n": true,
		"cccccccccccccccccccccccccc.n": true,
		"dddddddddddddddddddddddddd.n": true,
		"eeeeeeeeeeeeeeeeeeeeeeeeee.n": true,
	}
	if got := previousTreeRecords("N", existing); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong records:\n got  %v\n want %v", got, want)
	}

	// Without a parseable root, nothing is retained.
	existing["n"] = "garbage"
	if got := previousTreeRecords("n", existing); len(got) != 0 {
		t.Fatalf("expected no records for invalid root, got %v", got)
	}
}

func TestUnsplitTXT(t *testing.T) {
	t.Parallel()
	long := "enr:" + strings.Repeat("x", 600)
	for _, v := range []string{"", "enrtree-branch:", long} {
		if got := unsplitTXT(splitTXT(v)); got != v {
			t.Errorf("roundtrip failed for %q: got %q", v, got)
		}
	}
	if got := unsplitTXT("plain"); got != "plain" {
		t.Errorf("unquoted input changed: %q", got)
	}
}
