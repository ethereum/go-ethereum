package history

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/mattn/go-sqlite3"
)

const (
	sqliteName              = "history.sqlite"
	contentDeletionFraction = 0.05 // 5% of the content will be deleted when the storage capacity is hit and radius gets adjusted.
	// SQLite Statements
	createSql = `CREATE TABLE IF NOT EXISTS kvstore (
		key BLOB PRIMARY KEY,
		value BLOB
	);`
	getSql                     = "SELECT value FROM kvstore WHERE key = (?1);"
	putSql                     = "INSERT OR REPLACE INTO kvstore (key, value) VALUES (?1, ?2);"
	deleteSql                  = "DELETE FROM kvstore WHERE key = (?1);"
	containSql                 = "SELECT 1 FROM kvstore WHERE key = (?1);"
	getAllOrderedByDistanceSql = "SELECT key, length(value), xor(key, (?1)) as distance FROM kvstore ORDER BY distance DESC;"
	getFarthestDistanceSql     = "SELECT key, xor(key, (?1)) as distance FROM kvstore ORDER BY distance DESC Limit 1;"
	deleteOutOfRadiusStmt      = "DELETE FROM kvstore WHERE greater(xor(key, (?1)), (?2)) = 1"
	XorFindFarthestQuery       = `SELECT
		xor(key, (?1)) as distance
		FROM kvstore
		ORDER BY distance DESC`
)

var _ storage.ContentStorage = &ContentStorage{}
var once sync.Once

type ContentStorage struct {
	nodeId                 enode.ID
	storageCapacityInBytes uint64
	radius                 atomic.Value
	sqliteDB               *sql.DB
	getStmt                *sql.Stmt
	putStmt                *sql.Stmt
	delStmt                *sql.Stmt
	containStmt            *sql.Stmt
	log                    log.Logger
}

var portalStorageMetrics *portalwire.PortalStorageMetrics

func xor(contentId, nodeId []byte) []byte {
	// length of contentId maybe not 32bytes
	padding := make([]byte, 32)
	if len(contentId) != len(nodeId) {
		copy(padding, contentId)
	} else {
		padding = contentId
	}
	res := make([]byte, len(padding))
	for i := range padding {
		res[i] = padding[i] ^ nodeId[i]
	}
	return res
}

// a > b return 1; a = b return 0; else return -1
func greater(a, b []byte) int {
	return bytes.Compare(a, b)
}

func NewDB(dataDir string, network string) (*sql.DB, error) {
	dbPath := path.Join(dataDir, network)
	err := os.MkdirAll(dbPath, 0755)
	if err != nil {
		return nil, err
	}
	// avoid repeated register in tests
	once.Do(func() {
		sql.Register("sqlite3_custom", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				if err := conn.RegisterFunc("xor", xor, false); err != nil {
					return err
				}
				if err := conn.RegisterFunc("greater", greater, false); err != nil {
					return err
				}
				return nil
			},
		})
	})
	sqlDb, err := sql.Open("sqlite3_custom", path.Join(dbPath, fmt.Sprintf("%s.sqlite", network)))
	return sqlDb, err
}

func NewHistoryStorage(config storage.PortalStorageConfig) (storage.ContentStorage, error) {
	hs := &ContentStorage{
		nodeId:                 config.NodeId,
		sqliteDB:               config.DB,
		storageCapacityInBytes: config.StorageCapacityMB * 1000000,
		log:                    log.New("storage", config.NetworkName),
	}
	hs.radius.Store(storage.MaxDistance)

	err := hs.createTable()
	if err != nil {
		return nil, err
	}

	err = hs.initStmts()
	// Check whether we already have data, and use it to set radius
	hs.setRadiusToFarthestDistance()

	// necessary to test NetworkName==history because state also initialize HistoryStorage
	if strings.ToLower(config.NetworkName) == "history" {
		portalStorageMetrics, err = portalwire.NewPortalStorageMetrics(config.NetworkName, config.DB)
		if err != nil {
			return nil, err
		}
	}

	return hs, err
}

// Get the content according to the contentId
func (p *ContentStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	p.log.Trace("get content", "contentKey", hexutil.Encode(contentKey), "contentId", hexutil.Encode(contentId))
	var res []byte
	err := p.getStmt.QueryRow(contentId).Scan(&res)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrContentNotFound
	}
	return res, err
}

