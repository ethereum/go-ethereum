package downloader

type DoneEvent struct{}
type StartEvent struct{}
type FailedEvent struct{ Err error }
