package bigcache

import "time"

type clock interface {
	epoch() int64
}

type systemClock struct {
}

func (c systemClock) epoch() int64 {
	return time.Now().Unix()
}
