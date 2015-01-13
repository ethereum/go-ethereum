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

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var web3 = require('./web3'); // jshint ignore:line
}

/// should be used when we want to watch something
/// it's using inner polling mechanism and is notified about changes
var Filter = function(options, impl) {
    this.impl = impl;
    this.callbacks = [];

    var self = this;
    this.promise = impl.newFilter(options);
    this.promise.then(function (id) {
        self.id = id;
        web3.on(impl.changed, id, self.trigger.bind(self));
        web3.provider.startPolling({call: impl.changed, args: [id]}, id);
    });
};

/// alias for changed*
Filter.prototype.arrived = function(callback) {
    this.changed(callback);
};

/// gets called when there is new eth/shh message
Filter.prototype.changed = function(callback) {
    var self = this;
    this.promise.then(function(id) {
        self.callbacks.push(callback);
    });
};

/// trigger calling new message from people
Filter.prototype.trigger = function(messages) {
    for(var i = 0; i < this.callbacks.length; i++) {
        this.callbacks[i].call(this, messages);
    }
};

/// should be called to uninstall current filter
Filter.prototype.uninstall = function() {
    var self = this;
    this.promise.then(function (id) {
        self.impl.uninstallFilter(id);
        web3.provider.stopPolling(id);
        web3.off(impl.changed, id);
    });
};

/// should be called to manually trigger getting latest messages from the client
Filter.prototype.messages = function() {
    var self = this;
    return this.promise.then(function (id) {
        return self.impl.getMessages(id);
    });
};

/// alias for messages
Filter.prototype.logs = function () {
    return this.messages();
};

module.exports = Filter;
