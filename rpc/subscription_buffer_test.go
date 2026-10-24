// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type bufferedNotificationTestService struct {
	notifiers chan *Notifier
}

func (s *bufferedNotificationTestService) Buffered(ctx context.Context, count int) (*Subscription, error) {
	notifier, _ := NotifierFromContext(ctx)
	sub := notifier.CreateSubscription()
	for i := 0; i < count; i++ {
		if err := notifier.Notify(sub.ID, &types.Log{Topics: []common.Hash{}, Index: uint(i), Data: []byte{byte(i)}}); err != nil {
			return nil, err
		}
	}
	s.notifiers <- notifier
	return sub, nil
}

// Notifications buffered before the subscribe response must be delivered in order
// and released while the subscription is still active.
func TestSubscriptionReleasesBufferedNotifications(t *testing.T) {
	for _, count := range []int{1, 10, 100} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			server := NewServer()
			defer server.Stop()
			service := &bufferedNotificationTestService{notifiers: make(chan *Notifier, 1)}
			if err := server.RegisterName("test", service); err != nil {
				t.Fatal(err)
			}
			client := DialInProc(server)
			defer client.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			notifications := make(chan *types.Log, count+1)
			sub, err := client.Subscribe(ctx, "test", notifications, "buffered", count)
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Unsubscribe()
			notifier := <-service.notifiers
			for i := 0; i < count; i++ {
				select {
				case log := <-notifications:
					if log.Index != uint(i) || len(log.Data) != 1 || log.Data[0] != byte(i) {
						t.Fatalf("notification %d: unexpected log %+v", i, log)
					}
				case err := <-sub.Err():
					t.Fatalf("subscription ended before notification %d: %v", i, err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			// Synchronize with activation after receiving the last buffered message.
			notifier.mu.Lock()
			buffered := cap(notifier.buffer)
			notifier.mu.Unlock()
			if buffered != 0 {
				t.Errorf("active subscription retains a notification buffer with capacity %d", buffered)
			}
			if err := notifier.Notify(notifier.sub.ID, &types.Log{Topics: []common.Hash{}, Index: uint(count)}); err != nil {
				t.Fatal(err)
			}
			select {
			case log := <-notifications:
				if log.Index != uint(count) {
					t.Fatalf("post-activation notification: unexpected index %d", log.Index)
				}
			case err := <-sub.Err():
				t.Fatalf("subscription ended after activation: %v", err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			sub.Unsubscribe()
			select {
			case <-notifier.sub.Err():
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
