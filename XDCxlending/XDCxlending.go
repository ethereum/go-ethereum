package XDCxlending

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxDAO"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	lru "github.com/hashicorp/golang-lru"
)

const (
	ProtocolName       = "XDCxlending"
	ProtocolVersion    = uint64(1)
	ProtocolVersionStr = "1.0"
	defaultCacheLimit  = 1024
)

var (
	ErrNonceTooHigh = errors.New("nonce too high")
	ErrNonceTooLow  = errors.New("nonce too low")
)

type Lending struct {
	Triegc     *prque.Prque          // Priority queue mapping block numbers to tries to gc
	StateCache lendingstate.Database // State database to reuse between imports (contains state cache)    *lendingstate.TradingStateDB

	orderNonce map[common.Address]*big.Int

	XDCx                *XDCx.XDCX
	lendingItemHistory  *lru.Cache
	lendingTradeHistory *lru.Cache
}

func (l *Lending) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

func (l *Lending) Start(server *p2p.Server) error {
	return nil
}

func (l *Lending) SaveData() {
}

func (l *Lending) Stop() error {
	return nil
}

func New(XDCx *XDCx.XDCX) *Lending {
	itemCache, _ := lru.New(defaultCacheLimit)
	lendingTradeCache, _ := lru.New(defaultCacheLimit)
	lending := &Lending{
		orderNonce:          make(map[common.Address]*big.Int),
		Triegc:              prque.New(),
		lendingItemHistory:  itemCache,
		lendingTradeHistory: lendingTradeCache,
	}
	lending.StateCache = lendingstate.NewDatabase(XDCx.GetLevelDB())
	lending.XDCx = XDCx
	return lending
}

func (l *Lending) GetLevelDB() XDCxDAO.XDCXDAO {
	return l.XDCx.GetLevelDB()
}

func (l *Lending) GetMongoDB() XDCxDAO.XDCXDAO {
	return l.XDCx.GetMongoDB()
}

// APIs returns the RPC descriptors the Lending implementation offers
func (l *Lending) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicXDCXLendingAPI(l),
			Public:    true,
		},
	}
}

// Version returns the Lending sub-protocols version number.
func (l *Lending) Version() uint64 {
	return ProtocolVersion
}

func (l *Lending) ProcessOrderPending(header *types.Header, coinbase common.Address, chain consensus.ChainContext, pending map[common.Address]types.LendingTransactions, statedb *state.StateDB, lendingStatedb *lendingstate.LendingStateDB, tradingStateDb *tradingstate.TradingStateDB) ([]*lendingstate.LendingItem, map[common.Hash]lendingstate.MatchingResult) {
	lendingItems := []*lendingstate.LendingItem{}
	matchingResults := map[common.Hash]lendingstate.MatchingResult{}

	txs := types.NewLendingTransactionByNonce(types.LendingTxSigner{}, pending)
	for {
		tx := txs.Peek()
		if tx == nil {
			break
		}
		log.Debug("ProcessOrderPending start", "len", len(pending))
		log.Debug("Get pending orders to process", "address", tx.UserAddress(), "nonce", tx.Nonce())
		V, R, S := tx.Signature()

		bigstr := V.String()
		n, e := strconv.ParseInt(bigstr, 10, 8)
		if e != nil {
			continue
		}

		order := &lendingstate.LendingItem{
			Nonce:           big.NewInt(int64(tx.Nonce())),
			Quantity:        tx.Quantity(),
			Interest:        new(big.Int).SetUint64(tx.Interest()),
			Relayer:         tx.RelayerAddress(),
			Term:            tx.Term(),
			UserAddress:     tx.UserAddress(),
			LendingToken:    tx.LendingToken(),
			CollateralToken: tx.CollateralToken(),
			AutoTopUp:       tx.AutoTopUp(),
			Status:          tx.Status(),
			Side:            tx.Side(),
			Type:            tx.Type(),
			Hash:            tx.LendingHash(),
			LendingId:       tx.LendingId(),
			LendingTradeId:  tx.LendingTradeId(),
			ExtraData:       tx.ExtraData(),
			Signature: &lendingstate.Signature{
				V: byte(n),
				R: common.BigToHash(R),
				S: common.BigToHash(S),
			},
		}
		cancel := false
		if order.Status == lendingstate.LendingStatusCancelled {
			cancel = true
		}

		log.Info("Process order pending", "orderPending", order, "LendingToken", order.LendingToken.Hex(), "CollateralToken", order.CollateralToken)
		originalOrder := &lendingstate.LendingItem{}
		*originalOrder = *order
		originalOrder.Quantity = lendingstate.CloneBigInt(order.Quantity)

		if cancel {
			order.Status = lendingstate.LendingStatusCancelled
		}

		newTrades, newRejectedOrders, err := l.CommitOrder(header, coinbase, chain, statedb, lendingStatedb, tradingStateDb, lendingstate.GetLendingOrderBookHash(order.LendingToken, order.Term), order)
		for _, reject := range newRejectedOrders {
			log.Debug("Reject order", "reject", *reject)
		}

		switch err {
		case ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Debug("Skipping order with low nonce", "sender", tx.UserAddress(), "nonce", tx.Nonce())
			txs.Shift()
			continue

		case ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Debug("Skipping order account with high nonce", "sender", tx.UserAddress(), "nonce", tx.Nonce())
			txs.Pop()
			continue

		case nil:
			// everything ok
			txs.Shift()

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txs.Shift()
			continue
		}

		// orderID has been updated
		originalOrder.LendingId = order.LendingId
		originalOrder.ExtraData = order.ExtraData
		lendingItems = append(lendingItems, originalOrder)
		matchingResults[lendingstate.GetLendingCacheKey(order)] = lendingstate.MatchingResult{
			Trades:  newTrades,
			Rejects: newRejectedOrders,
		}
	}
	return lendingItems, matchingResults
}

