// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

//go:build go1.5
// +build go1.5

package metrics

import "runtime"

func gcCPUFraction(memStats *runtime.MemStats) float64 {
	return memStats.GCCPUFraction
}
