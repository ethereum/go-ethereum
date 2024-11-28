// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package core implements the Ethereum consensus protocol.
package core

import (
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// guardedOnGenesisBlock wraps the tracer OnGenesisBlock method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnGenesisBlock(genesis *types.Block, alloc types.GenesisAlloc) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnGenesisBlock exited with panic, disabling tracer", "err", err)
			bc.logger.OnGenesisBlock = nil
		}
	}()
	bc.logger.OnGenesisBlock(genesis, alloc)
}

// guardedOnBlockchainInit wraps the tracer OnBlockchainInit method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnBlockchainInit(chainConfig *params.ChainConfig) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnBlockchainInit exited with panic, disabling tracer", "err", err)
			bc.logger.OnBlockchainInit = nil
		}
	}()
	bc.logger.OnBlockchainInit(chainConfig)
}

// guardedOnBlockStart wraps the tracer OnBlockStart method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnBlockStart(blockEvent tracing.BlockEvent) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnBlockStart exited with panic, disabling tracer", "err", err)
			bc.logger.OnBlockStart = nil
		}
	}()
	bc.logger.OnBlockStart(blockEvent)
}

// guardedOnBlockEnd wraps the tracer OnBlockEnd method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnBlockEnd(err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnBlockEnd exited with panic, disabling tracer", "err", err)
			bc.logger.OnBlockEnd = nil
		}
	}()
	bc.logger.OnBlockEnd(err)
}

// guardedOnClose wraps the tracer OnClose method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnClose() {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnClose exited with panic, disabling tracer", "err", err)
			bc.logger.OnClose = nil
		}
	}()
	bc.logger.OnClose()
}

// guardedOnSkippedBlock wraps the tracer OnSkippedBlock method with 'recover'. If
// a panic occurs, the hook will be disabled.
func (bc *BlockChain) guardedOnSkippedBlock(event tracing.BlockEvent) {
	defer func() {
		if err := recover(); err != nil {
			log.Warn("Tracer OnSkippedBlock exited with panic, disabling tracer", "err", err)
			bc.logger.OnSkippedBlock = nil
		}
	}()
	bc.logger.OnSkippedBlock(event)
}
