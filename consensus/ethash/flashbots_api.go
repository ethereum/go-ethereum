package ethash

import "errors"

// FlashbotsAPI exposes Flashbots related methods for the RPC interface.
type FlashbotsAPI struct {
	ethash *Ethash
}

// GetWork returns a work package for external miner.
//
// The work package consists of 5 strings:
//   result[0] - 32 bytes hex encoded current block header pow-hash
//   result[1] - 32 bytes hex encoded seed hash used for DAG
//   result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//   result[3] - hex encoded block number
//   result[4] - hex encoded profit generated from this block
func (api *FlashbotsAPI) GetWork() ([5]string, error) {
	if api.ethash.remote == nil {
		return [5]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [5]string, 1)
		errc   = make(chan error, 1)
	)
	select {
	case api.ethash.remote.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.ethash.remote.exitCh:
		return [5]string{}, errEthashStopped
	}
	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [5]string{}, err
	}
}
