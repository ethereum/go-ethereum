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

if (process.env.NODE_ENV !== 'build') {
    var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line
}

var HttpRpcProvider = function (host) {
    this.handlers = [];
    this.host = host;
};

function formatJsonRpcObject(object) {
    return {
        jsonrpc: '2.0',
        method: object.call,
        params: object.args,
        id: object._id
    };
}

function formatJsonRpcMessage(message) {
    var object = JSON.parse(message);

    return {
        _id: object.id,
        data: object.result,
        error: object.error
    };
}

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

HttpRpcProvider.prototype.send = function (payload) {
    var self = this;
    this.sendRequest(payload, function (request) {
        self.handlers.forEach(function (handler) {
            handler.call(self, formatJsonRpcMessage(request.responseText));
        });
    });
};

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

Object.defineProperty(HttpRpcProvider.prototype, "onmessage", {
    set: function (handler) {
        this.handlers.push(handler);
    }
});

module.exports = HttpRpcProvider;
