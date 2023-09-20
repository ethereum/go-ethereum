package heimdall

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

type (
	requestTypeKey struct{}
	requestType    string

	meter struct {
		request map[bool]metrics.Meter // map[isSuccessful]metrics.Meter
		timer   metrics.Timer
	}
)

const (
	stateSyncRequest          requestType = "state-sync"
	spanRequest               requestType = "span"
	checkpointRequest         requestType = "checkpoint"
	checkpointCountRequest    requestType = "checkpoint-count"
	milestoneRequest          requestType = "milestone"
	milestoneCountRequest     requestType = "milestone-count"
	milestoneNoAckRequest     requestType = "milestone-no-ack"
	milestoneLastNoAckRequest requestType = "milestone-last-no-ack"
	milestoneIDRequest        requestType = "milestone-id"
)

func withRequestType(ctx context.Context, reqType requestType) context.Context {
	return context.WithValue(ctx, requestTypeKey{}, reqType)
}

func getRequestType(ctx context.Context) (requestType, bool) {
	reqType, ok := ctx.Value(requestTypeKey{}).(requestType)
	return reqType, ok
}

var (
	requestMeters = map[requestType]meter{
		stateSyncRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/statesync/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/statesync/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/statesync/duration", nil),
		},
		spanRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/span/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/span/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/span/duration", nil),
		},
		checkpointRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/checkpoint/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/checkpoint/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/checkpoint/duration", nil),
		},
		checkpointCountRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/checkpointcount/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/checkpointcount/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/checkpointcount/duration", nil),
		},
		milestoneRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/milestone/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/milestone/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/milestone/duration", nil),
		},
		milestoneCountRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/milestonecount/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/milestonecount/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/milestonecount/duration", nil),
		},
		milestoneNoAckRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/milestonenoack/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/milestonenoack/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/milestonenoack/duration", nil),
		},
		milestoneLastNoAckRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/milestonelastnoack/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/milestonelastnoack/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/milestonelastnoack/duration", nil),
		},
		milestoneIDRequest: {
			request: map[bool]metrics.Meter{
				true:  metrics.NewRegisteredMeter("client/requests/milestoneid/valid", nil),
				false: metrics.NewRegisteredMeter("client/requests/milestoneid/invalid", nil),
			},
			timer: metrics.NewRegisteredTimer("client/requests/milestoneid/duration", nil),
		},
	}
)

func sendMetrics(ctx context.Context, start time.Time, isSuccessful bool) {
	reqType, ok := getRequestType(ctx)
	if !ok {
		return
	}

	meters, ok := requestMeters[reqType]
	if !ok {
		return
	}

	meters.request[isSuccessful].Mark(1)
	meters.timer.Update(time.Since(start))
}
