// Copyright 2019 The go-ethereum Authors
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

package statediff

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// APIName is the namespace used for the state diffing service API
const APIName = "statediff"

// APIVersion is the version of the state diffing service API
const APIVersion = "0.0.1"

// PublicStateDiffAPI provides an RPC subscription interface
// that can be used to stream out state diffs as they
// are produced by a full node
type PublicStateDiffAPI struct {
	sds IService
}

// NewPublicStateDiffAPI creates an rpc subscription interface for the underlying statediff service
func NewPublicStateDiffAPI(sds IService) *PublicStateDiffAPI {
	return &PublicStateDiffAPI{
		sds: sds,
	}
}

// Stream is the public method to setup a subscription that fires off statediff service payloads as they are created
func (api *PublicStateDiffAPI) Stream(ctx context.Context) (*rpc.Subscription, error) {
	// ensure that the RPC connection supports subscriptions
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}

	// create subscription and start waiting for events
	rpcSub := notifier.CreateSubscription()

	go func() {
		// subscribe to events from the statediff service
		payloadChannel := make(chan Payload, chainEventChanSize)
		quitChan := make(chan bool, 1)
		api.sds.Subscribe(rpcSub.ID, payloadChannel, quitChan)
		// loop and await payloads and relay them to the subscriber with the notifier
		for {
			select {
			case payload := <-payloadChannel:
				if notifyErr := notifier.Notify(rpcSub.ID, payload); notifyErr != nil {
					log.Error("Failed to send state diff packet; error: " + notifyErr.Error())
					unSubErr := api.sds.Unsubscribe(rpcSub.ID)
					if unSubErr != nil {
						log.Error("Failed to unsubscribe from the state diff service; error: " + unSubErr.Error())
					}
					return
				}
			case err := <-rpcSub.Err():
				if err != nil {
					log.Error("State diff service rpcSub error: " + err.Error())
					err = api.sds.Unsubscribe(rpcSub.ID)
					if err != nil {
						log.Error("Failed to unsubscribe from the state diff service; error: " + err.Error())
					}
					return
				}
			case <-quitChan:
				// don't need to unsubscribe, service does so before sending the quit signal
				return
			}
		}
	}()

	return rpcSub, nil
}
