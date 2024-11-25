package beacon

import (
	"bytes"
	"context"
	"database/sql"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"

	"github.com/ethereum/go-ethereum/portalnetwork/storage"
)

const BytesInMB uint64 = 1000 * 1000

type Storage struct {
	storageCapacityInBytes uint64
	db                     *sql.DB
	log                    log.Logger
	spec                   *common.Spec
	cache                  *beaconStorageCache
}

var portalStorageMetrics *portalwire.PortalStorageMetrics

type beaconStorageCache struct {
	OptimisticUpdate []byte
	FinalityUpdate   []byte
}

var _ storage.ContentStorage = &Storage{}

func NewBeaconStorage(config storage.PortalStorageConfig) (storage.ContentStorage, error) {
	bs := &Storage{
		storageCapacityInBytes: config.StorageCapacityMB * BytesInMB,
		db:                     config.DB,
		log:                    log.New("beacon_storage"),
		spec:                   config.Spec,
		cache:                  &beaconStorageCache{},
	}
	if err := bs.setup(); err != nil {
		return nil, err
	}

	var err error
	portalStorageMetrics, err = portalwire.NewPortalStorageMetrics(config.NetworkName, config.DB)
	if err != nil {
		return nil, err
	}

	return bs, nil
}

func (bs *Storage) setup() error {
	if _, err := bs.db.Exec(CreateQueryDBBeacon); err != nil {
		return err
	}
	if _, err := bs.db.Exec(LCUpdateCreateTable); err != nil {
		return err
	}
	return nil
}

func (bs *Storage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	switch storage.ContentType(contentKey[0]) {
	case LightClientBootstrap:
		return bs.getContentValue(contentId)
	case LightClientUpdate:
		lightClientUpdateKey := new(LightClientUpdateKey)
		err := lightClientUpdateKey.UnmarshalSSZ(contentKey[1:])
		if err != nil {
			return nil, err
		}
		return bs.getLcUpdateValueByRange(lightClientUpdateKey.StartPeriod, lightClientUpdateKey.StartPeriod+lightClientUpdateKey.Count)
	case LightClientFinalityUpdate:
		if bs.cache.FinalityUpdate == nil {
			return nil, storage.ErrContentNotFound
		}
		return bs.cache.FinalityUpdate, nil
	case LightClientOptimisticUpdate:
		if bs.cache.OptimisticUpdate == nil {
			return nil, storage.ErrContentNotFound
		}
		return bs.cache.OptimisticUpdate, nil
	}
	return nil, nil
}

func (bs *Storage) Put(contentKey []byte, contentId []byte, content []byte) error {
	switch storage.ContentType(contentKey[0]) {
	case LightClientBootstrap:
		return bs.putContentValue(contentId, contentKey, content)
	case LightClientUpdate:
		lightClientUpdateKey := new(LightClientUpdateKey)
		err := lightClientUpdateKey.UnmarshalSSZ(contentKey[1:])
		if err != nil {
			return err
		}
		lightClientUpdateRange := new(LightClientUpdateRange)
		reader := codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content)))
		err = lightClientUpdateRange.Deserialize(bs.spec, reader)
		if err != nil {
			return err
		}
		for index, update := range *lightClientUpdateRange {
			var buf bytes.Buffer
			writer := codec.NewEncodingWriter(&buf)
			err := update.Serialize(bs.spec, writer)
			if err != nil {
				return err
			}
			period := lightClientUpdateKey.StartPeriod + uint64(index)
			err = bs.putLcUpdate(period, buf.Bytes())
			if err != nil {
				return err
			}
		}
		return nil
	case LightClientFinalityUpdate:
		bs.cache.FinalityUpdate = content
		return nil
	case LightClientOptimisticUpdate:
		bs.cache.OptimisticUpdate = content
		return nil
	}
	return nil
}

func (bs *Storage) Radius() *uint256.Int {
	return storage.MaxDistance
}

func (bs *Storage) getContentValue(contentId []byte) ([]byte, error) {
	res := make([]byte, 0)
	err := bs.db.QueryRowContext(context.Background(), ContentValueLookupQueryBeacon, contentId).Scan(&res)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrContentNotFound
	}
	return res, err
}

func (bs *Storage) getLcUpdateValueByRange(start, end uint64) ([]byte, error) {
	// LightClientUpdateRange := make([]ForkedLightClientUpdate, 0)
	var lightClientUpdateRange LightClientUpdateRange
	rows, err := bs.db.QueryContext(context.Background(), LCUpdateLookupQueryByRange, start, end)
	if err != nil {
		return nil, err
	}
	hasData := false
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {
			bs.log.Error("failed to close rows", "err", err)
		}
	}(rows)
	for rows.Next() {
		hasData = true
		var val []byte
		err = rows.Scan(&val)
		if err != nil {
			return nil, err
		}
		update := new(ForkedLightClientUpdate)
		dec := codec.NewDecodingReader(bytes.NewReader(val), uint64(len(val)))
		err = update.Deserialize(bs.spec, dec)
		if err != nil {
			return nil, err
		}
		lightClientUpdateRange = append(lightClientUpdateRange, *update)
	}
	if !hasData {
		return nil, storage.ErrContentNotFound
	}
	var buf bytes.Buffer
	err = lightClientUpdateRange.Serialize(bs.spec, codec.NewEncodingWriter(&buf))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (bs *Storage) putContentValue(contentId, contentKey, value []byte) error {
	length := 32 + len(contentKey) + len(value)
	_, err := bs.db.ExecContext(context.Background(), InsertQueryBeacon, contentId, contentKey, value, length)
	if metrics.Enabled && err == nil {
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(value)))
	}
	return err
}

func (bs *Storage) putLcUpdate(period uint64, value []byte) error {
	_, err := bs.db.ExecContext(context.Background(), InsertLCUpdateQuery, period, value, 0, len(value))
	if metrics.Enabled && err == nil {
		portalStorageMetrics.EntriesCount.Inc(1)
		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(value)))
	}
	return err
}
