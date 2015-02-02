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
/** @file filter.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var web3 = require('./web3'); // jshint ignore:line

/// should be used when we want to watch something
/// it's using inner polling mechanism and is notified about changes
/// TODO: change 'options' name cause it may be not the best matching one, since we have events
var Filter = function(options, impl) {

    if (typeof options !== "string") {

        // topics property is deprecated, warn about it!
        if (options.topics) {
            console.warn('"topics" is deprecated, use "topic" instead');
        }

        // evaluate lazy properties
        options = {
            to: options.to,
            topic: options.topic,
            earliest: options.earliest,
            latest: options.latest,
            max: options.max,
            skip: options.skip,
            address: options.address
        };

    }
    
    this.impl = impl;
    this.callbacks = [];

    this.id = impl.newFilter(options);
    web3.provider.startPolling({call: impl.changed, args: [this.id]}, this.id, this.trigger.bind(this));
};

/// alias for changed*
Filter.prototype.arrived = function(callback) {
    this.changed(callback);
};
Filter.prototype.happened = function(callback) {
    this.changed(callback);
};

/// gets called when there is new eth/shh message
Filter.prototype.changed = function(callback) {
    this.callbacks.push(callback);
};

/// trigger calling new message from people
Filter.prototype.trigger = function(messages) {
    for (var i = 0; i < this.callbacks.length; i++) {
        for (var j = 0; j < messages.length; j++) {
            this.callbacks[i].call(this, messages[j]);
        }
    }
};

/// should be called to uninstall current filter
Filter.prototype.uninstall = function() {
    this.impl.uninstallFilter(this.id);
    web3.provider.stopPolling(this.id);
};

/// should be called to manually trigger getting latest messages from the client
Filter.prototype.messages = function() {
    return this.impl.getMessages(this.id);
};

/// alias for messages
Filter.prototype.logs = function () {
    return this.messages();
};

module.exports = Filter;
