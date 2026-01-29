// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package web3ext contains XDPoS-specific web3 extensions.
package web3ext

// XDCJs contains the XDC-specific web3 JavaScript extensions
const XDCJs = `
web3._extend({
	property: 'xdc',
	methods: [
		new web3._extend.Method({
			name: 'getMasternodes',
			call: 'xdc_getMasternodes',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getMasternodesByNumber',
			call: 'xdc_getMasternodesByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getCandidates',
			call: 'xdc_getCandidates',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getCandidateOwner',
			call: 'xdc_getCandidateOwner',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getCandidateCap',
			call: 'xdc_getCandidateCap',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getBlockSignersByNumber',
			call: 'xdc_getBlockSignersByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getBlockSignersByHash',
			call: 'xdc_getBlockSignersByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getEpoch',
			call: 'xdc_getEpoch',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getCurrentEpoch',
			call: 'xdc_getCurrentEpoch',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getReward',
			call: 'xdc_getReward',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getPenalized',
			call: 'xdc_getPenalized',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'networkInformation',
			call: 'xdc_networkInformation',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getStakeBalance',
			call: 'xdc_getStakeBalance',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getVoterBalance',
			call: 'xdc_getVoterBalance',
			params: 3,
			inputFormatter: [null, null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getVoterCap',
			call: 'xdc_getVoterCap',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null, null]
		}),
		new web3._extend.Method({
			name: 'getVoters',
			call: 'xdc_getVoters',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getOwnerCount',
			call: 'xdc_getOwnerCount',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getCandidateStatus',
			call: 'xdc_getCandidateStatus',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getMaxCapacity',
			call: 'xdc_getMaxCapacity',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getMinCapacity',
			call: 'xdc_getMinCapacity',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getKYC',
			call: 'xdc_getKYC',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getLatestKYC',
			call: 'xdc_getLatestKYC',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getV2Block',
			call: 'xdc_getV2Block',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getMasternodeStatusByNumber',
			call: 'xdc_getMasternodeStatusByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getMissedRoundsInEpochByBlockNum',
			call: 'xdc_getMissedRoundsInEpochByBlockNum',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getRoundInfo',
			call: 'xdc_getRoundInfo',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getEpochInfo',
			call: 'xdc_getEpochInfo',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'epochSize',
			getter: 'xdc_epochSize'
		}),
		new web3._extend.Property({
			name: 'gapSize',
			getter: 'xdc_gapSize'
		}),
		new web3._extend.Property({
			name: 'masternodeCount',
			getter: 'xdc_masternodeCount'
		}),
	]
});
`

// XDCxJs contains the XDCx DEX specific web3 JavaScript extensions
const XDCxJs = `
web3._extend({
	property: 'XDCx',
	methods: [
		new web3._extend.Method({
			name: 'getOrderById',
			call: 'XDCx_getOrderById',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'getOrderCount',
			call: 'XDCx_getOrderCount',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBestBid',
			call: 'XDCx_getBestBid',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getBestAsk',
			call: 'XDCx_getBestAsk',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getPriceAtDepth',
			call: 'XDCx_getPriceAtDepth',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getBidsTree',
			call: 'XDCx_getBidsTree',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getAsksTree',
			call: 'XDCx_getAsksTree',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getOrderNonce',
			call: 'XDCx_getOrderNonce',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getTradingState',
			call: 'XDCx_getTradingState',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getLiquidationPrice',
			call: 'XDCx_getLiquidationPrice',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getLendingOrderCount',
			call: 'XDCx_getLendingOrderCount',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBorrowingOrderCount',
			call: 'XDCx_getBorrowingOrderCount',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getInvestingOrderCount',
			call: 'XDCx_getInvestingOrderCount',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getLendingTradeCount',
			call: 'XDCx_getLendingTradeCount',
			params: 1
		}),
	]
});
`

// XDCLendingJs contains lending-specific web3 JavaScript extensions
const XDCLendingJs = `
web3._extend({
	property: 'XDCLending',
	methods: [
		new web3._extend.Method({
			name: 'getLendingState',
			call: 'XDCLending_getLendingState',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getLendingOrderById',
			call: 'XDCLending_getLendingOrderById',
			params: 4
		}),
		new web3._extend.Method({
			name: 'getInvestingById',
			call: 'XDCLending_getInvestingById',
			params: 4
		}),
		new web3._extend.Method({
			name: 'getBorrowingById',
			call: 'XDCLending_getBorrowingById',
			params: 4
		}),
		new web3._extend.Method({
			name: 'getLendingTradeById',
			call: 'XDCLending_getLendingTradeById',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getTopBid',
			call: 'XDCLending_getTopBid',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getTopAsk',
			call: 'XDCLending_getTopAsk',
			params: 3
		}),
		new web3._extend.Method({
			name: 'getLiquidatedTrade',
			call: 'XDCLending_getLiquidatedTrade',
			params: 3
		}),
	]
});
`

// AllXDCModules returns all XDC-specific JavaScript modules
func AllXDCModules() string {
	return XDCJs + XDCxJs + XDCLendingJs
}
