package eth

import "container/list"

type PeerListEvent struct {
	Peers *list.List
}

type ChainSyncEvent struct {
	InSync bool
}
