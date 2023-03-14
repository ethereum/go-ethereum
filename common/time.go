package common

import "time"

const TimeMilliseconds = "15:04:05.000"

func NowMilliseconds() string {
	return time.Now().Format(TimeMilliseconds)
}
