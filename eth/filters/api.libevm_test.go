// Copyright 2026 the libevm authors.
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

package filters

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/event"
)

// A closeableTestBackend tracks all subscriptions that it produces, allowing
// them to be cleaned up to avoid the leak of [EventSystem.eventLoop].
type closeableTestBackend struct {
	testBackend
	subs event.SubscriptionScope
}

func (b *closeableTestBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	sub := b.testBackend.SubscribeNewTxsEvent(ch)
	return b.subs.Track(sub)
}

func (b *closeableTestBackend) Close() {
	b.subs.Close()
}

func TestClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	backend := &closeableTestBackend{
		testBackend: testBackend{
			db: rawdb.NewMemoryDatabase(),
		},
	}
	defer backend.Close()
	sys := NewFilterSystem(backend, Config{})
	api := NewFilterAPI(sys, false)
	CloseAPI(api)
}
