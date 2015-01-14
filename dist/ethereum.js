require=(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
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
/** @file abi.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

// TODO: is these line is supposed to be here? 
if ("build" !== 'build') {/*
    var web3 = require('./web3'); // jshint ignore:line
*/}

// TODO: make these be actually accurate instead of falling back onto JS's doubles.
var hexToDec = function (hex) {
    return parseInt(hex, 16).toString();
};

var decToHex = function (dec) {
    return parseInt(dec).toString(16);
};

var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

var findMethodIndex = function (json, methodName) {
    return findIndex(json, function (method) {
        return method.name === methodName;
    });
};

var padLeft = function (string, chars) {
    return new Array(chars - string.length + 1).join("0") + string;
};

/// Setups input formatters for solidity types
/// @returns an array of input formatters 
var setupInputTypes = function () {
    
    var prefixedType = function (prefix) {
        return function (type, value) {
            return type.indexOf(prefix) === 0;
        };
    };

    var namedType = function (name, formatter) {
        return function (type, value) {
            return type === name;
        };
    };
    
    var formatInt = function (value) {
        var padding = 32 * 2;
        if (typeof value === 'number')
            value = value.toString(16);
        else if (value.indexOf('0x') === 0)
            value = value.substr(2);
        else if (typeof value === 'string')
            value = value.toHex(value);
        else
            value = (+value).toString(16);
        return padLeft(value, padding);
    };

    var formatString = function (value) {
        return web3.fromAscii(value, 32).substr(2);
    };

    var formatBool = function (value) {
        return '000000000000000000000000000000000000000000000000000000000000000' + (value ?  '1' : '0');
    };

    return [
        { type: prefixedType('uint'), format: formatInt },
        { type: prefixedType('int'), format: formatInt },
        { type: prefixedType('hash'), format: formatInt },
        { type: prefixedType('string'), format: formatString }, 
        { type: prefixedType('real'), format: formatInt },
        { type: prefixedType('ureal'), format: formatInt },
        { type: namedType('address') },
        { type: namedType('bool'), format: formatBool }
    ];
};

var inputTypes = setupInputTypes();

var toAbiInput = function (json, methodName, params) {
    var bytes = "";
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    var method = json[index];
    var padding = 32 * 2;

    for (var i = 0; i < method.inputs.length; i++) {
        var typeMatch = false;
        for (var j = 0; j < inputTypes.length && !typeMatch; j++) {
            typeMatch = inputTypes[j].type(method.inputs[i].type, params[i]);
        }
        if (!typeMatch) {
            console.error('input parser does not support type: ' + method.inputs[i].type);
        }

        var formatter = inputTypes[j - 1].format;
        bytes += (formatter ? formatter(params[i]) : params[i]);
    }
    return bytes;
};

/// Setups output formaters for solidity types
/// @returns an array of output formatters
var setupOutputTypes = function () {

    /// @param expected type prefix (string)
    /// @returns function which checks if type has matching prefix. if yes, returns true, otherwise false
    var prefixedType = function (prefix) {
        return function (type) {
            var expected = prefix;
            return type.indexOf(expected) === 0;
        };
    };

    /// @param expected type name (string)
    /// @returns function which checks if type is matching expected one. if yes, returns true, otherwise false
    var namedType = function (name) {
        return function (type) {
            return name === type;
        };
    };

    /// @returns input bytes formatted to int
    var formatInt = function (value) {
        return value.length <= 8 ? +parseInt(value, 16) : hexToDec(value);
    };

    /// @returns input bytes formatted to hex
    var formatHash = function (value) {
        return "0x" + value;
    };

    /// @returns input bytes formatted to bool
    var formatBool = function (value) {
        return value === '0000000000000000000000000000000000000000000000000000000000000001' ? true : false;
    };

    /// @returns input bytes formatted to ascii string
    var formatString = function (value) {
        return web3.toAscii(value);
    };

    return [
        { type: prefixedType('uint'), format: formatInt },
        { type: prefixedType('int'), format: formatInt },
        { type: prefixedType('hash'), format: formatHash },
        { type: prefixedType('string'), format: formatString },
        { type: prefixedType('real'), format: formatInt },
        { type: prefixedType('ureal'), format: formatInt },
        { type: namedType('address') },
        { type: namedType('bool'), format: formatBool }
    ];
};