// there are 3 tasks need to complete (for SDK nodes) after matching
// 1. Put takerLendingItem to database
// 2.a Update status, filledAmount of makerLendingItem
// 2.b. Put lendingTrade to database
// 3. Update status of rejected items
func (l *Lending) SyncDataToSDKNode(chain consensus.ChainContext, statedb *state.StateDB, block *types.Block, takerLendingItem *lendingstate.LendingItem, txHash common.Hash, txMatchTime time.Time, trades []*lendingstate.LendingTrade, rejectedItems []*lendingstate.LendingItem, dirtyOrderCount *uint64) error {
	var (
		// originTakerLendingItem: item getting from database
		originTakerLendingItem, updatedTakerLendingItem *lendingstate.LendingItem
		makerDirtyHashes                                []string
		makerDirtyFilledAmount                          map[string]*big.Int
		err                                             error
	)
	db := l.GetMongoDB()
	db.InitLendingBulk()
	if takerLendingItem.Status == lendingstate.LendingStatusCancelled && len(rejectedItems) > 0 {
		// cancel order is rejected -> nothing change
		log.Debug("Cancel order is rejected", "order", lendingstate.ToJSON(takerLendingItem))
		return nil
	}
	// 1. put processed takerLendingItem to database
	lastState := lendingstate.LendingItemHistoryItem{}
	// Typically, takerItem has never existed in database
	// except cancel case: in this case, item existed in database with status = OPEN, then use send another lendingItem to cancel it
	val, err := db.GetObject(takerLendingItem.Hash, &lendingstate.LendingItem{Type: takerLendingItem.Type})
	if err == nil && val != nil {
		originTakerLendingItem = val.(*lendingstate.LendingItem)
		lastState = lendingstate.LendingItemHistoryItem{
			TxHash:       originTakerLendingItem.TxHash,
			FilledAmount: lendingstate.CloneBigInt(originTakerLendingItem.FilledAmount),
			Status:       originTakerLendingItem.Status,
			UpdatedAt:    originTakerLendingItem.UpdatedAt,
		}
	}
	if originTakerLendingItem != nil {
		updatedTakerLendingItem = originTakerLendingItem
	} else {
		updatedTakerLendingItem = takerLendingItem
		updatedTakerLendingItem.FilledAmount = new(big.Int)
	}

	if takerLendingItem.Status == lendingstate.LendingStatusNew {
		updatedTakerLendingItem.Status = lendingstate.LendingStatusOpen
	} else if takerLendingItem.Status == lendingstate.LendingStatusCancelled {
		updatedTakerLendingItem.Status = lendingstate.LendingStatusCancelled
		updatedTakerLendingItem.ExtraData = takerLendingItem.ExtraData
	}
	updatedTakerLendingItem.TxHash = txHash
	if updatedTakerLendingItem.CreatedAt.IsZero() {
		updatedTakerLendingItem.CreatedAt = txMatchTime
	}
	if txMatchTime.Before(updatedTakerLendingItem.UpdatedAt) || (txMatchTime.Equal(updatedTakerLendingItem.UpdatedAt) && *dirtyOrderCount == 0) {
		log.Debug("Ignore old lendingItem/lendingTrades taker", "txHash", txHash.Hex(), "txTime", txMatchTime.UnixNano(), "updatedAt", updatedTakerLendingItem.UpdatedAt.UnixNano())
		return nil
	}
	*dirtyOrderCount++

	l.UpdateLendingItemCache(updatedTakerLendingItem.LendingToken, updatedTakerLendingItem.CollateralToken, updatedTakerLendingItem.Hash, txHash, lastState)
	updatedTakerLendingItem.UpdatedAt = txMatchTime

	// 2. put trades to database and update status
	log.Debug("Got lendingTrades", "number", len(trades), "txhash", txHash.Hex())
	makerDirtyFilledAmount = make(map[string]*big.Int)

	tradeList := map[common.Hash]*lendingstate.LendingTrade{}
	for _, tradeRecord := range trades {
		// 2.a. put to trades
		if tradeRecord == nil {
			continue
		}
		if updatedTakerLendingItem.Type == lendingstate.Repay || updatedTakerLendingItem.Type == lendingstate.TopUp || updatedTakerLendingItem.Type == lendingstate.Recall {
			// repay, topup: assign hash = trade.hash
			updatedTakerLendingItem.Hash = tradeRecord.Hash
			updatedTakerLendingItem.CollateralToken = tradeRecord.CollateralToken
			updatedTakerLendingItem.FilledAmount = updatedTakerLendingItem.Quantity
			updatedTakerLendingItem.Interest = new(big.Int).SetUint64(tradeRecord.Interest)
			switch updatedTakerLendingItem.Type {
			case lendingstate.TopUp:
				updatedTakerLendingItem.Status = lendingstate.TopUp
				extraData, _ := json.Marshal(struct {
					Price *big.Int
				}{
					Price: new(big.Int).Div(new(big.Int).Mul(tradeRecord.LiquidationPrice, tradeRecord.DepositRate), tradeRecord.LiquidationRate),
				})
				updatedTakerLendingItem.ExtraData = string(extraData)
				// manual topUp item
				updatedTakerLendingItem.AutoTopUp = false
			case lendingstate.Repay:
				updatedTakerLendingItem.Status = lendingstate.Repay
				paymentBalance := lendingstate.CalculateTotalRepayValue(block.Time().Uint64(), tradeRecord.LiquidationTime, tradeRecord.Term, tradeRecord.Interest, tradeRecord.Amount)
				updatedTakerLendingItem.Quantity = paymentBalance
				updatedTakerLendingItem.FilledAmount = paymentBalance
				// manual repay item
				updatedTakerLendingItem.AutoTopUp = false
			case lendingstate.Recall:
				updatedTakerLendingItem.Status = lendingstate.Recall
				// manual recall item
				updatedTakerLendingItem.AutoTopUp = false
			}

			log.Debug("UpdateLendingTrade:", "type", updatedTakerLendingItem.Type, "hash", tradeRecord.Hash.Hex(), "status", tradeRecord.Status, "tradeId", tradeRecord.TradeId)
			tradeList[tradeRecord.Hash] = tradeRecord
			continue

		}
		if tradeRecord.CreatedAt.IsZero() {
			tradeRecord.CreatedAt = txMatchTime
		}
		tradeRecord.UpdatedAt = txMatchTime
		tradeRecord.TxHash = txHash
		tradeRecord.Hash = tradeRecord.ComputeHash()
		tradeList[tradeRecord.Hash] = tradeRecord

		// 2.b. update status and filledAmount
		filledAmount := new(big.Int)
		if tradeRecord.Amount != nil {
			filledAmount = lendingstate.CloneBigInt(tradeRecord.Amount)
		}
		// maker dirty order
		makerFilledAmount := big.NewInt(0)
		makerOrderHash := common.Hash{}
		if updatedTakerLendingItem.Side == lendingstate.Borrowing {
			makerOrderHash = tradeRecord.InvestingOrderHash
		} else {
			makerOrderHash = tradeRecord.BorrowingOrderHash
		}
		if amount, ok := makerDirtyFilledAmount[makerOrderHash.Hex()]; ok {
			makerFilledAmount = lendingstate.CloneBigInt(amount)
		}
		makerFilledAmount = new(big.Int).Add(makerFilledAmount, filledAmount)
		makerDirtyFilledAmount[makerOrderHash.Hex()] = makerFilledAmount
		makerDirtyHashes = append(makerDirtyHashes, makerOrderHash.Hex())

		if updatedTakerLendingItem.Type == lendingstate.Limit || updatedTakerLendingItem.Type == lendingstate.Market {
			//updatedTakerOrder = l.updateMatchedOrder(updatedTakerOrder, filledAmount, txMatchTime, txHash)
			//  update filledAmount, status of takerOrder
			updatedTakerLendingItem.FilledAmount = new(big.Int).Add(updatedTakerLendingItem.FilledAmount, filledAmount)
			if updatedTakerLendingItem.FilledAmount.Cmp(updatedTakerLendingItem.Quantity) < 0 && updatedTakerLendingItem.Type == lendingstate.Limit {
				updatedTakerLendingItem.Status = lendingstate.LendingStatusPartialFilled
			} else {
				updatedTakerLendingItem.Status = lendingstate.LendingStatusFilled
			}
		}
	}
	if err := l.UpdateLendingTrade(tradeList, txHash, txMatchTime); err != nil {
		return err
	}

	// for Market orders
	// filledAmount > 0 : FILLED
	// otherwise: REJECTED
	if updatedTakerLendingItem.Type == lendingstate.Market {
		if updatedTakerLendingItem.FilledAmount.Sign() > 0 {
			updatedTakerLendingItem.Status = lendingstate.LendingStatusFilled
		} else {
			updatedTakerLendingItem.Status = lendingstate.LendingStatusReject
		}
	}

	log.Debug("PutObject processed takerLendingItem",
		"term", updatedTakerLendingItem.Term, "userAddr", updatedTakerLendingItem.UserAddress.Hex(), "side", updatedTakerLendingItem.Side,
		"Interest", updatedTakerLendingItem.Interest, "quantity", updatedTakerLendingItem.Quantity, "filledAmount", updatedTakerLendingItem.FilledAmount, "status", updatedTakerLendingItem.Status,
		"hash", updatedTakerLendingItem.Hash.Hex(), "txHash", updatedTakerLendingItem.TxHash.Hex())

	if !(updatedTakerLendingItem.Type == lendingstate.Repay || updatedTakerLendingItem.Type == lendingstate.TopUp || updatedTakerLendingItem.Type == lendingstate.Recall) || updatedTakerLendingItem.Status != lendingstate.LendingStatusOpen {
		if err := db.PutObject(updatedTakerLendingItem.Hash, updatedTakerLendingItem); err != nil {
			return fmt.Errorf("SDKNode: failed to put processed takerOrder. Hash: %s Error: %s", updatedTakerLendingItem.Hash.Hex(), err.Error())
		}
	}

	items := db.GetListItemByHashes(makerDirtyHashes, &lendingstate.LendingItem{})
	if items != nil {
		makerItems := items.([]*lendingstate.LendingItem)
		log.Debug("Maker dirty lendingItem", "len", len(makerItems), "txhash", txHash.Hex())
		for _, m := range makerItems {
			if txMatchTime.Before(m.UpdatedAt) {
				log.Debug("Ignore old lendingItem/lendingTrades maker", "txHash", txHash.Hex(), "txTime", txMatchTime.UnixNano(), "updatedAt", m.UpdatedAt.UnixNano())
				continue
			}
			lastState = lendingstate.LendingItemHistoryItem{
				TxHash:       m.TxHash,
				FilledAmount: lendingstate.CloneBigInt(m.FilledAmount),
				Status:       m.Status,
				UpdatedAt:    m.UpdatedAt,
			}
			l.UpdateLendingItemCache(m.LendingToken, m.CollateralToken, m.Hash, txHash, lastState)
			m.TxHash = txHash
			m.UpdatedAt = txMatchTime
			m.FilledAmount = new(big.Int).Add(m.FilledAmount, makerDirtyFilledAmount[m.Hash.Hex()])
			if m.FilledAmount.Cmp(m.Quantity) < 0 {
				m.Status = lendingstate.LendingStatusPartialFilled
			} else {
				m.Status = lendingstate.LendingStatusFilled
			}
			log.Debug("PutObject processed makerLendingItem",
				"term", m.Term, "userAddr", m.UserAddress.Hex(), "side", m.Side,
				"Interest", m.Interest, "quantity", m.Quantity, "filledAmount", m.FilledAmount, "status", m.Status,
				"hash", m.Hash.Hex(), "txHash", m.TxHash.Hex())
			if err := db.PutObject(m.Hash, m); err != nil {
				return fmt.Errorf("SDKNode: failed to put processed makerOrder. Hash: %s Error: %s", m.Hash.Hex(), err.Error())
			}
		}
	}

	// 3. put rejected orders to leveldb and update status REJECTED
	log.Debug("Got rejected lendingItems", "number", len(rejectedItems), "rejectedLendingItems", rejectedItems)

	if len(rejectedItems) > 0 {
		var rejectedHashes []string
		// updateRejectedOrders
		for _, r := range rejectedItems {
			rejectedHashes = append(rejectedHashes, r.Hash.Hex())
			if updatedTakerLendingItem.Hash == r.Hash && !txMatchTime.Before(r.UpdatedAt) {
				// cache r history for handling reorg
				historyRecord := lendingstate.LendingItemHistoryItem{
					TxHash:       updatedTakerLendingItem.TxHash,
					FilledAmount: lendingstate.CloneBigInt(updatedTakerLendingItem.FilledAmount),
					Status:       updatedTakerLendingItem.Status,
					UpdatedAt:    updatedTakerLendingItem.UpdatedAt,
				}
				l.UpdateLendingItemCache(updatedTakerLendingItem.LendingToken, updatedTakerLendingItem.CollateralToken, updatedTakerLendingItem.Hash, txHash, historyRecord)
				// if whole order is rejected, status = REJECTED
				// otherwise, status = FILLED
				if updatedTakerLendingItem.FilledAmount.Sign() > 0 {
					updatedTakerLendingItem.Status = lendingstate.LendingStatusFilled
				} else {
					updatedTakerLendingItem.Status = lendingstate.LendingStatusReject
				}
				updatedTakerLendingItem.TxHash = txHash
				updatedTakerLendingItem.UpdatedAt = txMatchTime
				if err := db.PutObject(updatedTakerLendingItem.Hash, updatedTakerLendingItem); err != nil {
					return fmt.Errorf("SDKNode: failed to reject takerOrder. Hash: %s Error: %s", updatedTakerLendingItem.Hash.Hex(), err.Error())
				}
			}
		}
		items := db.GetListItemByHashes(rejectedHashes, &lendingstate.LendingItem{})
		if items != nil {
			dirtyRejectedItems := items.([]*lendingstate.LendingItem)
			for _, r := range dirtyRejectedItems {
				if txMatchTime.Before(r.UpdatedAt) {
					log.Debug("Ignore old orders/trades reject", "txHash", txHash.Hex(), "txTime", txMatchTime.UnixNano(), "updatedAt", updatedTakerLendingItem.UpdatedAt.UnixNano())
					continue
				}
				// cache lendingItem for handling reorg
				historyRecord := lendingstate.LendingItemHistoryItem{
					TxHash:       r.TxHash,
					FilledAmount: lendingstate.CloneBigInt(r.FilledAmount),
					Status:       r.Status,
					UpdatedAt:    r.UpdatedAt,
				}
				l.UpdateLendingItemCache(r.LendingToken, r.CollateralToken, r.Hash, txHash, historyRecord)
				dirtyFilledAmount, ok := makerDirtyFilledAmount[r.Hash.Hex()]
				if ok && dirtyFilledAmount != nil {
					r.FilledAmount = new(big.Int).Add(r.FilledAmount, dirtyFilledAmount)
				}
				// if whole order is rejected, status = REJECTED
				// otherwise, status = FILLED
				if r.FilledAmount.Sign() > 0 {
					r.Status = lendingstate.LendingStatusFilled
				} else {
					r.Status = lendingstate.LendingStatusReject
				}
				r.TxHash = txHash
				r.UpdatedAt = txMatchTime
				if err = db.PutObject(r.Hash, r); err != nil {
					return fmt.Errorf("SDKNode: failed to update rejectedOder to sdkNode %s", err.Error())
				}
			}
		}
	}

	if err := db.CommitLendingBulk(); err != nil {
		return fmt.Errorf("SDKNode fail to commit bulk update lendingItem/lendingTrades at txhash %s . Error: %s", txHash.Hex(), err.Error())
	}
	return nil
}

