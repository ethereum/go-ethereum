package downloader

// SetSyncProgress sets the current sync progress.
func (d *Downloader) SetSyncProgress(startingBlock, highestBlock uint64) {
	d.syncStatsLock.Lock()
	defer d.syncStatsLock.Unlock()

	d.syncStatsChainOrigin = startingBlock
	d.syncStatsChainHeight = highestBlock
}
