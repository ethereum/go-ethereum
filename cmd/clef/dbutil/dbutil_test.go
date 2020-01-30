package dbutil

import (
	"database/sql"
	"log"
	"testing"

	"gotest.tools/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestReadConfigYAML(t *testing.T) {
	testMySQLConfig(t, "./dbutil_test_mysql.yaml")
	testPQConfig(t, "./dbutil_test_postgres.yaml")
}

func testMySQLConfig(t *testing.T, path string) {
	conf, err := readConfigYAML(path)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, conf.Adapter, "mysql")
	assert.Equal(t, conf.Username, "test")
	assert.Equal(t, conf.Password, "testpw")
	assert.Equal(t, conf.Protocol, "tcp")
	assert.Equal(t, conf.Host, "localhost")
	assert.Equal(t, conf.Port, "3306")
	assert.Equal(t, conf.Database, "testdb")
	assert.Equal(t, conf.Params["fakeparam"], "fakeval")

	// check DSN generation
	assert.Equal(t, conf.DataSourceName(), "test:testpw@tcp(localhost:3306)/testdb?fakeparam=fakeval")
}

func testPQConfig(t *testing.T, path string) {
	conf, err := readConfigYAML(path)
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, conf.Adapter, "postgres")
	assert.Equal(t, conf.Username, "test")
	assert.Equal(t, conf.Password, "testpw")
	assert.Equal(t, conf.Protocol, "")
	assert.Equal(t, conf.Host, "localhost")
	assert.Equal(t, conf.Port, "5432")
	assert.Equal(t, conf.Database, "pqtestdb")
	assert.Equal(t, conf.Params["fakeparam"], "fakevalpq")

	// check DSN generation
	// postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]
	assert.Equal(t, conf.DataSourceName(), "postgresql://test:testpw@localhost:5432/pqtestdb?fakeparam=fakevalpq")
}

func TestKVStoreOperations(t *testing.T) {
	kvstore, err := NewKVStore("./dbutil_test_sqlite3.yaml", PasswordTable)
	if err != nil {
		log.Fatal("Cannot initiate KVStore:", err)
	}

	// Put
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", "v2"
	k3, v3 := "k3", "v3"
	kvstore.Put(k1, v1)
	kvstore.Put(k2, v2)

	// Get
	v, _ := kvstore.Get(k1)
	assert.Equal(t, v, v1)
	v, _ = kvstore.Get(k2)
	assert.Equal(t, v, v2)

	// Del
	kvstore.Del(k1)
	_, err = kvstore.Get(k1)
	assert.Equal(t, err, sql.ErrNoRows)

	// Update
	kvstore.Put(k2, "updated")
	v, _ = kvstore.Get(k2)
	assert.Equal(t, v, "updated")

	// All
	kvstore.Put(k1, v1)
	kvstore.Put(k3, v3)
	keys := kvstore.All()
	assert.Equal(t, len(keys), 3)
	assert.Equal(t, keys[0], "k2")
}
