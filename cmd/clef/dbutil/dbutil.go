package dbutil

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/yaml.v2"

	// here we are adding multiple default supported db drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Keystore database default table names
const (
	AccountTable = "accounts"
)

var (
	// ErrNotFound is returned if an unknown key is attempted to be retrieved.
	ErrNotFound = errors.New("not found")
)

var (
	querySQL  string = "SELECT * FROM tableName WHERE k = ?"
	updateSQL string = "UPDATE tableName SET v = ? WHERE k = ?"
	insertSQL string = "INSERT INTO tableName (k, v) VALUES (?, ?)"
	deleteSQL string = "DELETE FROM tableName WHERE k = ?"
	countSQL  string = "SELECT COUNT(*) FROM tableName"
	allSQL    string = "SELECT k FROM tableName"
)

// NewKVStore returns a new instance of KVStore
func NewKVStore(path, table string) (*KVStore, error) {
	conf, err := readConfigYAML(path)
	if err != nil {
		return nil, err
	}

	// sql.Open only validates the input, but didn't create a connection
	db, err := sql.Open(conf.Adapter, conf.DataSourceName())
	if err != nil {
		log.Error("failed to validate driver: #{driverName}, #{dataSourceName}")
		db.Close()
		return nil, err
	}

	// Connects to the database and make sure it is ok, connection will be closed shortly since default MaxIdle is 0
	err = db.Ping()
	if err != nil {
		log.Error("failed to connect to database: #{dataSourceName}")
		db.Close()
		return nil, err
	}

	// set connection limits
	db.SetMaxOpenConns(5)

	err = initTable(conf.Adapter, table, db)
	if err != nil {
		return nil, err
	}

	return &KVStore{
		Conf:  conf,
		db:    db,
		Table: table,
	}, nil
}

func readConfigYAML(path string) (*DBConf, error) {
	// yaml.Unmarshal()
	yamlContent, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warn("Cannot read yaml file from file:", path)
		return nil, err
	}

	conf := &DBConf{}
	if err = yaml.Unmarshal([]byte(yamlContent), conf); err != nil {
		log.Warn("Cannot parse yaml config: ", err)
		return nil, err
	}

	return conf, nil
}

