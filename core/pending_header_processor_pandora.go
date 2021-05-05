package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
)

/*
* Generated: May - 05 - 2021
* This file is generated to support Lukso pandora module
* Purpose: In response of https://github.com/lukso-network/pandora-execution-engine/issues/57 we need to have a pending
* in memory database. Which will hold the headers when they are locally validated but not validated by orchestrator node.
* Insert Headers operation will halt until the header is validated by orchestrator.
 */

// PandoraPendingHeaderContainer will hold temporary headers in a in memory db.
type PandoraPendingHeaderContainer struct {
	headerContainer ethdb.Database	// in-memory database which will hold headers temporarily
	pndHeaderFeed 	event.Feed	// announce new arrival of pending header
}

// NewPandoraPendingHeaderContainer will return a fully initiated in-memory header container
func NewPandoraPendingHeaderContainer() *PandoraPendingHeaderContainer {
	return &PandoraPendingHeaderContainer{
		headerContainer: rawdb.NewMemoryDatabase(),
	}
}

// WriteHeaderBatch dumps a batch of header into header container
func (container *PandoraPendingHeaderContainer) WriteHeaderBatch (headers []*types.Header) {
	for _, header := range headers {
		container.WriteHeader(header)
	}
}

// WriteHeader dump a single header in the header container
func (container *PandoraPendingHeaderContainer) WriteHeader (header *types.Header)  {
	// write the header into db
	rawdb.WriteHeader(container.headerContainer, header)

	// make the header as the top of the container queue. It will help us to get the last pushed header instance
	rawdb.WriteHeadHeaderHash(container.headerContainer, header.Hash())
}

// ReadHeaderSince will receive a from header hash and return a batch of headers from that header.
func (container *PandoraPendingHeaderContainer) ReadHeaderSince (from common.Hash) []*types.Header {
	fromHeaderNumber := rawdb.ReadHeaderNumber(container.headerContainer, from)
	lastHeaderNumber := rawdb.ReadHeaderNumber(container.headerContainer, rawdb.ReadHeadHeaderHash(container.headerContainer))

	var headers []*types.Header
	for i := *fromHeaderNumber; i <= *lastHeaderNumber; i++ {

		header := container.readHeader(i)
		headers = append(headers, header)
	}
	return headers
}

// readHeader reads a single header which is given as the header number
func (container *PandoraPendingHeaderContainer) readHeader(headerNumber uint64) *types.Header {
	hashes := rawdb.ReadAllHashes(container.headerContainer, headerNumber)
	return rawdb.ReadHeader(container.headerContainer, hashes[0], headerNumber)
}
