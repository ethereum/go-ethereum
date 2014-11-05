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
/** @file qt.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

var QtProvider = function() {
    this.handlers = [];

    var self = this;
    navigator.qt.onmessage = function (message) {
        self.handlers.forEach(function (handler) {
            handler.call(self, JSON.parse(message.data));
        });
    };
};

QtProvider.prototype.send = function(payload) {
    navigator.qt.postMessage(JSON.stringify(payload));
};

Object.defineProperty(QtProvider.prototype, "onmessage", {
    set: function(handler) {
        this.handlers.push(handler);
    }
});

module.exports = QtProvider;