func initTable(driverName, tableName string, db *sql.DB) error {
	var err error
	switch driverName {
	case "postgres":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id SERIAL PRIMARY KEY,
	k VARCHAR(255) UNIQUE NOT NULL,
	v TEXT NOT NULL
)
		`, tableName))
	case "mysql":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INT AUTO_INCREMENT PRIMARY KEY,
	k VARCHAR(255) UNIQUE NOT NULL, 
	v TEXT NOT NULL
)
		`, tableName))
	case "sqlite3":
		_, err = db.Exec(fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY, 
	k TEXT, 
	v TEXT
)
		`, tableName))
	default:
		err = fmt.Errorf("unsupported driver type: %s", driverName)
	}
	return err
}

// KVStore is used for abstracting a generic database as a simple key value storage
type KVStore struct {
	Table string
	Conf  *DBConf
	db    *sql.DB
}

// Get returns the previously stored value, or an error if it does not exist or
// key is of 0-length.
func (kvstore *KVStore) Get(key string) (string, error) {
	sql := kvstore.adjustSQLPlaceholder(querySQL)
	v, err := kvstore.queryRow(sql, key)
	if err != nil {
		return "", err
	}
	return v, nil
}

// Put stores a value by key. 0-length keys results in noop.
func (kvstore *KVStore) Put(key, value string) error {
	if len(key) == 0 {
		return errors.New("0-length key")
	}
	if !kvstore.Exists(key) {
		return kvstore.insertRow(key, value)
	} else {
		return kvstore.updateRow(key, value)
	}
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (kvstore *KVStore) Del(key string) {
	sql := kvstore.adjustSQLPlaceholder(deleteSQL)
	kvstore.exec(sql, key)
}

// All returns all keys in the database
func (kvstore *KVStore) All() []string {
	size := kvstore.Size()
	sql := kvstore.adjustSQLPlaceholder(allSQL)
	rows, err := kvstore.db.Query(sql)
	if err != nil {
		log.Error("Error retrieving all keys: ", err)
		return nil
	}
	defer rows.Close()

	result := make([]string, size)
	row := DBRow{}
	index := 0
	for rows.Next() {
		err = rows.Scan(&row.key)
		if err != nil {
			log.Error("Cannot retrieve database row", err)
		}

		result[index] = row.key
		index++
	}
	return result
}

// Exists returns a boolean indicates if the key exists or not
func (kvstore *KVStore) Exists(key string) bool {
	v, err := kvstore.Get(key)
	return err == nil && v != ""
}

// Size returns number of entries that exists in the kvstore
func (kvstore *KVStore) Size() int {
	var size int
	sql := kvstore.adjustSQLPlaceholder(countSQL)
	err := kvstore.db.QueryRow(sql).Scan(&size)
	if err != nil {
		log.Error("Error counting key numbers: ", err)
		return 0
	}
	return size
}

func (kvstore *KVStore) insertRow(key, value string) error {
	sql := kvstore.adjustSQLPlaceholder(insertSQL)
	return kvstore.exec(sql, key, value)
}

func (kvstore *KVStore) updateRow(key, value string) error {
	sql := kvstore.adjustSQLPlaceholder(updateSQL)
	return kvstore.exec(sql, value, key)
}

func (kvstore *KVStore) adjustSQLPlaceholder(sql string) string {
	switch kvstore.Conf.Adapter {
	case "postgres":
		params := strings.Count(sql, "?")
		for i := 1; i <= params; i++ {
			sql = strings.Replace(sql, "?", fmt.Sprintf("$%d", i), 1)
		}
	default:
		// for MS SQL Server / MySQL / SQLite
		// since they're already using ? as placeholder, do nothing
	}

	return strings.ReplaceAll(sql, "tableName", kvstore.Table)
}

func (kvstore *KVStore) queryRow(query string, args ...interface{}) (string, error) {
	row := DBRow{}
	err := kvstore.db.QueryRow(query, args...).Scan(&row.id, &row.key, &row.val)
	if err != nil {
		log.Warn("No existing result is present in the database. It should be fine if you're creating a new entry")
		return "", err
	}
	return row.val, nil
}

func (kvstore *KVStore) exec(query string, args ...interface{}) error {
	_, err := kvstore.db.Exec(query, args...)
	if err != nil {
		log.Warn("Failed to execute sql", query, args)
		return err
	}
	return nil
}

// DBRow is the structure to hold a row of our configuration database
// table schemas for all three tables (kps, js, config) are the same
type DBRow struct {
	id  int
	key string
	val string
}

// DBConf is used to hold database configuration
type DBConf struct {
	Adapter  string            `yaml:"adapter"`
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Host     string            `yaml:"host"`
	Port     string            `yaml:"port"`
	Database string            `yaml:"database"`
	Protocol string            `yaml:"protocol"`
	Params   map[string]string `yaml:"params,omitempty"`
}

// DataSourceName returns the valid dsn for passing to sql.Open
func (conf *DBConf) DataSourceName() string {
	var dsn string
	switch conf.Adapter {
	case "mysql":
		// valid mysql connection string is shown below
		// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
		dsn = fmt.Sprintf("%s:%s@%s(%s:%s)/%s", conf.Username, conf.Password, conf.Protocol, conf.Host, conf.Port, conf.Database)
		dsn = conf.appendParams(conf.Params, dsn)
	case "postgres":
		// valid connection string is shown below
		// postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]
		dsn = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", conf.Username, conf.Password, conf.Host, conf.Port, conf.Database)
		dsn = conf.appendParams(conf.Params, dsn)
	default:
		// For sqlite3, this dsn will create an in-memory database for testing
		// For other databases, it will incur an error
		dsn = ""
	}
	return dsn
}

func (conf *DBConf) appendParams(params map[string]string, dataSourceName string) string {
	if len(params) == 0 {
		return dataSourceName
	}
	dataSourceName += "?"
	for k, v := range params {
		dataSourceName += fmt.Sprintf("%s=%s&", k, v)
	}
	dataSourceName = strings.TrimSuffix(dataSourceName, "&")
	return dataSourceName
}
