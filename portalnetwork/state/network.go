package state

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type StateNetwork struct {
	portalProtocol *discover.PortalProtocol
	closeCtx       context.Context
	closeFunc      context.CancelFunc
	log            log.Logger
}

func NewStateNetwork(portalProtocol *discover.PortalProtocol) *StateNetwork {
	ctx, cancel := context.WithCancel(context.Background())

	return &StateNetwork{
		portalProtocol: portalProtocol,
		closeCtx:       ctx,
		closeFunc:      cancel,
		log:            log.New("sub-protocol", "state"),
	}
}

func (h *StateNetwork) Start() error {
	err := h.portalProtocol.Start()
	if err != nil {
		return err
	}
	go h.processContentLoop(h.closeCtx)
	h.log.Debug("state network start successfully")
	return nil
}

func (h *StateNetwork) Stop() {
	h.closeFunc()
	h.portalProtocol.Stop()
}

func (h *StateNetwork) processContentLoop(ctx context.Context) {
	contentChan := h.portalProtocol.GetContent()
	for {
		select {
		case <-ctx.Done():
			return
		case contentElement := <-contentChan:
			err := h.validateContents(contentElement.ContentKeys, contentElement.Contents)
			if err != nil {
				h.log.Error("validate content failed", "err", err)
				continue
			}

			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
					var gossippedNum int
					gossippedNum, err = h.portalProtocol.Gossip(&contentElement.Node, contentElement.ContentKeys, contentElement.Contents)
					h.log.Trace("gossippedNum", "gossippedNum", gossippedNum)
					if err != nil {
						h.log.Error("gossip failed", "err", err)
						return
					}
				}
			}(ctx)
		}
	}
}

func (h *StateNetwork) validateContents(contentKeys [][]byte, contents [][]byte) error {
	// TODO
	panic("implement me")
}
