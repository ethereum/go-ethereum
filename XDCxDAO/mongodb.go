package XDCxDAO

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	lru "github.com/hashicorp/golang-lru"
	"strings"
	"time"
)

const (
	ordersCollection        = "orders"
	tradesCollection        = "trades"
	lendingItemsCollection  = "lending_items"
	lendingTradesCollection = "lending_trades"
	lendingTopUpCollection  = "lending_topups"
	lendingRepayCollection  = "lending_repays"
	lendingRecallCollection = "lending_recalls"
	epochPriceCollection    = "epoch_prices"
)

type MongoDatabase struct {
	Session          *mgo.Session
	dbName           string
	emptyKey         []byte
	cacheItems       *lru.Cache // Cache for reading
	orderBulk        *mgo.Bulk
	tradeBulk        *mgo.Bulk
	epochPriceBulk   *mgo.Bulk
	lendingItemBulk  *mgo.Bulk
	topUpBulk        *mgo.Bulk
	recallBulk       *mgo.Bulk
	repayBulk        *mgo.Bulk
	lendingTradeBulk *mgo.Bulk
}

// InitSession initializes a new session with mongodb
func NewMongoDatabase(session *mgo.Session, dbName string, mongoURL string, replicaSetName string, cacheLimit int) (*MongoDatabase, error) {
	if session == nil {
		// in case of multiple database instances
		hosts := strings.Split(mongoURL, ",")
		dbInfo := &mgo.DialInfo{
			Addrs:          hosts,
			Database:       dbName,
			ReplicaSetName: replicaSetName,
			Timeout:        30 * time.Second,
		}
		ns, err := mgo.DialWithInfo(dbInfo)
		if err != nil {
			return nil, err
		}
		session = ns
	}
	itemCacheLimit := defaultCacheLimit
	if cacheLimit > 0 {
		itemCacheLimit = cacheLimit
	}
	cacheItems, _ := lru.New(itemCacheLimit)

	db := &MongoDatabase{
		Session:    session,
		dbName:     dbName,
		cacheItems: cacheItems,
	}
	if err := db.EnsureIndexes(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *MongoDatabase) IsEmptyKey(key []byte) bool {
	return key == nil || len(key) == 0 || bytes.Equal(key, db.emptyKey)
}

func (db *MongoDatabase) getCacheKey(key []byte) string {
	return hex.EncodeToString(key)
}

func (db *MongoDatabase) HasObject(hash common.Hash, val interface{}) (bool, error) {
	if db.IsEmptyKey(hash.Bytes()) {
		return false, nil
	}
	cacheKey := db.getCacheKey(hash.Bytes())
	if db.cacheItems.Contains(cacheKey) {
		return true, nil
	}

	sc := db.Session.Copy()
	defer sc.Close()
	var (
		count int
		err   error
	)
	query := bson.M{"hash": hash.Hex()}
	switch val.(type) {
	case *tradingstate.OrderItem:
		// Find key in ordersCollection collection
		count, err = sc.DB(db.dbName).C(ordersCollection).Find(query).Limit(1).Count()

		if err != nil {
			return false, err
		}

		if count == 1 {
			return true, nil
		}
	case *tradingstate.Trade:
		// Find key in tradesCollection collection
		count, err = sc.DB(db.dbName).C(tradesCollection).Find(query).Limit(1).Count()

		if err != nil {
			return false, err
		}

		if count == 1 {
			return true, nil
		}
	case *lendingstate.LendingItem:
		// Find key in lendingItemsCollection collection
		item := val.(*lendingstate.LendingItem)
		switch item.Type {
		case lendingstate.Repay:
			count, err = sc.DB(db.dbName).C(lendingRepayCollection).Find(query).Limit(1).Count()
		case lendingstate.TopUp:
			count, err = sc.DB(db.dbName).C(lendingTopUpCollection).Find(query).Limit(1).Count()
		case lendingstate.Recall:
			count, err = sc.DB(db.dbName).C(lendingRecallCollection).Find(query).Limit(1).Count()
		default:
			count, err = sc.DB(db.dbName).C(lendingItemsCollection).Find(query).Limit(1).Count()
		}

		if err != nil {
			return false, err
		}

		if count == 1 {
			return true, nil
		}
	case *lendingstate.LendingTrade:
		// Find key in lendingTradesCollection collection
		count, err = sc.DB(db.dbName).C(lendingTradesCollection).Find(query).Limit(1).Count()

		if err != nil {
			return false, err
		}

		if count == 1 {
			return true, nil
		}

	}
	return false, nil
}

func (db *MongoDatabase) GetObject(hash common.Hash, val interface{}) (interface{}, error) {

	if db.IsEmptyKey(hash.Bytes()) {
		return nil, nil
	}

	cacheKey := db.getCacheKey(hash.Bytes())
	if cached, ok := db.cacheItems.Get(cacheKey); ok {
		return cached, nil
	} else {
		sc := db.Session.Copy()
		defer sc.Close()

		query := bson.M{"hash": hash.Hex()}

		switch val.(type) {
		case *tradingstate.OrderItem:
			var oi *tradingstate.OrderItem
			err := sc.DB(db.dbName).C(ordersCollection).Find(query).One(&oi)
			if err != nil {
				return nil, err
			}
			db.cacheItems.Add(cacheKey, oi)
			return oi, nil
		case *tradingstate.Trade:
			var t *tradingstate.Trade
			err := sc.DB(db.dbName).C(tradesCollection).Find(query).One(&t)
			if err != nil {
				return nil, err
			}
			db.cacheItems.Add(cacheKey, t)
			return t, nil
		case *lendingstate.LendingItem:
			var li *lendingstate.LendingItem
			var err error
			item := val.(*lendingstate.LendingItem)
			switch item.Type {
			case lendingstate.Repay:
				err = sc.DB(db.dbName).C(lendingRepayCollection).Find(query).One(&li)
			case lendingstate.TopUp:
				err = sc.DB(db.dbName).C(lendingTopUpCollection).Find(query).One(&li)
			case lendingstate.Recall:
				err = sc.DB(db.dbName).C(lendingRecallCollection).Find(query).One(&li)
			default:
				err = sc.DB(db.dbName).C(lendingItemsCollection).Find(query).One(&li)
			}
			if err != nil {
				return nil, err
			}
			db.cacheItems.Add(cacheKey, li)
			return li, nil
		case *lendingstate.LendingTrade:
			var t *lendingstate.LendingTrade
			err := sc.DB(db.dbName).C(lendingTradesCollection).Find(query).One(&t)
			if err != nil {
				return nil, err
			}
			db.cacheItems.Add(cacheKey, t)
			return t, nil
		default:
			return nil, nil
		}
	}
}

func (db *MongoDatabase) PutObject(hash common.Hash, val interface{}) error {
	cacheKey := db.getCacheKey(hash.Bytes())
	db.cacheItems.Add(cacheKey, val)

	switch val.(type) {
	case *tradingstate.Trade:
		// PutObject trade into tradesCollection collection
		db.tradeBulk.Insert(val.(*tradingstate.Trade))
	case *tradingstate.OrderItem:
		// PutObject order into ordersCollection collection
		o := val.(*tradingstate.OrderItem)
		if o.Status == tradingstate.OrderStatusOpen {
			db.orderBulk.Insert(o)
		} else {
			query := bson.M{"hash": o.Hash.Hex()}
			db.orderBulk.Upsert(query, o)
		}
		return nil
	case *tradingstate.EpochPriceItem:
		item := val.(*tradingstate.EpochPriceItem)
		query := bson.M{"hash": item.Hash.Hex()}
		db.epochPriceBulk.Upsert(query, item)
		return nil
	case *lendingstate.LendingTrade:
		lt := val.(*lendingstate.LendingTrade)
		// PutObject LendingTrade into tradesCollection collection
		if existed, err := db.HasObject(hash, val); err == nil && existed {
			query := bson.M{"hash": lt.Hash.Hex()}
			db.lendingTradeBulk.Upsert(query, lt)
		} else {
			db.lendingTradeBulk.Insert(lt)
		}
	case *lendingstate.LendingItem:
		// PutObject order into ordersCollection collection
		li := val.(*lendingstate.LendingItem)
		switch li.Type {
		case lendingstate.Repay:
			if li.Status != lendingstate.LendingStatusReject {
				li.Status = lendingstate.Repay
			}
			db.repayBulk.Insert(li)
			return nil
		case lendingstate.TopUp:
			if li.Status != lendingstate.LendingStatusReject {
				li.Status = lendingstate.TopUp
			}
			db.topUpBulk.Insert(li)
			return nil
		case lendingstate.Recall:
			if li.Status != lendingstate.LendingStatusReject {
				li.Status = lendingstate.Recall
			}
			db.recallBulk.Insert(li)
			return nil
		default:
			if li.Status == lendingstate.LendingStatusOpen {
				db.lendingItemBulk.Insert(li)
			} else {
				query := bson.M{"hash": li.Hash.Hex()}
				db.lendingItemBulk.Upsert(query, li)
			}
			return nil
		}

	default:
		log.Error("PutObject: unknown type of object", "val", val)
	}

	return nil
}

func (db *MongoDatabase) DeleteObject(hash common.Hash, val interface{}) error {
	cacheKey := db.getCacheKey(hash.Bytes())
	db.cacheItems.Remove(cacheKey)

	sc := db.Session.Copy()
	defer sc.Close()

	query := bson.M{"hash": hash.Hex()}

	found, err := db.HasObject(hash, val)
	if err != nil {
		return err
	}

	if found {
		var err error
		switch val.(type) {
		case *tradingstate.OrderItem:
			err = sc.DB(db.dbName).C(ordersCollection).Remove(query)
			if err != nil && err != mgo.ErrNotFound {
				return fmt.Errorf("failed to delete orderItem. Err: %v", err)
			}
		case *tradingstate.Trade:
			err = sc.DB(db.dbName).C(tradesCollection).Remove(query)
			if err != nil && err != mgo.ErrNotFound {
				return fmt.Errorf("failed to delete XDCx trade. Err: %v", err)
			}
		case *lendingstate.LendingItem:
			item := val.(*lendingstate.LendingItem)
			switch item.Type {
			case lendingstate.Repay:
				err = sc.DB(db.dbName).C(lendingRepayCollection).Remove(query)
			case lendingstate.TopUp:
				err = sc.DB(db.dbName).C(lendingTopUpCollection).Remove(query)
			case lendingstate.Recall:
				err = sc.DB(db.dbName).C(lendingRecallCollection).Remove(query)
			default:
				err = sc.DB(db.dbName).C(lendingItemsCollection).Remove(query)
			}
			if err != nil && err != mgo.ErrNotFound {
				return fmt.Errorf("failed to delete lendingItem. Err: %v", err)
			}
		case *lendingstate.LendingTrade:
			err = sc.DB(db.dbName).C(lendingTradesCollection).Remove(query)
			if err != nil && err != mgo.ErrNotFound {
				return fmt.Errorf("failed to delete lendingTrade. Err: %v", err)
			}

		}
	}

	return nil
}

func (db *MongoDatabase) InitBulk() {
	sc := db.Session
	db.orderBulk = sc.DB(db.dbName).C(ordersCollection).Bulk()
	db.tradeBulk = sc.DB(db.dbName).C(tradesCollection).Bulk()
	db.epochPriceBulk = sc.DB(db.dbName).C(epochPriceCollection).Bulk()
}

func (db *MongoDatabase) InitLendingBulk() {
	sc := db.Session
	db.lendingItemBulk = sc.DB(db.dbName).C(lendingItemsCollection).Bulk()
	db.lendingTradeBulk = sc.DB(db.dbName).C(lendingTradesCollection).Bulk()
	db.topUpBulk = sc.DB(db.dbName).C(lendingTopUpCollection).Bulk()
	db.repayBulk = sc.DB(db.dbName).C(lendingRepayCollection).Bulk()
	db.recallBulk = sc.DB(db.dbName).C(lendingRecallCollection).Bulk()
}

func (db *MongoDatabase) CommitBulk() error {
	if _, err := db.orderBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.tradeBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.epochPriceBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	return nil
}

func (db *MongoDatabase) CommitLendingBulk() error {
	if _, err := db.lendingItemBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.lendingTradeBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.topUpBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.repayBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	if _, err := db.recallBulk.Run(); err != nil && !mgo.IsDup(err) {
		return err
	}
	return nil
}

func (db *MongoDatabase) Put(key []byte, val []byte) error {
	// for levelDB only
	return nil
}

func (db *MongoDatabase) Delete(key []byte) error {
	// for levelDB only
	return nil
}

func (db *MongoDatabase) Has(key []byte) (bool, error) {
	// for levelDB only
	return false, nil
}

func (db *MongoDatabase) Get(key []byte) ([]byte, error) {
	// for levelDB only
	return nil, nil
}

func (db *MongoDatabase) DeleteItemByTxHash(txhash common.Hash, val interface{}) {
	sc := db.Session.Copy()
	defer sc.Close()

	query := bson.M{"txHash": txhash.Hex()}
	switch val.(type) {
	case *tradingstate.OrderItem:
		if err := sc.DB(db.dbName).C(ordersCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
			log.Error("DeleteItemByTxHash: failed to delete order", "txhash", txhash, "err", err)
		}
	case *tradingstate.Trade:
		if err := sc.DB(db.dbName).C(tradesCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
			log.Error("DeleteItemByTxHash: failed to delete trade", "txhash", txhash, "err", err)
		}
	case *lendingstate.LendingItem:
		item := val.(*lendingstate.LendingItem)
		switch item.Type {
		case lendingstate.Repay:
			if err := sc.DB(db.dbName).C(lendingRepayCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
				log.Error("DeleteItemByTxHash: failed to delete repayItem", "txhash", txhash, "err", err)
			}
			return
		case lendingstate.TopUp:
			if err := sc.DB(db.dbName).C(lendingTopUpCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
				log.Error("DeleteItemByTxHash: failed to delete topupItem", "txhash", txhash, "err", err)
			}
			return
		case lendingstate.Recall:
			if err := sc.DB(db.dbName).C(lendingRecallCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
				log.Error("DeleteItemByTxHash: failed to delete recallItem", "txhash", txhash, "err", err)
			}
			return
		default:
			if err := sc.DB(db.dbName).C(lendingItemsCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
				log.Error("DeleteItemByTxHash: failed to delete lendingItem", "txhash", txhash, "err", err)
			}
			return
		}

	case *lendingstate.LendingTrade:
		if err := sc.DB(db.dbName).C(lendingTradesCollection).Remove(query); err != nil && err != mgo.ErrNotFound {
			log.Error("DeleteItemByTxHash: failed to delete lendingTrade", "txhash", txhash, "err", err)
		}
	default:
		log.Error("DeleteItemByTxHash: Unknown object type", "txhash", txhash, "object", val)
	}

}

func (db *MongoDatabase) GetListItemByTxHash(txhash common.Hash, val interface{}) interface{} {
	sc := db.Session.Copy()
	defer sc.Close()

	query := bson.M{"txHash": txhash.Hex()}
	switch val.(type) {
	case *tradingstate.OrderItem:
		result := []*tradingstate.OrderItem{}
		if err := sc.DB(db.dbName).C(ordersCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByTxHash (orders)", "err", err, "Txhash", txhash)
		}
		return result
	case *tradingstate.Trade:
		result := []*tradingstate.Trade{}
		if err := sc.DB(db.dbName).C(tradesCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByTxHash (trades)", "err", err, "Txhash", txhash)
		}
		return result
	case *lendingstate.LendingItem:
		item := val.(*lendingstate.LendingItem)
		result := []*lendingstate.LendingItem{}
		switch item.Type {
		case lendingstate.Repay:
			if err := sc.DB(db.dbName).C(lendingRepayCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByTxHash (repayItems)", "err", err, "txhash", txhash)
			}
			return result
		case lendingstate.TopUp:
			if err := sc.DB(db.dbName).C(lendingTopUpCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByTxHash (topupItems)", "err", err, "txhash", txhash)
			}
			return result
		case lendingstate.Recall:
			if err := sc.DB(db.dbName).C(lendingRecallCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByTxHash (recallItems)", "err", err, "txhash", txhash)
			}
			return result
		default:
			if err := sc.DB(db.dbName).C(lendingItemsCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByTxHash (lendingItems)", "err", err, "txhash", txhash)
			}
			return result
		}
	case *lendingstate.LendingTrade:
		result := []*lendingstate.LendingTrade{}
		if err := sc.DB(db.dbName).C(lendingTradesCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByTxHash (lendingTrades)", "err", err, "Txhash", txhash)
		}
		return result
	default:
		log.Error("GetListItemByTxHash: Unknown object type", "txhash", txhash, "object", val)
	}
	return nil
}

func (db *MongoDatabase) GetListItemByHashes(hashes []string, val interface{}) interface{} {
	sc := db.Session.Copy()
	defer sc.Close()

	query := bson.M{"hash": bson.M{"$in": hashes}}

	switch val.(type) {
	case *tradingstate.OrderItem:
		result := []*tradingstate.OrderItem{}
		if err := sc.DB(db.dbName).C(ordersCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByHashes (orders)", "err", err, "hashes", hashes)
		}
		return result
	case *tradingstate.Trade:
		result := []*tradingstate.Trade{}
		if err := sc.DB(db.dbName).C(tradesCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByHashes (trades)", "err", err, "hashes", hashes)
		}
		return result
	case *lendingstate.LendingItem:
		item := val.(*lendingstate.LendingItem)
		result := []*lendingstate.LendingItem{}
		switch item.Type {
		case lendingstate.Repay:
			if err := sc.DB(db.dbName).C(lendingRepayCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByHashes (repayItems)", "err", err, "hashes", hashes)
			}
			return result
		case lendingstate.TopUp:
			if err := sc.DB(db.dbName).C(lendingTopUpCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByHashes (topupItems)", "err", err, "hashes", hashes)
			}
			return result
		case lendingstate.Recall:
			if err := sc.DB(db.dbName).C(lendingRecallCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByHashes (recallItems)", "err", err, "hashes", hashes)
			}
			return result
		default:
			if err := sc.DB(db.dbName).C(lendingItemsCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
				log.Error("failed to GetListItemByHashes (lendingItems)", "err", err, "hashes", hashes)
			}
			return result
		}
	case *lendingstate.LendingTrade:
		result := []*lendingstate.LendingTrade{}
		if err := sc.DB(db.dbName).C(lendingTradesCollection).Find(query).All(&result); err != nil && err != mgo.ErrNotFound {
			log.Error("failed to GetListItemByHashes (lendingTrades)", "err", err, "hashes", hashes)
		}
		return result
	default:
		log.Error("GetListItemByHashes: Unknown object type", "hashes", hashes, "object", val)
	}
	return nil
}

func (db *MongoDatabase) EnsureIndexes() error {
	orderHashIndex := mgo.Index{
		Key:        []string{"hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_order_hash",
	}
	orderTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_order_tx_hash",
	}
	tradeHashIndex := mgo.Index{
		Key:        []string{"hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_trade_hash",
	}
	tradeTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_trade_tx_hash",
	}
	lendingItemHashIndex := mgo.Index{
		Key:        []string{"hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_item_hash",
	}
	lendingItemTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_item_tx_hash",
	}
	lendingTradeHashIndex := mgo.Index{
		Key:        []string{"hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_trade_hash",
	}
	lendingTradeTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_trade_tx_hash",
	}
	repayHashIndex := mgo.Index{
		Key:        []string{"hash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_repay_hash",
	}
	repayTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_repay_tx_hash",
	}

	repayUniqueIndex := mgo.Index{
		Key:        []string{"txHash", "hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_repay_unique",
	}

	recallHashIndex := mgo.Index{
		Key:        []string{"hash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_recall_hash",
	}
	recallTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_recall_tx_hash",
	}

	recallUniqueIndex := mgo.Index{
		Key:        []string{"txHash", "hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_recall_unique",
	}

	topupHashIndex := mgo.Index{
		Key:        []string{"hash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_topup_hash",
	}
	topupTxHashIndex := mgo.Index{
		Key:        []string{"txHash"},
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_topup_tx_hash",
	}

	topUpUniqueIndex := mgo.Index{
		Key:        []string{"txHash", "hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_lending_topup_unique",
	}

	epochPriceIndex := mgo.Index{
		Key:        []string{"hash"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
		Name:       "index_epoch_price",
	}

	sc := db.Session.Copy()
	defer sc.Close()

	indexes, _ := sc.DB(db.dbName).C(ordersCollection).Indexes()
	if !existingIndex(orderHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(ordersCollection).EnsureIndex(orderHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", orderHashIndex.Name, err)
		}
	}
	if !existingIndex(orderTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(ordersCollection).EnsureIndex(orderTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", orderTxHashIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(tradesCollection).Indexes()
	if !existingIndex(tradeHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(tradesCollection).EnsureIndex(tradeHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", tradeHashIndex.Name, err)
		}
	}
	if !existingIndex(tradeTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(tradesCollection).EnsureIndex(tradeTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", tradeTxHashIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(lendingItemsCollection).Indexes()
	if !existingIndex(lendingItemHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingItemsCollection).EnsureIndex(lendingItemHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", lendingItemHashIndex.Name, err)
		}
	}
	if !existingIndex(lendingItemTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingItemsCollection).EnsureIndex(lendingItemTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", lendingItemTxHashIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(lendingTradesCollection).Indexes()
	if !existingIndex(lendingTradeHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingTradesCollection).EnsureIndex(lendingTradeHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", lendingTradeHashIndex.Name, err)
		}
	}
	if !existingIndex(lendingTradeTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingTradesCollection).EnsureIndex(lendingTradeTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", lendingTradeTxHashIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(lendingRepayCollection).Indexes()
	if !existingIndex(repayHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRepayCollection).EnsureIndex(repayHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", repayHashIndex.Name, err)
		}
	}
	if !existingIndex(repayTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRepayCollection).EnsureIndex(repayTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", repayTxHashIndex.Name, err)
		}
	}

	if !existingIndex(repayUniqueIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRepayCollection).EnsureIndex(repayUniqueIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", repayUniqueIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(lendingRecallCollection).Indexes()
	if !existingIndex(recallHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRecallCollection).EnsureIndex(recallHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", recallHashIndex.Name, err)
		}
	}
	if !existingIndex(recallTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRecallCollection).EnsureIndex(recallTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", recallTxHashIndex.Name, err)
		}
	}
	if !existingIndex(recallUniqueIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingRecallCollection).EnsureIndex(repayUniqueIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", repayUniqueIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(lendingTopUpCollection).Indexes()
	if !existingIndex(topupHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingTopUpCollection).EnsureIndex(topupHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", topupHashIndex.Name, err)
		}
	}
	if !existingIndex(topupTxHashIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingTopUpCollection).EnsureIndex(topupTxHashIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", topupTxHashIndex.Name, err)
		}
	}

	if !existingIndex(topUpUniqueIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(lendingTopUpCollection).EnsureIndex(repayUniqueIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", repayUniqueIndex.Name, err)
		}
	}

	indexes, _ = sc.DB(db.dbName).C(epochPriceCollection).Indexes()
	if !existingIndex(epochPriceIndex.Name, indexes) {
		if err := sc.DB(db.dbName).C(epochPriceCollection).EnsureIndex(epochPriceIndex); err != nil {
			return fmt.Errorf("failed to create index %s . Err: %v", epochPriceIndex.Name, err)
		}
	}
	return nil
}

func (db *MongoDatabase) Close() error {
	return db.Close()
}

// HasAncient returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) HasAncient(kind string, number uint64) (bool, error) {
	return false, errNotSupported
}

// Ancient returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// Ancients returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) Ancients() (uint64, error) {
	return 0, errNotSupported
}

// AncientSize returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) AncientSize(kind string) (uint64, error) {
	return 0, errNotSupported
}

// AppendAncient returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) AppendAncient(number uint64, hash, header, body, receipts, td []byte) error {
	return errNotSupported
}

// TruncateAncients returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) TruncateAncients(items uint64) error {
	return errNotSupported
}

// Sync returns an error as we don't have a backing chain freezer.
func (db *MongoDatabase) Sync() error {
	return errNotSupported
}

func (db *MongoDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return db.NewIterator(prefix, start)
}

func (db *MongoDatabase) Stat(property string) (string, error) {
	return db.Stat(property)
}

func (db *MongoDatabase) Compact(start []byte, limit []byte) error {
	return db.Compact(start, limit)
}

func (db *MongoDatabase) NewBatch() ethdb.Batch {
	// for levelDB only
	return nil
}

type keyvalue struct {
	key   []byte
	value []byte
}
type Batch struct {
	db         *MongoDatabase
	collection string
	b          []keyvalue
	size       int
}

func (b *Batch) SetCollection(collection string) {
	// for levelDB only
}

func (b *Batch) Put(key, value []byte) error {
	// for levelDB only
	return nil
}

func (b *Batch) Write() error {
	// for levelDB only
	return nil
}

func (b *Batch) ValueSize() int {
	// for levelDB only
	return int(0)
}
func (b *Batch) Reset() {
	// for levelDB only
}

func existingIndex(indexName string, indexes []mgo.Index) bool {
	if len(indexes) == 0 {
		return false
	}
	for _, index := range indexes {
		if index.Name == indexName {
			return true
		}
	}
	return false
}