var outputTypes = setupOutputTypes();

var fromAbiOutput = function (json, methodName, output) {
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    output = output.slice(2);

    var result = [];
    var method = json[index];
    var padding = 32 * 2;
    for (var i = 0; i < method.outputs.length; i++) {
        var typeMatch = false;
        for (var j = 0; j < outputTypes.length && !typeMatch; j++) {
            typeMatch = outputTypes[j].type(method.outputs[i].type);
        }

        if (!typeMatch) {
            // not found output parsing
            console.error('output parser does not support type: ' + method.outputs[i].type);
            continue;
        }
        var res = output.slice(0, padding);
        var formatter = outputTypes[j - 1].format;
        result.push(formatter ? formatter(res) : ("0x" + res));
        output = output.slice(padding);
    }

    return result;
};

/// @param json abi for contract
/// @returns input parser object for given json abi
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function () {
            var params = Array.prototype.slice.call(arguments);
            return toAbiInput(json, method.name, params);
        };
    });

    return parser;
};

/// @param json abi for contract
/// @returns output parser for given json abi
var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function (output) {
            return fromAbiOutput(json, method.name, output);
        };
    });

    return parser;
};

/// @param json abi for contract
/// @param method name for which we want to get method signature
/// @returns (promise) contract method signature for method with given name
var methodSignature = function (json, name) {
    var method = json[findMethodIndex(json, name)];
    var result = name + '(';
    var inputTypes = method.inputs.map(function (inp) {
        return inp.type;
    });
    result += inputTypes.join(',');
    result += ')';

    return web3.sha3(web3.fromAscii(result));
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    methodSignature: methodSignature
};


},{}],2:[function(require,module,exports){
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
if ("build" !== 'build') {/*
    var WebSocket = require('ws'); // jshint ignore:line
    var web3 = require('./web3'); // jshint ignore:line
*/}

/**
 * AutoProvider object prototype is implementing 'provider protocol'
 * Automatically tries to setup correct provider(Qt, WebSockets or HttpRpc)
 * First it checkes if we are ethereum browser (if navigator.qt object is available)
 * if yes, we are using QtProvider
 * if no, we check if it is possible to establish websockets connection with ethereum (ws://localhost:40404/eth is default)
 * if it's not possible, we are using httprpc provider (http://localhost:8080)
 * The constructor allows you to specify uris on which we are trying to connect over http or websockets
 * You can do that by passing objects with fields httrpc and websockets
 */
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

/// Sends message forward to the provider, that is being used
/// if provider is not yet set, enqueues the message
AutoProvider.prototype.send = function (payload) {
    if (this.provider) {
        this.provider.send(payload);
        return;
    }
    this.sendQueue.push(payload);
};

/// On incoming message sends the message to the provider that is currently being used
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

},{}],3:[function(require,module,exports){
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
/** @file contract.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

// TODO: is these line is supposed to be here? 
if ("build" !== 'build') {/*
    var web3 = require('./web3'); // jshint ignore:line
*/}

var abi = require('./abi');

/// method signature length in bytes
var ETH_METHOD_SIGNATURE_LENGTH = 4;

/**
 * This method should be called when we want to call / transact some solidity method from javascript
 * it returns an object which has same methods available as solidity contract description
 * usage example: 
 *
 * var abi = [{
 *      name: 'myMethod',
 *      inputs: [{ name: 'a', type: 'string' }],
 *      outputs: [{name 'd', type: 'string' }]
 * }];  // contract abi
 *
 * var myContract = web3.eth.contract('0x0123123121', abi); // creation of contract object
 *
 * myContract.myMethod('this is test string param for call').cal(); // myMethod call
 * myContract.myMethod('this is test string param for transact').transact() // myMethod transact
 *
 * @param address - address of the contract, which should be called
 * @param desc - abi json description of the contract, which is being created
 * @returns contract object
 */
var contract = function (address, desc) {
    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    var contract = {};

    desc.forEach(function (method) {
        contract[method.name] = function () {
            var params = Array.prototype.slice.call(arguments);
            var parsed = inputParser[method.name].apply(null, params);

            var onSuccess = function (result) {
                return outputParser[method.name](result);
            };

            return {
                call: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    return abi.methodSignature(desc, method.name).then(function (signature) {
                        extra.data = signature.slice(0, 2 + ETH_METHOD_SIGNATURE_LENGTH * 2) + parsed;
                        return web3.eth.call(extra).then(onSuccess);
                    });
                },
                transact: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    return abi.methodSignature(desc, method.name).then(function (signature) {
                        extra.data = signature.slice(0, 2 + ETH_METHOD_SIGNATURE_LENGTH * 2) + parsed;
                        return web3.eth.transact(extra).then(onSuccess);
                    });
                }
            };
        };
    });

    return contract;
};

