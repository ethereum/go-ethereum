package crypto

// import (
// 	"github.com/ethereum/go-ethereum/ethdb"
// 	// "io/ioutil"
// 	"fmt"
// 	"os"
// 	"path"
// 	"testing"
// )

// // test if persistence layer works
// func TestDBKeyManager(t *testing.T) {
// 	memdb, _ := ethdb.NewMemDatabase()
// 	keyManager0 := NewDBKeyManager(memdb)
// 	err := keyManager0.Init("", 0, false)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	keyManager1 := NewDBKeyManager(memdb)
// 	err = keyManager1.Init("", 0, false)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	if string(keyManager0.PrivateKey()) != string(keyManager1.PrivateKey()) {
// 		t.Error("Expected private keys %x, %x, to be identical via db persistence", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	}
// 	err = keyManager1.Init("", 0, true)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	if string(keyManager0.PrivateKey()) == string(keyManager1.PrivateKey()) {
// 		t.Error("Expected private keys %x, %x, to be be different despite db persistence if force generate", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	}
// }

// func TestFileKeyManager(t *testing.T) {
// 	basedir0 := "/tmp/ethtest0"
// 	os.RemoveAll(basedir0)
// 	os.Mkdir(basedir0, 0777)

// 	keyManager0 := NewFileKeyManager(basedir0)
// 	err := keyManager0.Init("", 0, false)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}

// 	keyManager1 := NewFileKeyManager(basedir0)

// 	err = keyManager1.Init("", 0, false)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	if string(keyManager0.PrivateKey()) != string(keyManager1.PrivateKey()) {
// 		t.Error("Expected private keys %x, %x, to be identical via db persistence", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	}

// 	err = keyManager1.Init("", 0, true)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	if string(keyManager0.PrivateKey()) == string(keyManager1.PrivateKey()) {
// 		t.Error("Expected private keys %x, %x, to be be different despite db persistence if force generate", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	}
// }

// // cursor errors
// func TestCursorErrors(t *testing.T) {
// 	memdb, _ := ethdb.NewMemDatabase()
// 	keyManager0 := NewDBKeyManager(memdb)
// 	err := keyManager0.Init("", 0, false)
// 	err = keyManager0.Init("", 1, false)
// 	if err == nil {
// 		t.Error("Expected cursor error")
// 	}
// 	err = keyManager0.SetCursor(1)
// 	if err == nil {
// 		t.Error("Expected cursor error")
// 	}
// }

// func TestExportImport(t *testing.T) {
// 	memdb, _ := ethdb.NewMemDatabase()
// 	keyManager0 := NewDBKeyManager(memdb)
// 	err := keyManager0.Init("", 0, false)
// 	basedir0 := "/tmp/ethtest0"
// 	os.RemoveAll(basedir0)
// 	os.Mkdir(basedir0, 0777)
// 	keyManager0.Export(basedir0)

// 	keyManager1 := NewFileKeyManager(basedir0)
// 	err = keyManager1.Init("", 0, false)
// 	if err != nil {
// 		t.Error("Unexpected error: ", err)
// 	}
// 	fmt.Printf("keyRing: %v\n", keyManager0.KeyPair())
// 	fmt.Printf("keyRing: %v\n", keyManager1.KeyPair())
// 	if string(keyManager0.PrivateKey()) != string(keyManager1.PrivateKey()) {
// 		t.Error("Expected private keys %x, %x, to be identical via export to filestore basedir", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	}
// 	path.Join("")

// 	// memdb, _ = ethdb.NewMemDatabase()
// 	// keyManager2 := NewDBKeyManager(memdb)
// 	// err = keyManager2.InitFromSecretsFile("", 0, path.Join(basedir0, "default.prv"))
// 	// if err != nil {
// 	// 	t.Error("Unexpected error: ", err)
// 	// }
// 	// if string(keyManager0.PrivateKey()) != string(keyManager2.PrivateKey()) {
// 	// 	t.Error("Expected private keys %s, %s, to be identical via export/import prv", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	// }

// 	// memdb, _ = ethdb.NewMemDatabase()
// 	// keyManager3 := NewDBKeyManager(memdb)
// 	// err = keyManager3.InitFromSecretsFile("", 0, path.Join(basedir0, "default.mne"))
// 	// if err != nil {
// 	// 	t.Error("Unexpected error: ", err)
// 	// }
// 	// if string(keyManager0.PrivateKey()) != string(keyManager3.PrivateKey()) {
// 	// 	t.Error("Expected private keys %s, %s, to be identical via export/import mnemonic file", keyManager0.PrivateKey(), keyManager1.PrivateKey())
// 	// }
// }
