// Copyright 2015 Google Inc.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid

import (
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	var stringTest string = "f47ac10b-58cc-0372-8567-0e02b2c3d479"
	var byteTest []byte = Parse(stringTest)
	var badTypeTest int = 6
	var invalidTest string = "f47ac10b-58cc-0372-8567-0e02b2c3d4"
	var invalidByteTest []byte = Parse(invalidTest)

	var uuid UUID
	err := (&uuid).Scan(stringTest)
	if err != nil {
		t.Fatal(err)
	}

	err = (&uuid).Scan(byteTest)
	if err != nil {
		t.Fatal(err)
	}

	err = (&uuid).Scan(badTypeTest)
	if err == nil {
		t.Error("int correctly parsed and shouldn't have")
	}
	if !strings.Contains(err.Error(), "unable to scan type") {
		t.Error("attempting to parse an int returned an incorrect error message")
	}

	err = (&uuid).Scan(invalidTest)
	if err == nil {
		t.Error("invalid uuid was parsed without error")
	}
	if !strings.Contains(err.Error(), "invalid UUID") {
		t.Error("attempting to parse an invalid UUID returned an incorrect error message")
	}

	err = (&uuid).Scan(invalidByteTest)
	if err == nil {
		t.Error("invalid byte uuid was parsed without error")
	}
	if !strings.Contains(err.Error(), "invalid UUID") {
		t.Error("attempting to parse an invalid byte UUID returned an incorrect error message")
	}
}
