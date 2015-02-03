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
/** @file jsonrpc.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var messageId = 1;

/// Should be called to valid json create payload object
/// @param method of jsonrpc call, required
/// @param params, an array of method params, optional
/// @returns valid jsonrpc payload object
var toPayload = function (method, params) {
    if (!method)
        console.error('jsonrpc method should be specified!');

    return {
        jsonrpc: '2.0',
        method: method,
        params: params || [],
        id: messageId++
    }; 
};

/// Should be called to check if jsonrpc response is valid
/// @returns true if response is valid, otherwise false 
var isValidResponse = function (response) {
    return !!response &&
        !response.error &&
        response.jsonrpc === '2.0' &&
        typeof response.id === 'number' &&
        (!!response.result || typeof response.result === 'boolean');
};

/// Should be called to create batch payload object
/// @param messages, an array of objects with method (required) and params (optional) fields
var toBatchPayload = function (messages) {
    return messages.map(function (message) {
        return toPayload(message.method, message.params);
    }); 
};

module.exports = {
    toPayload: toPayload,
    isValidResponse: isValidResponse,
    toBatchPayload: toBatchPayload
};


