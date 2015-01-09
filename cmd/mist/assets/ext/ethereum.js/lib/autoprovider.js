/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file autoprovider.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

/*
 * @brief if qt object is available, uses QtProvider,
 * if not tries to connect over websockets
 * if it fails, it uses HttpRpcProvider
 */

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var WebSocket = require('ws'); // jshint ignore:line
    var web3 = require('./web3'); // jshint ignore:line
}

var AutoProvider = function (userOptions) {
    if (web3.haveProvider()) {
        return;
    }

    // before we determine what provider we are, we have to cache request
    this.sendQueue = [];
    this.onmessageQueue = [];

    if (navigator.qt) {
        this.provider = new web3.providers.QtProvider();
        return;
    }

    userOptions = userOptions || {};
    var options = {
        httprpc: userOptions.httprpc || 'http://localhost:8080',
        websockets: userOptions.websockets || 'ws://localhost:40404/eth'
    };

    var self = this;
    var closeWithSuccess = function (success) {
        ws.close();
        if (success) {
            self.provider = new web3.providers.WebSocketProvider(options.websockets);
        } else {
            self.provider = new web3.providers.HttpRpcProvider(options.httprpc);
            self.poll = self.provider.poll.bind(self.provider);
        }
        self.sendQueue.forEach(function (payload) {
            self.provider(payload);
        });
        self.onmessageQueue.forEach(function (handler) {
            self.provider.onmessage = handler;
        });
    };

    var ws = new WebSocket(options.websockets);

    ws.onopen = function() {
        closeWithSuccess(true);
    };

    ws.onerror = function() {
        closeWithSuccess(false);
    };
};

AutoProvider.prototype.send = function (payload) {
    if (this.provider) {
        this.provider.send(payload);
        return;
    }
    this.sendQueue.push(payload);
};

Object.defineProperty(AutoProvider.prototype, 'onmessage', {
    set: function (handler) {
        if (this.provider) {
            this.provider.onmessage = handler;
            return;
        }
        this.onmessageQueue.push(handler);
    }
});

module.exports = AutoProvider;
