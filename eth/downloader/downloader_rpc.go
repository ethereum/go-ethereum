package downloader

import (
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

type DownloaderService struct {
	d *Downloader
}

func NewDownloaderService(d *Downloader) *DownloaderService {
	return &DownloaderService{d}
}

type Progress struct {
	Origin  uint64 `json:"startingBlock"`
	Current uint64 `json:"currentBlock"`
	Height  uint64 `json:"highestBlock"`
}

type SyncingResult struct {
	Syncing bool `json:"syncing"`
	Status  Progress `json:"status"`
}

func (s *DownloaderService) Syncing() (rpc.Subscription, error) {
	sub := s.d.mux.Subscribe(StartEvent{}, DoneEvent{}, FailedEvent{})

	output := func(event interface{}) interface{} {
		switch event.(type) {
		case StartEvent:
			result := &SyncingResult{Syncing: true}
			result.Status.Origin, result.Status.Current, result.Status.Height = s.d.Progress()
			return result
		case DoneEvent, FailedEvent:
			return false
		}
		return nil
	}

	return rpc.NewSubscriptionWithOutputFormat(sub, output), nil
}