type PutResult struct {
	err    error
	pruned bool
	count  int
}

func (p *PutResult) Err() error {
	return p.err
}

func (p *PutResult) Pruned() bool {
	return p.pruned
}

func (p *PutResult) PrunedCount() int {
	return p.count
}

func newPutResultWithErr(err error) PutResult {
	return PutResult{
		err: err,
	}
}

func (p *ContentStorage) Radius() *uint256.Int {
	radius := p.radius.Load()
	val := radius.(*uint256.Int)
	return val
}

func (p *ContentStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	res := p.put(contentId, content)
	return res.Err()
}

// Put saves the contentId and content
func (p *ContentStorage) put(contentId []byte, content []byte) PutResult {
	_, err := p.putStmt.Exec(contentId, content)
	if err != nil {
		return newPutResultWithErr(err)
	}

	dbSize, err := p.UsedSize()
	if err != nil {
		return newPutResultWithErr(err)
	}
	if dbSize > p.storageCapacityInBytes {
		count, err := p.deleteContentFraction(contentDeletionFraction)
		//
		if err != nil {
			log.Warn("failed to delete oversize item")
			return newPutResultWithErr(err)
		}
		return PutResult{pruned: true, count: count}
	}

	if metrics.Enabled {
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(content)))
	}
	return PutResult{}
}

func (p *ContentStorage) Close() error {
	err := p.getStmt.Close()
	if err != nil {
		return err
	}
	err = p.putStmt.Close()
	if err != nil {
		return err
	}
	err = p.delStmt.Close()
	if err != nil {
		return err
	}
	err = p.containStmt.Close()
	if err != nil {
		return err
	}
	return p.sqliteDB.Close()
}

func (p *ContentStorage) createTable() error {
	stmt, err := p.sqliteDB.Prepare(createSql)
	if err != nil {
		return err
	}
	defer func(stat *sql.Stmt) {
		if err = stat.Close(); err != nil {
			p.log.Error("failed to close statement", "err", err)
		}
	}(stmt)
	_, err = stmt.Exec()
	return err
}

func (p *ContentStorage) initStmts() error {
	var stat *sql.Stmt
	var err error
	if stat, err = p.sqliteDB.Prepare(getSql); err != nil {
		return nil
	}
	p.getStmt = stat
	if stat, err = p.sqliteDB.Prepare(putSql); err != nil {
		return nil
	}
	p.putStmt = stat
	if stat, err = p.sqliteDB.Prepare(deleteSql); err != nil {
		return nil
	}
	p.delStmt = stat
	if stat, err = p.sqliteDB.Prepare(containSql); err != nil {
		return nil
	}
	p.containStmt = stat
	return nil
}

// Size get database size, content size and similar
func (p *ContentStorage) Size() (uint64, error) {
	sql := "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size();"
	return p.queryRowUint64(sql)
}

func (p *ContentStorage) UnusedSize() (uint64, error) {
	sql := "SELECT freelist_count * page_size as size FROM pragma_freelist_count(), pragma_page_size();"
	return p.queryRowUint64(sql)
}

// UsedSize = Size - UnusedSize
func (p *ContentStorage) UsedSize() (uint64, error) {
	size, err := p.Size()
	if err != nil {
		return 0, err
	}
	unusedSize, err := p.UnusedSize()
	if err != nil {
		return 0, err
	}
	return size - unusedSize, err
}

// ContentCount return the total content count
func (p *ContentStorage) ContentCount() (uint64, error) {
	sql := "SELECT COUNT(key) FROM kvstore;"
	return p.queryRowUint64(sql)
}

func (p *ContentStorage) ContentSize() (uint64, error) {
	sql := "SELECT SUM(length(value)) FROM kvstore"
	return p.queryRowUint64(sql)
}

func (p *ContentStorage) SizeByKey(contentId []byte) (uint64, error) {
	sql := "SELECT SUM( length(value) ) FROM kvstore WHERE key = " + string(contentId) + ";"
	return p.queryRowUint64(sql)
}