func (l *Lending) UpdateLiquidatedTrade(blockTime uint64, result lendingstate.FinalizedResult, trades map[common.Hash]*lendingstate.LendingTrade) error {
	db := l.GetMongoDB()
	db.InitLendingBulk()

	txhash := result.TxHash
	txTime := time.Unix(int64(blockTime), 0).UTC()
	if err := l.UpdateLendingTrade(trades, txhash, txTime); err != nil {
		return err
	}

	// adding auto repay transaction
	if len(result.AutoRepay) > 0 {
		for _, hash := range result.AutoRepay {
			trade := trades[hash]
			if trade == nil {
				continue
			}
			paymentBalance := lendingstate.CalculateTotalRepayValue(blockTime, trade.LiquidationTime, trade.Term, trade.Interest, trade.Amount)
			repayItem := &lendingstate.LendingItem{
				Quantity:        paymentBalance,
				Interest:        big.NewInt(int64(trade.Interest)),
				Side:            "",
				Type:            lendingstate.Repay,
				LendingToken:    trade.LendingToken,
				CollateralToken: trade.CollateralToken,
				FilledAmount:    paymentBalance,
				Status:          lendingstate.Repay,
				Relayer:         trade.BorrowingRelayer,
				Term:            trade.Term,
				UserAddress:     trade.Borrower,
				Signature:       nil,
				Hash:            trade.Hash,
				TxHash:          txhash,
				Nonce:           nil,
				CreatedAt:       txTime,
				UpdatedAt:       txTime,
				LendingId:       0,
				LendingTradeId:  trade.TradeId,
				AutoTopUp:       true, // auto repay
				ExtraData:       "",
			}
			if err := db.PutObject(repayItem.Hash, repayItem); err != nil {
				return err
			}
		}
	}

	// adding auto topup transaction
	if len(result.AutoTopUp) > 0 {
		oldTradeHashes := []string{}
		for _, hash := range result.AutoTopUp {
			oldTradeHashes = append(oldTradeHashes, hash.Hex())
		}
		items := db.GetListItemByHashes(oldTradeHashes, &lendingstate.LendingTrade{})
		if items != nil && len(items.([]*lendingstate.LendingTrade)) > 0 {
			for _, oldTrade := range items.([]*lendingstate.LendingTrade) {
				newTrade := trades[oldTrade.Hash]
				topUpAmount := new(big.Int).Sub(newTrade.CollateralLockedAmount, oldTrade.CollateralLockedAmount)
				extraData, _ := json.Marshal(struct {
					Price *big.Int
				}{
					Price: new(big.Int).Div(new(big.Int).Mul(newTrade.LiquidationPrice, common.BaseTopUp), common.RateTopUp),
				})
				topUpItem := &lendingstate.LendingItem{
					Quantity:        topUpAmount,
					Interest:        big.NewInt(int64(oldTrade.Interest)),
					Side:            "",
					Type:            lendingstate.TopUp,
					LendingToken:    oldTrade.LendingToken,
					CollateralToken: oldTrade.CollateralToken,
					FilledAmount:    topUpAmount,
					Status:          lendingstate.TopUp,
					AutoTopUp:       true, // auto topup
					Relayer:         oldTrade.BorrowingRelayer,
					Term:            oldTrade.Term,
					UserAddress:     oldTrade.Borrower,
					Signature:       nil,
					Hash:            oldTrade.Hash,
					TxHash:          txhash,
					Nonce:           nil,
					CreatedAt:       txTime,
					UpdatedAt:       txTime,
					LendingId:       0,
					LendingTradeId:  oldTrade.TradeId,
					ExtraData:       string(extraData),
				}
				if err := db.PutObject(topUpItem.Hash, topUpItem); err != nil {
					return err
				}
			}
		}
	}

	// adding auto recall transaction
	if len(result.AutoRecall) > 0 {
		oldTradeHashes := []string{}
		for _, hash := range result.AutoRecall {
			oldTradeHashes = append(oldTradeHashes, hash.Hex())
		}
		items := db.GetListItemByHashes(oldTradeHashes, &lendingstate.LendingTrade{})
		if items != nil && len(items.([]*lendingstate.LendingTrade)) > 0 {
			for _, oldTrade := range items.([]*lendingstate.LendingTrade) {
				newTrade := trades[oldTrade.Hash]
				recallAmount := new(big.Int).Sub(oldTrade.CollateralLockedAmount, newTrade.CollateralLockedAmount)
				extraData, _ := json.Marshal(struct {
					Price *big.Int
				}{
					Price: new(big.Int).Div(new(big.Int).Mul(newTrade.LiquidationPrice, oldTrade.DepositRate), oldTrade.LiquidationRate),
				})
				topUpItem := &lendingstate.LendingItem{
					Quantity:        recallAmount,
					Interest:        big.NewInt(int64(oldTrade.Interest)),
					Side:            "",
					Type:            lendingstate.Recall,
					LendingToken:    oldTrade.LendingToken,
					CollateralToken: oldTrade.CollateralToken,
					FilledAmount:    recallAmount,
					Status:          lendingstate.Recall,
					AutoTopUp:       true, // auto recall
					Relayer:         oldTrade.BorrowingRelayer,
					Term:            oldTrade.Term,
					UserAddress:     oldTrade.Borrower,
					Signature:       nil,
					Hash:            oldTrade.Hash,
					TxHash:          txhash,
					Nonce:           nil,
					CreatedAt:       txTime,
					UpdatedAt:       txTime,
					LendingId:       0,
					LendingTradeId:  oldTrade.TradeId,
					ExtraData:       string(extraData),
				}
				if err := db.PutObject(topUpItem.Hash, topUpItem); err != nil {
					return err
				}
			}
		}
	}

	if err := db.CommitLendingBulk(); err != nil {
		return fmt.Errorf("failed to updateLendingTrade . Err: %v", err)
	}

	return nil
}

