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

export type Message = {
    home?: HomeMessage,
    chain?: ChainMessage,
    txpool?: TxPoolMessage,
    network?: NetworkMessage,
    system?: SystemMessage,
    logs?: LogsMessage,
};

export type HomeMessage = {
    memory?: Chart,
    traffic?: Chart,
};

export type Chart = {
    history?: Array<ChartEntry>,
    new?: ChartEntry,
};

export type ChartEntry = {
    time: Date,
    value: number,
};

export type ChainMessage = {
    /* TODO (kurkomisi) */
};

export type TxPoolMessage = {
    /* TODO (kurkomisi) */
};

export type NetworkMessage = {
    /* TODO (kurkomisi) */
};

export type SystemMessage = {
    /* TODO (kurkomisi) */
};

export type LogsMessage = {
    log: string,
};