func (p *ContentStorage) SizeByKeys(ids [][]byte) (uint64, error) {
	sql := "SELECT SUM( length(value) ) FROM kvstore WHERE key IN (?" + strings.Repeat(", ?", len(ids)-1) + ");"
	return p.queryRowUint64(sql)
}

func (p *ContentStorage) SizeOutRadius(radius *uint256.Int) (uint64, error) {
	sql := "SELECT SUM( length(value) ) FROM kvstore WHERE greater(xor(key, (?1)), (?2)) = 1;"
	var size uint64
	err := p.sqliteDB.QueryRow(sql, p.nodeId[:], radius.Bytes()).Scan(&size)
	return size, err
}

func (p *ContentStorage) queryRowUint64(sqlStr string) (uint64, error) {
	// sql := "SELECT SUM(length(value)) FROM kvstore"
	stmt, err := p.sqliteDB.Prepare(sqlStr)
	if err != nil {
		return 0, err
	}
	defer func(stat *sql.Stmt) {
		if err = stat.Close(); err != nil {
			p.log.Error("failed to close statement", "err", err)
		}
	}(stmt)
	var res uint64
	err = stmt.QueryRow().Scan(&res)
	return res, err
}

// GetLargestDistance find the largest distance
func (p *ContentStorage) GetLargestDistance() (*uint256.Int, error) {
	stmt, err := p.sqliteDB.Prepare(XorFindFarthestQuery)
	if err != nil {
		return nil, err
	}
	defer func(stat *sql.Stmt) {
		if err = stat.Close(); err != nil {
			p.log.Error("failed to close statement", "err", err)
		}
	}(stmt)
	var distance []byte

	err = stmt.QueryRow(p.nodeId[:]).Scan(&distance)
	if err != nil {
		return nil, err
	}
	res := uint256.NewInt(0)
	err = res.UnmarshalSSZ(distance)

	return res, err
}

// EstimateNewRadius calculates an estimated new radius based on the current radius, used size, and storage capacity.
// The method takes the currentRadius as input and returns the estimated new radius and an error (if any).
// It calculates the size ratio of usedSize to storageCapacityInBytes and adjusts the currentRadius accordingly.
// If the size ratio is greater than 0, it performs the adjustment; otherwise, it returns the currentRadius unchanged.
// The method returns an error if there is any issue in determining the used size.
func (p *ContentStorage) EstimateNewRadius(currentRadius *uint256.Int) (*uint256.Int, error) {
	currrentSize, err := p.UsedSize()
	if err != nil {
		return nil, err
	}
	sizeRatio := currrentSize / p.storageCapacityInBytes
	if sizeRatio > 0 {
		bigFormat := new(big.Int).SetUint64(sizeRatio)
		if metrics.Enabled {
			newRadius := new(uint256.Int).Div(currentRadius, uint256.MustFromBig(bigFormat))
			newRadius.Mul(newRadius, uint256.NewInt(100))
			newRadius.Mod(newRadius, storage.MaxDistance)
			portalStorageMetrics.RadiusRatio.Update(newRadius.Float64() / 100)
		}
		return new(uint256.Int).Div(currentRadius, uint256.MustFromBig(bigFormat)), nil
	}
	return currentRadius, nil
}

func (p *ContentStorage) setRadiusToFarthestDistance() {
	rows, err := p.sqliteDB.Query(getFarthestDistanceSql, p.nodeId[:])
	if err != nil {
		p.log.Error("failed to query farthest distance ", "err", err)
		return
	}
	defer func(rows *sql.Rows) {
		if rows != nil {
			return
		}
		err = rows.Close()
		if err != nil {
			p.log.Error("failed to close rows", "err", err)
		}
	}(rows)

	if rows.Next() {
		var contentId []byte
		var distance []byte
		err = rows.Scan(&contentId, &distance)
		if err != nil {
			p.log.Error("failed to scan rows for farthest distance", "err", err)
		}
		dis := uint256.NewInt(0)
		err = dis.UnmarshalSSZ(distance)
		if err != nil {
			p.log.Error("failed to unmarshal ssz for farthest distance", "err", err)
		}
		p.radius.Store(dis)
	}
}

