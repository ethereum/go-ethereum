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

var web3 = require('./web3'); // jshint ignore:line

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
    this.polls = [];
    this.provider = undefined;
    this.id = 1;

    var self = this;
    var poll = function () {
        if (self.provider) {
            self.polls.forEach(function (data) {
                data.data._id = self.id;
                self.id++;
                var result = self.provider.send(data.data);
            
                result = JSON.parse(result);
                
                // dont call the callback if result is not an array, or empty one
                if (result.error || !(result.result instanceof Array) || result.result.length === 0) {
                    return;
                }

                data.callback(result.result);
            });
        }
        setTimeout(poll, 1000);
    };
    poll();
};

/// sends outgoing requests
ProviderManager.prototype.send = function(data) {

    data.args = data.args || [];
    data._id = this.id++;

    if (this.provider === undefined) {
        console.error('provider is not set');
        return null; 
    }

    //TODO: handle error here? 
    var result = this.provider.send(data);
    result = JSON.parse(result);

    if (result.error) {
        console.log(result.error);
        return null;
    }

    return result.result;
};

/// setups provider, which will be used for sending messages
ProviderManager.prototype.set = function(provider) {
    this.provider = provider;
};

/// this method is only used, when we do not have native qt bindings and have to do polling on our own
/// should be callled, on start watching for eth/shh changes
ProviderManager.prototype.startPolling = function (data, pollId, callback) {
    this.polls.push({data: data, id: pollId, callback: callback});
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

