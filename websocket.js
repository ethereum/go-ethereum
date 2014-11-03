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
/** @file websocket.js
 * @authors:
 * @date 2014
 */

(function() {
    var WebSocketProvider = function(host) {
        // onmessage handlers
        this.handlers = [];
        // queue will be filled with messages if send is invoked before the ws is ready
        this.queued = [];
        this.ready = false;

        this.ws = new WebSocket(host);

        var self = this;
        this.ws.onmessage = function(event) {
            for(var i = 0; i < self.handlers.length; i++) {
                self.handlers[i].call(self, JSON.parse(event.data), event)
            }
        };

        this.ws.onopen = function() {
            self.ready = true;

            for(var i = 0; i < self.queued.length; i++) {
                // Resend
                self.send(self.queued[i]);
            }
        };
    };
    WebSocketProvider.prototype.send = function(payload) {
        if(this.ready) {
            var data = JSON.stringify(payload);

            this.ws.send(data);
        } else {
            this.queued.push(payload);
        }
    };

    WebSocketProvider.prototype.onMessage = function(handler) {
        this.handlers.push(handler);
    };

    WebSocketProvider.prototype.unload = function() {
        this.ws.close();
    };
    Object.defineProperty(WebSocketProvider.prototype, "onmessage", {
        set: function(provider) { this.onMessage(provider); }
    });

    if(typeof(web3) !== "undefined" && web3.providers !== undefined) {
        web3.providers.WebSocketProvider = WebSocketProvider;
    }
})();
