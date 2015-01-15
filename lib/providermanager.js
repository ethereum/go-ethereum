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
/** @file providermanager.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var web3 = require('./web3'); // jshint ignore:line
}

/**
 * Provider manager object prototype
 * It's responsible for passing messages to providers
 * If no provider is set it's responsible for queuing requests
 * It's also responsible for polling the ethereum node for incoming messages
 * Default poll timeout is 12 seconds
 * If we are running ethereum.js inside ethereum browser, there are backend based tools responsible for polling,
 * and provider manager polling mechanism is not used
 */
var ProviderManager = function() {
    this.queued = [];
    this.polls = [];
    this.ready = false;
    this.provider = undefined;
    this.id = 1;

    var self = this;
    var poll = function () {
        if (self.provider && self.provider.poll) {
            self.polls.forEach(function (data) {
                data.data._id = self.id;
                self.id++;
                self.provider.poll(data.data, data.id);
            });
        }
        setTimeout(poll, 12000);
    };
    poll();
};

/// sends outgoing requests, if provider is not available, enqueue the request
ProviderManager.prototype.send = function(data, cb) {
    data._id = this.id;
    if (cb) {
        web3._callbacks[data._id] = cb;
    }

    data.args = data.args || [];
    this.id++;

    if(this.provider !== undefined) {
        this.provider.send(data);
    } else {
        console.warn("provider is not set");
        this.queued.push(data);
    }
};

/// setups provider, which will be used for sending messages
ProviderManager.prototype.set = function(provider) {
    if(this.provider !== undefined && this.provider.unload !== undefined) {
        this.provider.unload();
    }

    this.provider = provider;
    this.ready = true;
};

/// resends queued messages
ProviderManager.prototype.sendQueued = function() {
    for(var i = 0; this.queued.length; i++) {
        // Resend
        this.send(this.queued[i]);
    }
};

/// @returns true if the provider i properly set
ProviderManager.prototype.installed = function() {
    return this.provider !== undefined;
};

/// this method is only used, when we do not have native qt bindings and have to do polling on our own
/// should be callled, on start watching for eth/shh changes
ProviderManager.prototype.startPolling = function (data, pollId) {
    if (!this.provider || !this.provider.poll) {
        return;
    }
    this.polls.push({data: data, id: pollId});
};

/// should be called to stop polling for certain watch changes
ProviderManager.prototype.stopPolling = function (pollId) {
    for (var i = this.polls.length; i--;) {
        var poll = this.polls[i];
        if (poll.id === pollId) {
            this.polls.splice(i, 1);
        }
    }
};

module.exports = ProviderManager;