module.exports = contract;


},{"./abi":1}],4:[function(require,module,exports){
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
if ("build" !== 'build') {/*
    var web3 = require('./web3'); // jshint ignore:line
*/}

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

},{}],5:[function(require,module,exports){
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
if ("build" !== 'build') {/*
    var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line
*/}

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


},{}],6:[function(require,module,exports){
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
if ("build" !== 'build') {/*
    var web3 = require('./web3'); // jshint ignore:line
*/}

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


},{}],7:[function(require,module,exports){
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
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

/**
 * QtProvider object prototype is implementing 'provider protocol'
 * Should be used inside ethereum browser. It's compatible with cpp and go clients.
 * It uses navigator.qt object to pass the messages to native bindings
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

/// Prototype object method
/// Should be called when we want to send single api request to native bindings
/// Asynchronous
/// Response will be received by navigator.qt.onmessage method and passed to handlers
/// @param payload is inner message object
QtProvider.prototype.send = function(payload) {
    navigator.qt.postMessage(JSON.stringify(payload));
};

/// Prototype object property
/// Should be used to set message handlers for this provider
Object.defineProperty(QtProvider.prototype, "onmessage", {
    set: function(handler) {
        this.handlers.push(handler);
    }
});

module.exports = QtProvider;

},{}],8:[function(require,module,exports){
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
/** @file web3.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var Filter = require('./filter');
var ProviderManager = require('./providermanager');

/// Recursively resolves all promises in given object and replaces the resolved values with promises
/// @param any object/array/promise/anything else..
/// @returns (resolves) object with replaced promises with their result 
function flattenPromise (obj) {
    if (obj instanceof Promise) {
        return Promise.resolve(obj);
    }

    if (obj instanceof Array) {
        return new Promise(function (resolve) {
            var promises = obj.map(function (o) {
                return flattenPromise(o);
            });

            return Promise.all(promises).then(function (res) {
                for (var i = 0; i < obj.length; i++) {
                    obj[i] = res[i];
                }
                resolve(obj);
            });
        });
    }

    if (obj instanceof Object) {
        return new Promise(function (resolve) {
            var keys = Object.keys(obj);
            var promises = keys.map(function (key) {
                return flattenPromise(obj[key]);
            });

            return Promise.all(promises).then(function (res) {
                for (var i = 0; i < keys.length; i++) {
                    obj[keys[i]] = res[i];
                }
                resolve(obj);
            });
        });
    }

    return Promise.resolve(obj);
}

/// @returns an array of objects describing web3 api methods
var web3Methods = function () {
    return [
    { name: 'sha3', call: 'web3_sha3' }
    ];
};

/// @returns an array of objects describing web3.eth api methods
var ethMethods = function () {
    var blockCall = function (args) {
        return typeof args[0] === "string" ? "eth_blockByHash" : "eth_blockByNumber";
    };

    var transactionCall = function (args) {
        return typeof args[0] === "string" ? 'eth_transactionByHash' : 'eth_transactionByNumber';
    };

    var uncleCall = function (args) {
        return typeof args[0] === "string" ? 'eth_uncleByHash' : 'eth_uncleByNumber';
    };

    var methods = [
    { name: 'balanceAt', call: 'eth_balanceAt' },
    { name: 'stateAt', call: 'eth_stateAt' },
    { name: 'storageAt', call: 'eth_storageAt' },
    { name: 'countAt', call: 'eth_countAt'},
    { name: 'codeAt', call: 'eth_codeAt' },
    { name: 'transact', call: 'eth_transact' },
    { name: 'call', call: 'eth_call' },
    { name: 'block', call: blockCall },
    { name: 'transaction', call: transactionCall },
    { name: 'uncle', call: uncleCall },
    { name: 'compilers', call: 'eth_compilers' },
    { name: 'lll', call: 'eth_lll' },
    { name: 'solidity', call: 'eth_solidity' },
    { name: 'serpent', call: 'eth_serpent' },
    { name: 'logs', call: 'eth_logs' }
    ];
    return methods;
};

/// @returns an array of objects describing web3.eth api properties
var ethProperties = function () {
    return [
    { name: 'coinbase', getter: 'eth_coinbase', setter: 'eth_setCoinbase' },
    { name: 'listening', getter: 'eth_listening', setter: 'eth_setListening' },
    { name: 'mining', getter: 'eth_mining', setter: 'eth_setMining' },
    { name: 'gasPrice', getter: 'eth_gasPrice' },
    { name: 'account', getter: 'eth_account' },
    { name: 'accounts', getter: 'eth_accounts' },
    { name: 'peerCount', getter: 'eth_peerCount' },
    { name: 'defaultBlock', getter: 'eth_defaultBlock', setter: 'eth_setDefaultBlock' },
    { name: 'number', getter: 'eth_number'}
    ];
};

/// @returns an array of objects describing web3.db api methods
var dbMethods = function () {
    return [
    { name: 'put', call: 'db_put' },
    { name: 'get', call: 'db_get' },
    { name: 'putString', call: 'db_putString' },
    { name: 'getString', call: 'db_getString' }
    ];
};

/// @returns an array of objects describing web3.shh api methods
var shhMethods = function () {
    return [
    { name: 'post', call: 'shh_post' },
    { name: 'newIdentity', call: 'shh_newIdentity' },
    { name: 'haveIdentity', call: 'shh_haveIdentity' },
    { name: 'newGroup', call: 'shh_newGroup' },
    { name: 'addToGroup', call: 'shh_addToGroup' }
    ];
};

/// @returns an array of objects describing web3.eth.watch api methods
var ethWatchMethods = function () {
    var newFilter = function (args) {
        return typeof args[0] === 'string' ? 'eth_newFilterString' : 'eth_newFilter';
    };

    return [
    { name: 'newFilter', call: newFilter },
    { name: 'uninstallFilter', call: 'eth_uninstallFilter' },
    { name: 'getMessages', call: 'eth_filterLogs' }
    ];
};

/// @returns an array of objects describing web3.shh.watch api methods
var shhWatchMethods = function () {
    return [
    { name: 'newFilter', call: 'shh_newFilter' },
    { name: 'uninstallFilter', call: 'shh_uninstallFilter' },
    { name: 'getMessage', call: 'shh_getMessages' }
    ];
};

/// creates methods in a given object based on method description on input
/// setups api calls for these methods
var setupMethods = function (obj, methods) {
    methods.forEach(function (method) {
        obj[method.name] = function () {
            return flattenPromise(Array.prototype.slice.call(arguments)).then(function (args) {
                var call = typeof method.call === "function" ? method.call(args) : method.call;
                return {call: call, args: args};
            }).then(function (request) {
                return new Promise(function (resolve, reject) {
                    web3.provider.send(request, function (err, result) {
                        if (!err) {
                            resolve(result);
                            return;
                        }
                        reject(err);
                    });
                });
            }).catch(function(err) {
                console.error(err);
            });
        };
    });
};

/// creates properties in a given object based on properties description on input
/// setups api calls for these properties
var setupProperties = function (obj, properties) {
    properties.forEach(function (property) {
        var proto = {};
        proto.get = function () {
            return new Promise(function(resolve, reject) {
                web3.provider.send({call: property.getter}, function(err, result) {
                    if (!err) {
                        resolve(result);
                        return;
                    }
                    reject(err);
                });
            });
        };
        if (property.setter) {
            proto.set = function (val) {
                return flattenPromise([val]).then(function (args) {
                    return new Promise(function (resolve) {
                        web3.provider.send({call: property.setter, args: args}, function (err, result) {
                            if (!err) {
                                resolve(result);
                                return;
                            }
                            reject(err);
                        });
                    });
                }).catch(function (err) {
                    console.error(err);
                });
            };
        }
        Object.defineProperty(obj, property.name, proto);
    });
};

// TODO: import from a dependency, don't duplicate.
var hexToDec = function (hex) {
    return parseInt(hex, 16).toString();
};

var decToHex = function (dec) {
    return parseInt(dec).toString(16);
};

/// setups web3 object, and it's in-browser executed methods
var web3 = {
    _callbacks: {},
    _events: {},
    providers: {},

    toHex: function(str) {
        var hex = "";
        for(var i = 0; i < str.length; i++) {
            var n = str.charCodeAt(i).toString(16);
            hex += n.length < 2 ? '0' + n : n;
        }

        return hex;
    },

    /// @returns ascii string representation of hex value prefixed with 0x
    toAscii: function(hex) {
        // Find termination
        var str = "";
        var i = 0, l = hex.length;
        if (hex.substring(0, 2) === '0x')
            i = 2;
        for(; i < l; i+=2) {
            var code = parseInt(hex.substr(i, 2), 16);
            if(code === 0) {
                break;
            }

            str += String.fromCharCode(code);
        }

        return str;
    },

    /// @returns hex representation (prefixed by 0x) of ascii string
    fromAscii: function(str, pad) {
        pad = pad === undefined ? 0 : pad;
        var hex = this.toHex(str);
        while(hex.length < pad*2)
            hex += "00";
        return "0x" + hex;
    },

    /// @returns decimal representaton of hex value prefixed by 0x
    toDecimal: function (val) {
        return hexToDec(val.substring(2));
    },

    /// @returns hex representation (prefixed by 0x) of decimal value
    fromDecimal: function (val) {
        return "0x" + decToHex(val);
    },

    /// used to transform value/string to eth string
    toEth: function(str) {
        var val = typeof str === "string" ? str.indexOf('0x') === 0 ? parseInt(str.substr(2), 16) : parseInt(str) : str;
        var unit = 0;
        var units = [ 'wei', 'Kwei', 'Mwei', 'Gwei', 'szabo', 'finney', 'ether', 'grand', 'Mether', 'Gether', 'Tether', 'Pether', 'Eether', 'Zether', 'Yether', 'Nether', 'Dether', 'Vether', 'Uether' ];
        while (val > 3000 && unit < units.length - 1)
        {
            val /= 1000;
            unit++;
        }
        var s = val.toString().length < val.toFixed(2).length ? val.toString() : val.toFixed(2);
        var replaceFunction = function($0, $1, $2) {
            return $1 + ',' + $2;
        };

        while (true) {
            var o = s;
            s = s.replace(/(\d)(\d\d\d[\.\,])/, replaceFunction);
            if (o === s)
                break;
        }
        return s + ' ' + units[unit];
    },

    /// eth object prototype
    eth: {
        watch: function (params) {
            return new Filter(params, ethWatch);
        }
    },

    /// db object prototype
    db: {},

    /// shh object prototype
    shh: {
        watch: function (params) {
            return new Filter(params, shhWatch);
        }
    },

    /// used by filter to register callback with given id
    on: function(event, id, cb) {
        if(web3._events[event] === undefined) {
            web3._events[event] = {};
        }

        web3._events[event][id] = cb;
        return this;
    },

    /// used by filter to unregister callback with given id
    off: function(event, id) {
        if(web3._events[event] !== undefined) {
            delete web3._events[event][id];
        }

        return this;
    },

    /// used to trigger callback registered by filter
    trigger: function(event, id, data) {
        var callbacks = web3._events[event];
        if (!callbacks || !callbacks[id]) {
            return;
        }
        var cb = callbacks[id];
        cb(data);
    },

    /// @returns true if provider is installed
    haveProvider: function() {
        return !!web3.provider.provider;
    }
};

/// setups all api methods
setupMethods(web3, web3Methods());
setupMethods(web3.eth, ethMethods());
setupProperties(web3.eth, ethProperties());
setupMethods(web3.db, dbMethods());
setupMethods(web3.shh, shhMethods());

var ethWatch = {
    changed: 'eth_changed'
};

setupMethods(ethWatch, ethWatchMethods());

var shhWatch = {
    changed: 'shh_changed'
};

setupMethods(shhWatch, shhWatchMethods());

web3.provider = new ProviderManager();

web3.setProvider = function(provider) {
    provider.onmessage = messageHandler;
    web3.provider.set(provider);
    web3.provider.sendQueued();
};

/// callled when there is new incoming message
function messageHandler(data) {
    if(data._event !== undefined) {
        web3.trigger(data._event, data._id, data.data);
        return;
    }

    if(data._id) {
        var cb = web3._callbacks[data._id];
        if (cb) {
            cb.call(this, data.error, data.data);
            delete web3._callbacks[data._id];
        }
    }
}

if (typeof(module) !== "undefined")
    module.exports = web3;

},{"./filter":4,"./providermanager":6}],9:[function(require,module,exports){
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
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

// TODO: is these line is supposed to be here? 
if ("build" !== 'build') {/*
    var WebSocket = require('ws'); // jshint ignore:line
*/}

/**
 * WebSocketProvider object prototype is implementing 'provider protocol'
 * Should be used when we want to connect to ethereum backend over websockets
 * It's compatible with go client
 * The constructor allows to specify host uri
 */
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
            self.handlers[i].call(self, JSON.parse(event.data), event);
        }
    };

    this.ws.onopen = function() {
        self.ready = true;

        for (var i = 0; i < self.queued.length; i++) {
            // Resend
            self.send(self.queued[i]);
        }
    };
};

