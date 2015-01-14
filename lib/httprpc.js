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
/** @file httprpc.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

// TODO: is these line is supposed to be here? 
if (process.env.NODE_ENV !== 'build') {
    var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line
}

/**
 * HttpRpcProvider object prototype is implementing 'provider protocol'
 * Should be used when we want to connect to ethereum backend over http && jsonrpc
 * It's compatible with cpp client
 * The contructor allows to specify host uri
 * This provider is using in-browser polling mechanism
 */
var HttpRpcProvider = function (host) {
    this.handlers = [];
    this.host = host;
};

/// Transforms inner message to proper jsonrpc object
/// @param inner message object
/// @returns jsonrpc object
function formatJsonRpcObject(object) {
    return {
        jsonrpc: '2.0',
        method: object.call,
        params: object.args,
        id: object._id
    };
}

/// Transforms jsonrpc object to inner message
/// @param incoming jsonrpc message 
/// @returns inner message object
function formatJsonRpcMessage(message) {
    var object = JSON.parse(message);

    return {
        _id: object.id,
        data: object.result,
        error: object.error
    };
}

/// Prototype object method 
/// Asynchronously sends request to server
/// @param payload is inner message object
/// @param cb is callback which is being called when response is comes back
HttpRpcProvider.prototype.sendRequest = function (payload, cb) {
    var data = formatJsonRpcObject(payload);

    var request = new XMLHttpRequest();
    request.open("POST", this.host, true);
    request.send(JSON.stringify(data));
    request.onreadystatechange = function () {
        if (request.readyState === 4 && cb) {
            cb(request);
        }
    };
};

/// Prototype object method
/// Should be called when we want to send single api request to server
/// Asynchronous
/// On response it passes message to handlers
/// @param payload is inner message object
HttpRpcProvider.prototype.send = function (payload) {
    var self = this;
    this.sendRequest(payload, function (request) {
        self.handlers.forEach(function (handler) {
            handler.call(self, formatJsonRpcMessage(request.responseText));
        });
    });
};

/// Prototype object method
/// Should be called only for polling requests
/// Asynchronous
/// On response it passege message to handlers, but only if message's result is true or not empty array
/// Otherwise response is being silently ignored
/// @param payload is inner message object
/// @id is id of poll that we are calling
HttpRpcProvider.prototype.poll = function (payload, id) {
    var self = this;
    this.sendRequest(payload, function (request) {
        var parsed = JSON.parse(request.responseText);
        if (parsed.error || (parsed.result instanceof Array ? parsed.result.length === 0 : !parsed.result)) {
            return;
        }
        self.handlers.forEach(function (handler) {
            handler.call(self, {_event: payload.call, _id: id, data: parsed.result});
        });
    });
};

/// Prototype object property
/// Should be used to set message handlers for this provider
Object.defineProperty(HttpRpcProvider.prototype, "onmessage", {
    set: function (handler) {
        this.handlers.push(handler);
    }
});

module.exports = HttpRpcProvider;