func (p *ContentStorage) deleteContentFraction(fraction float64) (deleteCount int, err error) {
	if fraction <= 0 || fraction >= 1 {
		return deleteCount, errors.New("fraction should be between 0 and 1")
	}
	totalContentSize, err := p.ContentSize()
	if err != nil {
		return deleteCount, err
	}
	bytesToDelete := uint64(fraction * float64(totalContentSize))
	// deleteElements := 0
	deleteBytes := 0

	rows, err := p.sqliteDB.Query(getAllOrderedByDistanceSql, p.nodeId[:])
	if err != nil {
		return deleteCount, err
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			p.log.Error("failed to close rows", "err", err)
		}
	}(rows)
	idsToDelete := make([][]byte, 0)
	for deleteBytes < int(bytesToDelete) && rows.Next() {
		var contentId []byte
		var payloadLen int
		var distance []byte
		err = rows.Scan(&contentId, &payloadLen, &distance)
		if err != nil {
			return deleteCount, err
		}
		idsToDelete = append(idsToDelete, contentId)
		// err = p.del(contentId)
		if err != nil {
			return deleteCount, err
		}
		deleteBytes += payloadLen
		deleteCount++
	}
	// set the largest distince
	if rows.Next() {
		var contentId []byte
		var payloadLen int
		var distance []byte
		err = rows.Scan(&contentId, &payloadLen, &distance)
		if err != nil {
			return 0, err
		}
		dis := uint256.NewInt(0)
		err = dis.UnmarshalSSZ(distance)
		if err != nil {
			return 0, err
		}
		p.radius.Store(dis)
		if metrics.Enabled {
			dis.Mul(dis, uint256.NewInt(100))
			dis.Mod(dis, storage.MaxDistance)
			portalStorageMetrics.RadiusRatio.Update(dis.Float64() / 100)
		}
	}
	// row must close first, or database is locked
	// rows.Close() can call multi times
	err = rows.Close()
	if err != nil {
		return 0, err
	}
	err = p.batchDel(idsToDelete)
	return
}

func (p *ContentStorage) del(contentId []byte) error {
	var sizeDel uint64
	var err error
	if metrics.Enabled {
		sizeDel, err = p.SizeByKey(contentId)
		if err != nil {
			return err
		}
	}
	_, err = p.delStmt.Exec(contentId)
	if metrics.Enabled && err == nil {
		portalStorageMetrics.EntriesCount.Dec(1)
		portalStorageMetrics.ContentStorageUsage.Dec(int64(sizeDel))
	}
	return err
}

func (p *ContentStorage) batchDel(ids [][]byte) error {
	var sizeDel uint64
	var err error
	if metrics.Enabled {
		sizeDel, err = p.SizeByKeys(ids)
		if err != nil {
			return err
		}
	}
	query := "DELETE FROM kvstore WHERE key IN (?" + strings.Repeat(", ?", len(ids)-1) + ")"
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	// delete items
	_, err = p.sqliteDB.Exec(query, args...)
	if metrics.Enabled && err == nil {
		portalStorageMetrics.EntriesCount.Dec(int64(len(args)))
		portalStorageMetrics.ContentStorageUsage.Dec(int64(sizeDel))
	}
	return err
}

// ReclaimSpace reclaims space in the ContentStorage's SQLite database by performing a VACUUM operation.
// It returns an error if the VACUUM operation encounters any issues.
func (p *ContentStorage) ReclaimSpace() error {
	_, err := p.sqliteDB.Exec("VACUUM;")
	return err
}

func (p *ContentStorage) deleteContentOutOfRadius(radius *uint256.Int) error {
	var sizeDel uint64
	var err error
	if metrics.Enabled {
		sizeDel, err = p.SizeOutRadius(radius)
		if err != nil {
			return err
		}
	}
	res, err := p.sqliteDB.Exec(deleteOutOfRadiusStmt, p.nodeId[:], radius.Bytes())
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	p.log.Trace("delete items", "count", count)
	if metrics.Enabled && err == nil {
		portalStorageMetrics.EntriesCount.Dec(count)
		portalStorageMetrics.ContentStorageUsage.Dec(int64(sizeDel))
	}
	return err
}

// ForcePrune delete the content which distance is further than the given radius
func (p *ContentStorage) ForcePrune(radius *uint256.Int) error {
	return p.deleteContentOutOfRadius(radius)
}