/// Prototype object method
/// Should be called when we want to send single api request to server
/// Asynchronous, it's using websockets
/// Response for the call will be received by ws.onmessage
/// @param payload is inner message object
WebSocketProvider.prototype.send = function(payload) {
    if (this.ready) {
        var data = JSON.stringify(payload);

        this.ws.send(data);
    } else {
        this.queued.push(payload);
    }
};

/// Prototype object method
/// Should be called to add handlers
WebSocketProvider.prototype.onMessage = function(handler) {
    this.handlers.push(handler);
};

/// Prototype object method
/// Should be called to close websockets connection
WebSocketProvider.prototype.unload = function() {
    this.ws.close();
};

/// Prototype object property
/// Should be used to set message handlers for this provider
Object.defineProperty(WebSocketProvider.prototype, "onmessage", {
    set: function(provider) { this.onMessage(provider); }
});

if (typeof(module) !== "undefined")
    module.exports = WebSocketProvider;

},{}],"web3":[function(require,module,exports){
var web3 = require('./lib/web3');
web3.providers.WebSocketProvider = require('./lib/websocket');
web3.providers.HttpRpcProvider = require('./lib/httprpc');
web3.providers.QtProvider = require('./lib/qt');
web3.providers.AutoProvider = require('./lib/autoprovider');
web3.eth.contract = require('./lib/contract');

module.exports = web3;

},{"./lib/autoprovider":2,"./lib/contract":3,"./lib/httprpc":5,"./lib/qt":7,"./lib/web3":8,"./lib/websocket":9}]},{},["web3"])


//# sourceMappingURL=ethereum.js.map