// Copyright 2016 The Go-etacoin Authors
// This file is part of Go-etacoin.
//
// Go-etacoin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Go-etacoin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Go-etacoin.  If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("eth/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("eth/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("eth/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("eth/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("eth/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("eth/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("eth/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("eth/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("eth/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("eth/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("eth/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("eth/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("eth/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("eth/fetcher/filter/bodies/out")
)
