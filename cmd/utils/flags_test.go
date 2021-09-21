package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWalkMatch(t *testing.T) {
	type args struct {
		root    string
		pattern string
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	test1Dir, _ := ioutil.TempDir(dir, "test1")
	test2Dir, _ := ioutil.TempDir(dir, "test2")
	err = ioutil.WriteFile(filepath.Join(test1Dir, "test1.ldb"), []byte("hello"), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(test2Dir, "test2.abc"), []byte("hello"), os.ModePerm)
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
