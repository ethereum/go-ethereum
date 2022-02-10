// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

//go:build !cgo || appengine || js
// +build !cgo appengine js

package metrics

func numCgoCall() int64 {
	return 0
}