func (l *Lending) UpdateLendingTrade(trades map[common.Hash]*lendingstate.LendingTrade, txhash common.Hash, txTime time.Time) error {
	db := l.GetMongoDB()
	hashQuery := []string{}
	if len(trades) == 0 {
		return nil
	}
	for _, trade := range trades {
		hashQuery = append(hashQuery, trade.Hash.Hex())
	}
	items := db.GetListItemByHashes(hashQuery, &lendingstate.LendingTrade{})
	if items != nil && len(items.([]*lendingstate.LendingTrade)) > 0 {
		for _, trade := range items.([]*lendingstate.LendingTrade) {
			history := lendingstate.LendingTradeHistoryItem{
				TxHash:                 trade.TxHash,
				CollateralLockedAmount: trade.CollateralLockedAmount,
				LiquidationPrice:       trade.LiquidationPrice,
				Status:                 trade.Status,
				UpdatedAt:              trade.UpdatedAt,
			}
			l.UpdateLendingTradeCache(trade.Hash, txhash, history)
			trade.TxHash = txhash
			trade.UpdatedAt = txTime

			newTrade := trades[trade.Hash]
			trade.CollateralLockedAmount = newTrade.CollateralLockedAmount
			trade.Status = newTrade.Status
			trade.LiquidationPrice = newTrade.LiquidationPrice
			trade.ExtraData = newTrade.ExtraData

			if err := db.PutObject(trade.Hash, trade); err != nil {
				return err
			}
		}
		log.Debug("UpdateLendingTrade successfully", "txhash", txhash, "hash", hashQuery)
	} else {
		// not update, just upsert
		for _, trade := range trades {
			if err := db.PutObject(trade.Hash, trade); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Lending) GetLendingState(block *types.Block, author common.Address) (*lendingstate.LendingStateDB, error) {
	root, err := l.GetLendingStateRoot(block, author)
	if err != nil {
		return nil, err
	}
	if l.StateCache == nil {
		return nil, errors.New("Not initialized XDCx")
	}
	state, err := lendingstate.New(root, l.StateCache)
	if err != nil {
		log.Info("Not found lending state when GetLendingState", "block", block.Number(), "lendingRoot", root.Hex())
	}
	return state, err
}

func (l *Lending) GetStateCache() lendingstate.Database {
	return l.StateCache
}

func (l *Lending) HasLendingState(block *types.Block, author common.Address) bool {
	root, err := l.GetLendingStateRoot(block, author)
	if err != nil {
		return false
	}
	_, err = l.StateCache.OpenTrie(root)
	if err != nil {
		return false
	}
	return true
}

func (l *Lending) GetTriegc() *prque.Prque {
	return l.Triegc
}

func (l *Lending) GetLendingStateRoot(block *types.Block, author common.Address) (common.Hash, error) {
	for _, tx := range block.Transactions() {
		from := *(tx.From())
		if tx.To() != nil && tx.To().Hex() == common.TradingStateAddr && from.String() == author.String() {
			if len(tx.Data()) >= 64 {
				return common.BytesToHash(tx.Data()[32:]), nil
			}
		}
	}
	return lendingstate.EmptyRoot, nil
}

func (l *Lending) UpdateLendingItemCache(LendingToken, CollateralToken common.Address, hash common.Hash, txhash common.Hash, lastState lendingstate.LendingItemHistoryItem) {
	var lendingCacheAtTxHash map[common.Hash]lendingstate.LendingItemHistoryItem
	c, ok := l.lendingItemHistory.Get(txhash)
	if !ok || c == nil {
		lendingCacheAtTxHash = make(map[common.Hash]lendingstate.LendingItemHistoryItem)
	} else {
		lendingCacheAtTxHash = c.(map[common.Hash]lendingstate.LendingItemHistoryItem)
	}
	orderKey := lendingstate.GetLendingItemHistoryKey(LendingToken, CollateralToken, hash)
	_, ok = lendingCacheAtTxHash[orderKey]
	if !ok {
		lendingCacheAtTxHash[orderKey] = lastState
	}
	l.lendingItemHistory.Add(txhash, lendingCacheAtTxHash)
}

func (l *Lending) UpdateLendingTradeCache(hash common.Hash, txhash common.Hash, lastState lendingstate.LendingTradeHistoryItem) {
	var lendingCacheAtTxHash map[common.Hash]lendingstate.LendingTradeHistoryItem
	c, ok := l.lendingTradeHistory.Get(txhash)
	if !ok || c == nil {
		lendingCacheAtTxHash = make(map[common.Hash]lendingstate.LendingTradeHistoryItem)
	} else {
		lendingCacheAtTxHash = c.(map[common.Hash]lendingstate.LendingTradeHistoryItem)
	}
	_, ok = lendingCacheAtTxHash[hash]
	if !ok {
		lendingCacheAtTxHash[hash] = lastState
	}
	l.lendingTradeHistory.Add(txhash, lendingCacheAtTxHash)
}

func (l *Lending) RollbackLendingData(txhash common.Hash) error {
	db := l.GetMongoDB()
	db.InitLendingBulk()

	// rollback lendingItem
	items := db.GetListItemByTxHash(txhash, &lendingstate.LendingItem{})
	if items != nil {
		for _, item := range items.([]*lendingstate.LendingItem) {
			c, ok := l.lendingItemHistory.Get(txhash)
			log.Debug("XDCxlending reorg: rollback lendingItem", "txhash", txhash.Hex(), "item", lendingstate.ToJSON(item), "lendingItemHistory", c)
			if !ok {
				log.Debug("XDCxlending reorg: remove item due to no lendingItemHistory", "item", lendingstate.ToJSON(item))
				if err := db.DeleteObject(item.Hash, &lendingstate.LendingItem{}); err != nil {
					return fmt.Errorf("failed to remove reorg LendingItem. Err: %v . Item: %s", err.Error(), lendingstate.ToJSON(item))
				}
				continue
			}
			cacheAtTxHash := c.(map[common.Hash]lendingstate.LendingItemHistoryItem)
			lendingItemHistory, _ := cacheAtTxHash[lendingstate.GetLendingItemHistoryKey(item.LendingToken, item.CollateralToken, item.Hash)]
			if (lendingItemHistory == lendingstate.LendingItemHistoryItem{}) {
				log.Debug("XDCxlending reorg: remove item due to empty lendingItemHistory", "item", lendingstate.ToJSON(item))
				if err := db.DeleteObject(item.Hash, &lendingstate.LendingItem{}); err != nil {
					return fmt.Errorf("failed to remove reorg LendingItem. Err: %v . Item: %s", err.Error(), lendingstate.ToJSON(item))
				}
				continue
			}
			item.TxHash = lendingItemHistory.TxHash
			item.Status = lendingItemHistory.Status
			item.FilledAmount = lendingstate.CloneBigInt(lendingItemHistory.FilledAmount)
			item.UpdatedAt = lendingItemHistory.UpdatedAt
			log.Debug("XDCxlending reorg: update item to the last lendingItemHistory", "item", lendingstate.ToJSON(item), "lendingItemHistory", lendingItemHistory)
			if err := db.PutObject(item.Hash, item); err != nil {
				return fmt.Errorf("failed to update reorg LendingItem. Err: %v . Item: %s", err.Error(), lendingstate.ToJSON(item))
			}
		}
	}

	// rollback lendingTrade
	items = db.GetListItemByTxHash(txhash, &lendingstate.LendingTrade{})
	if items != nil {
		for _, trade := range items.([]*lendingstate.LendingTrade) {
			c, ok := l.lendingTradeHistory.Get(txhash)
			log.Debug("XDCxlending reorg: rollback LendingTrade", "txhash", txhash.Hex(), "trade", lendingstate.ToJSON(trade), "LendingTradeHistory", c)
			if !ok {
				log.Debug("XDCxlending reorg: remove trade due to no LendingTradeHistory", "trade", lendingstate.ToJSON(trade))
				if err := db.DeleteObject(trade.Hash, &lendingstate.LendingTrade{}); err != nil {
					return fmt.Errorf("failed to remove reorg LendingTrade. Err: %v . Trade: %s", err.Error(), lendingstate.ToJSON(trade))
				}
				continue
			}
			cacheAtTxHash := c.(map[common.Hash]lendingstate.LendingTradeHistoryItem)
			lendingTradeHistoryItem, _ := cacheAtTxHash[trade.Hash]
			if (lendingTradeHistoryItem == lendingstate.LendingTradeHistoryItem{}) {
				log.Debug("XDCxlending reorg: remove trade due to empty LendingTradeHistory", "trade", lendingstate.ToJSON(trade))
				if err := db.DeleteObject(trade.Hash, &lendingstate.LendingTrade{}); err != nil {
					return fmt.Errorf("failed to remove reorg LendingTrade. Err: %v . Trade: %s", err.Error(), lendingstate.ToJSON(trade))
				}
				continue
			}
			trade.TxHash = lendingTradeHistoryItem.TxHash
			trade.Status = lendingTradeHistoryItem.Status
			trade.CollateralLockedAmount = lendingstate.CloneBigInt(lendingTradeHistoryItem.CollateralLockedAmount)
			trade.LiquidationPrice = lendingstate.CloneBigInt(lendingTradeHistoryItem.LiquidationPrice)
			trade.UpdatedAt = lendingTradeHistoryItem.UpdatedAt
			log.Debug("XDCxlending reorg: update trade to the last lendingTradeHistoryItem", "trade", lendingstate.ToJSON(trade), "lendingTradeHistoryItem", lendingTradeHistoryItem)
			if err := db.PutObject(trade.Hash, trade); err != nil {
				return fmt.Errorf("failed to update reorg LendingTrade. Err: %v . Trade: %s", err.Error(), lendingstate.ToJSON(trade))
			}
		}
	}

	// remove repay/topup/recall history
	db.DeleteItemByTxHash(txhash, &lendingstate.LendingItem{Type: lendingstate.Repay})
	db.DeleteItemByTxHash(txhash, &lendingstate.LendingItem{Type: lendingstate.TopUp})
	db.DeleteItemByTxHash(txhash, &lendingstate.LendingItem{Type: lendingstate.Recall})

	if err := db.CommitLendingBulk(); err != nil {
		return fmt.Errorf("failed to RollbackLendingData. %v", err)
	}
	return nil
}

func (l *Lending) ProcessLiquidationData(header *types.Header, chain consensus.ChainContext, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, lendingState *lendingstate.LendingStateDB) (updatedTrades map[common.Hash]*lendingstate.LendingTrade, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades []*lendingstate.LendingTrade, err error) {
	time := header.Time
	updatedTrades = map[common.Hash]*lendingstate.LendingTrade{} // sum of liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades
	liquidatedTrades = []*lendingstate.LendingTrade{}
	autoRepayTrades = []*lendingstate.LendingTrade{}
	autoTopUpTrades = []*lendingstate.LendingTrade{}
	autoRecallTrades = []*lendingstate.LendingTrade{}

	allPairs, err := lendingstate.GetAllLendingPairs(statedb)
	if err != nil {
		log.Debug("Not found all trading pairs", "error", err)
		return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
	}
	allLendingBooks, err := lendingstate.GetAllLendingBooks(statedb)
	if err != nil {
		log.Debug("Not found all lending books", "error", err)
		return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
	}

	// liquidate trades by time
	for lendingBook := range allLendingBooks {
		lowestTime, tradingIds := lendingState.GetLowestLiquidationTime(lendingBook, time)
		log.Debug("ProcessLiquidationData time", "tradeIds", len(tradingIds))
		for lowestTime.Sign() > 0 && lowestTime.Cmp(time) < 0 {
			for _, tradingId := range tradingIds {
				log.Debug("ProcessRepay", "lowestTime", lowestTime, "time", time, "lendingBook", lendingBook.Hex(), "tradingId", tradingId.Hex())
				trade, err := l.ProcessRepayLendingTrade(header, chain, lendingState, statedb, tradingState, lendingBook, tradingId.Big().Uint64())
				if err != nil {
					log.Error("Fail when process payment ", "time", time, "lendingBook", lendingBook.Hex(), "tradingId", tradingId, "error", err)
					return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
				}
				if trade != nil && trade.Hash != (common.Hash{}) {
					updatedTrades[trade.Hash] = trade
					if trade.Status == lendingstate.TradeStatusLiquidated {
						liquidatedTrades = append(liquidatedTrades, trade)
					} else if trade.Status == lendingstate.TradeStatusClosed {
						autoRepayTrades = append(autoRepayTrades, trade)
					}
				}
			}
			lowestTime, tradingIds = lendingState.GetLowestLiquidationTime(lendingBook, time)
		}
	}

	for _, lendingPair := range allPairs {
		orderbook := tradingstate.GetTradingOrderBookHash(lendingPair.CollateralToken, lendingPair.LendingToken)
		_, collateralPrice, err := l.GetCollateralPrices(header, chain, statedb, tradingState, lendingPair.CollateralToken, lendingPair.LendingToken)
		if err != nil || collateralPrice == nil || collateralPrice.Sign() == 0 {
			log.Error("Fail when get price collateral/lending ", "CollateralToken", lendingPair.CollateralToken.Hex(), "LendingToken", lendingPair.LendingToken.Hex(), "error", err)
			// ignore this pair, do not throw error
			continue
		}
		// liquidate trades
		highestLiquidatePrice, liquidationData := tradingState.GetHighestLiquidationPriceData(orderbook, collateralPrice)
		for highestLiquidatePrice.Sign() > 0 && collateralPrice.Cmp(highestLiquidatePrice) < 0 {
			for lendingBook, tradingIds := range liquidationData {
				for _, tradingIdHash := range tradingIds {
					trade := lendingState.GetLendingTrade(lendingBook, tradingIdHash)
					if trade.AutoTopUp {
						if newTrade, err := l.AutoTopUp(statedb, tradingState, lendingState, lendingBook, tradingIdHash, collateralPrice); err == nil {
							// if this action complete successfully, do not liquidate this trade in this epoch
							log.Debug("AutoTopUp", "borrower", trade.Borrower.Hex(), "collateral", newTrade.CollateralToken.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLockedAmount", newTrade.CollateralLockedAmount)
							autoTopUpTrades = append(autoTopUpTrades, newTrade)
							updatedTrades[newTrade.Hash] = newTrade
							continue
						}
					}
					log.Debug("LiquidationTrade", "highestLiquidatePrice", highestLiquidatePrice, "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex())
					newTrade, err := l.LiquidationTrade(lendingState, statedb, tradingState, lendingBook, tradingIdHash.Big().Uint64())
					if err != nil {
						log.Error("Fail when remove liquidation newTrade", "time", time, "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "error", err)
						return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
					}
					if newTrade != nil && newTrade.Hash != (common.Hash{}) {
						newTrade.Status = lendingstate.TradeStatusLiquidated
						liquidationData := lendingstate.LiquidationData{
							RecallAmount:      common.Big0,
							LiquidationAmount: newTrade.CollateralLockedAmount,
							CollateralPrice:   collateralPrice,
							Reason:            lendingstate.LiquidatedByPrice,
						}
						extraData, _ := json.Marshal(liquidationData)
						newTrade.ExtraData = string(extraData)
						liquidatedTrades = append(liquidatedTrades, newTrade)
						updatedTrades[newTrade.Hash] = newTrade
					}
				}
			}
			highestLiquidatePrice, liquidationData = tradingState.GetHighestLiquidationPriceData(orderbook, collateralPrice)
		}
		// recall trades
		depositRate, liquidationRate, recallRate := lendingstate.GetCollateralDetail(statedb, lendingPair.CollateralToken)
		recalLiquidatePrice := new(big.Int).Mul(collateralPrice, common.BaseRecall)
		recalLiquidatePrice = new(big.Int).Div(recalLiquidatePrice, recallRate)
		newLiquidatePrice := new(big.Int).Mul(collateralPrice, liquidationRate)
		newLiquidatePrice = new(big.Int).Div(newLiquidatePrice, depositRate)
		allLowertLiquidationData := tradingState.GetAllLowerLiquidationPriceData(orderbook, recalLiquidatePrice)
		log.Debug("ProcessLiquidationData", "orderbook", orderbook.Hex(), "collateralPrice", collateralPrice, "recallRate", recallRate, "recalLiquidatePrice", recalLiquidatePrice, "newLiquidatePrice", newLiquidatePrice, "allLowertLiquidationData", len(allLowertLiquidationData))
		for price, liquidationData := range allLowertLiquidationData {
			if price.Sign() > 0 && recalLiquidatePrice.Cmp(price) > 0 {
				for lendingBook, tradingIds := range liquidationData {
					for _, tradingIdHash := range tradingIds {
						log.Debug("Process Recall", "price", price, "lendingBook", lendingBook, "tradingIdHash", tradingIdHash.Hex())
						trade := lendingState.GetLendingTrade(lendingBook, tradingIdHash)
						log.Debug("TestRecall", "borrower", trade.Borrower.Hex(), "lendingToken", trade.LendingToken.Hex(), "collateral", trade.CollateralToken.Hex(), "price", price, "tradingIdHash", tradingIdHash.Hex())
						if trade.AutoTopUp {
							err, _, newTrade := l.ProcessRecallLendingTrade(lendingState, statedb, tradingState, lendingBook, tradingIdHash, newLiquidatePrice)
							if err != nil {
								log.Error("ProcessRecallLendingTrade", "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLiquidatePrice", newLiquidatePrice, "err", err)
								return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
							}
							// if this action complete successfully, do not liquidate this trade in this epoch
							log.Debug("AutoRecall", "borrower", trade.Borrower.Hex(), "collateral", newTrade.CollateralToken.Hex(), "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLockedAmount", newTrade.CollateralLockedAmount)
							autoRecallTrades = append(autoRecallTrades, newTrade)
							updatedTrades[newTrade.Hash] = newTrade
						}
					}
				}
			}
		}
	}

	log.Debug("ProcessLiquidationData", "updatedTrades", len(updatedTrades), "liquidated", len(liquidatedTrades), "autoRepay", len(autoRepayTrades), "autoTopUp", len(autoTopUpTrades), "autoRecall", len(autoRecallTrades))
	return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
}
