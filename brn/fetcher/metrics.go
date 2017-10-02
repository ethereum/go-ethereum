// Copyright 2015 The go-burnout Authors
// This file is part of the go-burnout library.
//
// The go-burnout library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-burnout library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-burnout library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/burnout/go-burnout/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("brn/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("brn/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("brn/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("brn/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("brn/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("brn/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("brn/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("brn/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("brn/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("brn/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("brn/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("brn/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("brn/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("brn/fetcher/filter/bodies/out")
)
