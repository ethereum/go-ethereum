// @flow

// Copyright 2017 The go-ethereum Authors
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

export type Content = {
	general: General,
	home: Home,
	chain: Chain,
	txpool: TxPool,
	network: Network,
	system: System,
	logs: Logs,
};

export type General = {
	version: ?string,
	commit: ?string,
};

export type Home = {
	activeMemory: ChartEntries,
	virtualMemory: ChartEntries,
	networkIngress: ChartEntries,
	networkEgress: ChartEntries,
	processCPU: ChartEntries,
	systemCPU: ChartEntries,
	diskRead: ChartEntries,
	diskWrite: ChartEntries,
};

export type ChartEntries = Array<ChartEntry>;

export type ChartEntry = {
	time: Date,
	value: number,
};

export type Chain = {
	/* TODO (kurkomisi) */
};

export type TxPool = {
	/* TODO (kurkomisi) */
};

export type Network = {
	/* TODO (kurkomisi) */
};

export type System = {
	/* TODO (kurkomisi) */
};

export type Logs = {
	log: Array<string>,
};
