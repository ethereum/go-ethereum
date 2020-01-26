package storage

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ds  *DBStorage
	key string
)

func init() {
	key = "AES256Key-32Characters1234567890"
	tmpDir, _ := ioutil.TempDir("", "eth-encrypted-db-storge-test")
	ds, _ = NewDBStorage([]byte(key), "sqlite3", filepath.Join(tmpDir, "test.db"), "kps")
}

func TestDBStorage(t *testing.T) {
	// test Put
	k1, v1 := "k1", "v1"
	ds.Put(k1, v1)

	// test Get
	ret, err := ds.Get(k1)
	if err != nil || ret != v1 {
		t.Fatal("Get didn't return correct result")
	}

	// test Put when there's duplicate
	v2 := "v2"
	ds.Put(k1, v2)
	ret, err = ds.Get(k1)
	if err != nil || ret != v2 {
		t.Fatal("Get didn't return correct result")
	}

	// test Del
	ds.Del(k1)
	ret, err = ds.Get(k1)
	if err != ErrNotFound {
		t.Fatal("Del didn't work as expected")
	}
}

func TestSwappedKeysForDBStorage(t *testing.T) {
	ds.Put("k1", "v1")
	ds.Put("k2", "v2")

	// now make a modified copy
	swap := func() {
		creds1, _, _ := ds.queryRow("SELECT * FROM kps WHERE k = 'k1'")
		creds2, _, _ := ds.queryRow("SELECT * FROM kps WHERE k = 'k2'")
		ds.exec("UPDATE kps SET v = ? WHERE k = ?", creds1.val, "k2")
		ds.exec("UPDATE kps SET v = ? WHERE k = ?", creds2.val, "k1")
	}
	swap()
	if v, _ := ds.Get("k1"); v != "" {
		t.Errorf("swapped value should return empty")
	}
	swap()
	if v, _ := ds.Get("k1"); v != "v1" {
		t.Errorf(v)
		t.Errorf("double-swapped value should work fine")
	}
}
