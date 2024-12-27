// Copyright 2019 The go-ethereum Authors
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

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func Test_SplitTagsFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args string
		want map[string]string
	}{
		{
			"2 tags case",
			"host=localhost,bzzkey=123",
			map[string]string{
				"host":   "localhost",
				"bzzkey": "123",
			},
		},
		{
			"1 tag case",
			"host=localhost123",
			map[string]string{
				"host": "localhost123",
			},
		},
		{
			"empty case",
			"",
			map[string]string{},
		},
		{
			"garbage",
			"smth=smthelse=123",
			map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SplitTagsFlag(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTagsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWalkMatch(t *testing.T) {
	type args struct {
		root    string
		pattern string
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	test1Dir, _ := os.MkdirTemp(dir, "test1")
	test2Dir, _ := os.MkdirTemp(dir, "test2")
	err = os.WriteFile(filepath.Join(test1Dir, "test1.ldb"), []byte("hello"), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(test2Dir, "test2.abc"), []byte("hello"), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		os.RemoveAll(test1Dir)
		os.RemoveAll(test2Dir)
	}()

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"match test",
			args{
				root:    test1Dir,
				pattern: "*ldb",
			},
			[]string{filepath.Join(test1Dir, "test1.ldb")},
			false,
		},
		{
			"mismatch test",
			args{
				root:    test2Dir,
				pattern: "*ldb",
			},
			[]string{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WalkMatch(tt.args.root, tt.args.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("WalkMatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WalkMatch() got = %v, want %v", got, tt.want)
			}
		})
	}
}
