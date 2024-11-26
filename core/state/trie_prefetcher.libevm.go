// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package state

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm/options"
	"github.com/ava-labs/libevm/libevm/sync"
	"github.com/ava-labs/libevm/log"
)

// A PrefetcherOption configures behaviour of trie prefetching.
type PrefetcherOption = options.Option[prefetcherConfig]

type prefetcherConfig struct {
	newWorkers func() WorkerPool
}

// A WorkerPool executes functions asynchronously. Done() is called to signal
// that the pool is no longer needed and that Execute() is guaranteed to not be
// called again.
type WorkerPool interface {
	Execute(func())
	Done()
}

// WithWorkerPools configures trie prefetching to execute asynchronously. The
// provided constructor is called once for each trie being fetched but it MAY
// return the same pool.
func WithWorkerPools(ctor func() WorkerPool) PrefetcherOption {
	return options.Func[prefetcherConfig](func(c *prefetcherConfig) {
		c.newWorkers = ctor
	})
}

type subfetcherPool struct {
	workers WorkerPool
	tries   sync.Pool[Trie]
	wg      sync.WaitGroup
}

// applyTo configures the [subfetcher] to use a [WorkerPool] if one was provided
// with a [PrefetcherOption].
func (c *prefetcherConfig) applyTo(sf *subfetcher) {
	sf.pool = &subfetcherPool{
		tries: sync.Pool[Trie]{
			// Although the workers may be shared between all subfetchers, each
			// MUST have its own Trie pool.
			New: func() Trie {
				return sf.db.CopyTrie(sf.trie)
			},
		},
	}
	if c.newWorkers != nil {
		sf.pool.workers = c.newWorkers()
	}
}

// releaseWorkerPools calls Done() on all [WorkerPool]s. This MUST only be
// called after [subfetcher.abort] returns on ALL fetchers as a pool is allowed
// to be shared between them. This is because we guarantee in the public API
// that no further calls will be made to Execute() after a call to Done().
func (p *triePrefetcher) releaseWorkerPools() {
	for _, f := range p.fetchers {
		if w := f.pool.workers; w != nil {
			w.Done()
		}
	}
}

func (p *subfetcherPool) wait() {
	p.wg.Wait()
}

// execute runs the provided function with a copy of the subfetcher's Trie.
// Copies are stored in a [sync.Pool] to reduce creation overhead. If p was
// configured with a [WorkerPool] then it is used for function execution,
// otherwise `fn` is just called directly.
func (p *subfetcherPool) execute(fn func(Trie)) {
	p.wg.Add(1)
	do := func() {
		t := p.tries.Get()
		fn(t)
		p.tries.Put(t)
		p.wg.Done()
	}

	if w := p.workers; w != nil {
		w.Execute(do)
	} else {
		do()
	}
}

// GetAccount optimistically pre-fetches an account, dropping the returned value
// and logging errors. See [subfetcherPool.execute] re worker pools.
func (p *subfetcherPool) GetAccount(addr common.Address) {
	p.execute(func(t Trie) {
		if _, err := t.GetAccount(addr); err != nil {
			log.Error("account prefetching failed", "address", addr, "err", err)
		}
	})
}

// GetStorage is the storage equivalent of [subfetcherPool.GetAccount].
func (p *subfetcherPool) GetStorage(addr common.Address, key []byte) {
	p.execute(func(t Trie) {
		if _, err := t.GetStorage(addr, key); err != nil {
			log.Error("storage prefetching failed", "address", addr, "key", key, "err", err)
		}
	})
}
