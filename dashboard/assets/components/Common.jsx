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

// isNullOrUndefined returns true if the given variable is null or undefined.
export const isNullOrUndefined = variable => variable === null || typeof variable === 'undefined';

export const LIMIT = {
    memory:  200, // Maximum number of memory data samples.
    traffic: 200, // Maximum number of traffic data samples.
    log:     200, // Maximum number of logs.
};
// The sidebar menu and the main content are rendered based on these elements.
export const TAGS = (() => {
    const T = {
        home:         { title: "Home", },
        chain:        { title: "Chain", },
        transactions: { title: "Transactions", },
        network:      { title: "Network", },
        system:       { title: "System", },
        logs:         { title: "Logs", },
    };
    // Using the key is circumstantial in some cases, so it is better to insert it also as a value.
    // This way the mistyping is prevented.
    for(let key in T) {
        T[key]['id'] = key;
    }
    return T;
})();

export const DATA_KEYS = (() => {
    const DK = {};
    ["memory", "traffic", "logs"].map(key => {
       DK[key] = key;
    });
    return DK;
})();

// Temporary - taken from Material-UI
export const DRAWER_WIDTH = 240;
