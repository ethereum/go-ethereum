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

var calcBitPadding = function (type, expected) {
    var value = type.slice(expected.length);
    if (value === "") {
        return 32;
    }
    return parseInt(value) / 8;
};

var calcBytePadding = function (type, expected) {
    var value = type.slice(expected.length);
    if (value === "") {
        return 32;
    }
    return parseInt(value);
};

var calcRealPadding = function (type, expected) {
    var value = type.slice(expected.length);
    if (value === "") {
        return 32;
    }
    var sizes = value.split('x');
    for (var padding = 0, i = 0; i < sizes; i++) {
        padding += (sizes[i] / 8);
    }
    return padding;
};

var setupInputTypes = function () {
    
    var prefixedType = function (prefix, calcPadding) {
        return function (type, value) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return false;
            }

            var padding = calcPadding(type, expected);
            if (typeof value === "number")
                value = value.toString(16);
            else if (typeof value === "string")
                value = web3.toHex(value); 
            else if (value.indexOf('0x') === 0)
                value = value.substr(2);
            else
                value = (+value).toString(16);
            return padLeft(value, padding * 2);
        };
    };

    var namedType = function (name, padding, formatter) {
        return function (type, value) {
            if (type !== name) {
                return false;
            }

            return padLeft(formatter ? formatter(value) : value, padding * 2);
        };
    };

    var formatBool = function (value) {
        return value ? '0x1' : '0x0';
    };

    return [
        prefixedType('uint', calcBitPadding),
        prefixedType('int', calcBitPadding),
        prefixedType('hash', calcBitPadding),
        prefixedType('string', calcBytePadding),
        prefixedType('real', calcRealPadding),
        prefixedType('ureal', calcRealPadding),
        namedType('address', 20),
        namedType('bool', 1, formatBool),
    ];
};

var inputTypes = setupInputTypes();

var toAbiInput = function (json, methodName, params) {
    var bytes = "";
    var index = findMethodIndex(json, methodName);

    if (index === -1) {
        return;
    }

    bytes = "0x" + padLeft(index.toString(16), 2);
    var method = json[index];

    for (var i = 0; i < method.inputs.length; i++) {
        var found = false;
        for (var j = 0; j < inputTypes.length && !found; j++) {
            found = inputTypes[j](method.inputs[i].type, params[i]);
        }
        if (!found) {
            console.error('unsupported json type: ' + method.inputs[i].type);
        }
        bytes += found;
    }
    return bytes;
};

var setupOutputTypes = function () {

    var prefixedType = function (prefix, calcPadding) {
        return function (type) {
            var expected = prefix;
            if (type.indexOf(expected) !== 0) {
                return -1;
            }

            var padding = calcPadding(type, expected);
            return padding * 2;
        };
    };

    var namedType = function (name, padding) {
        return function (type) {
            return name === type ? padding * 2 : -1;
        };
    };

    var formatInt = function (value) {
        return value.length <= 8 ? +parseInt(value, 16) : hexToDec(value);
    };

    var formatHash = function (value) {
        return "0x" + value;
    };

    var formatBool = function (value) {
        return value === '1' ? true : false;
    };

    var formatString = function (value) {
        return web3.toAscii(value);
    };

    return [
    { padding: prefixedType('uint', calcBitPadding), format: formatInt },
    { padding: prefixedType('int', calcBitPadding), format: formatInt },
    { padding: prefixedType('hash', calcBitPadding), format: formatHash },
    { padding: prefixedType('string', calcBytePadding), format: formatString },
    { padding: prefixedType('real', calcRealPadding), format: formatInt },
    { padding: prefixedType('ureal', calcRealPadding), format: formatInt },
    { padding: namedType('address', 20) },
    { padding: namedType('bool', 1), format: formatBool }
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
    for (var i = 0; i < method.outputs.length; i++) {
        var padding = -1;
        for (var j = 0; j < outputTypes.length && padding === -1; j++) {
            padding = outputTypes[j].padding(method.outputs[i].type);
        }

        if (padding === -1) {
            // not found output parsing
            continue;
        }
        var res = output.slice(0, padding);
        var formatter = outputTypes[j - 1].format;
        result.push(formatter ? formatter(res) : ("0x" + res));
        output = output.slice(padding);
    }

    return result;
};

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

var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        parser[method.name] = function (output) {
            return fromAbiOutput(json, method.name, output);
        };
    });

    return parser;
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser
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

AutoProvider.prototype.send = function (payload) {
    if (this.provider) {
        this.provider.send(payload);
        return;
    }
    this.sendQueue.push(payload);
};

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
    var web3 = require('./web3'); 
*/}

var abi = require('./abi');

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
                    extra.data = parsed;
                    return web3.eth.call(extra).then(onSuccess);
                },
                transact: function (extra) {
                    extra = extra || {};
                    extra.to = address;
                    extra.data = parsed;
                    return web3.eth.transact(extra).then(onSuccess);
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
/** @file qt.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
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
/** @file main.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

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

var web3Methods = function () {
    return [
    { name: 'sha3', call: 'web3_sha3' }
    ];
};

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

var dbMethods = function () {
    return [
    { name: 'put', call: 'db_put' },
    { name: 'get', call: 'db_get' },
    { name: 'putString', call: 'db_putString' },
    { name: 'getString', call: 'db_getString' }
    ];
};

var shhMethods = function () {
    return [
    { name: 'post', call: 'shh_post' },
    { name: 'newIdentity', call: 'shh_newIdentity' },
    { name: 'haveIdentity', call: 'shh_haveIdentity' },
    { name: 'newGroup', call: 'shh_newGroup' },
    { name: 'addToGroup', call: 'shh_addToGroup' }
    ];
};

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

var shhWatchMethods = function () {
    return [
    { name: 'newFilter', call: 'shh_newFilter' },
    { name: 'uninstallFilter', call: 'shh_uninstallFilter' },
    { name: 'getMessage', call: 'shh_getMessages' }
    ];
};

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

    toAscii: function(hex) {
        // Find termination
        var str = "";
        var i = 0, l = hex.length;
        if (hex.substring(0, 2) === '0x')
            i = 2;
        for(; i < l; i+=2) {
            var code = hex.charCodeAt(i);
            if(code === 0) {
                break;
            }

            str += String.fromCharCode(parseInt(hex.substr(i, 2), 16));
        }

        return str;
    },

    fromAscii: function(str, pad) {
        pad = pad === undefined ? 32 : pad;
        var hex = this.toHex(str);
        while(hex.length < pad*2)
            hex += "00";
        return "0x" + hex;
    },

    toDecimal: function (val) {
        return hexToDec(val.substring(2));
    },

    fromDecimal: function (val) {
        return "0x" + decToHex(val);
    },

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

    eth: {
        prototype: Object(), // jshint ignore:line
        watch: function (params) {
            return new Filter(params, ethWatch);
        }
    },

    db: {
        prototype: Object() // jshint ignore:line
    },

    shh: {
        prototype: Object(), // jshint ignore:line
        watch: function (params) {
            return new Filter(params, shhWatch);
        }
    },

    on: function(event, id, cb) {
        if(web3._events[event] === undefined) {
            web3._events[event] = {};
        }

        web3._events[event][id] = cb;
        return this;
    },

    off: function(event, id) {
        if(web3._events[event] !== undefined) {
            delete web3._events[event][id];
        }

        return this;
    },

    trigger: function(event, id, data) {
        var callbacks = web3._events[event];
        if (!callbacks || !callbacks[id]) {
            return;
        }
        var cb = callbacks[id];
        cb(data);
    }
};

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

ProviderManager.prototype.set = function(provider) {
    if(this.provider !== undefined && this.provider.unload !== undefined) {
        this.provider.unload();
    }

    this.provider = provider;
    this.ready = true;
};

ProviderManager.prototype.sendQueued = function() {
    for(var i = 0; this.queued.length; i++) {
        // Resend
        this.send(this.queued[i]);
    }
};

ProviderManager.prototype.installed = function() {
    return this.provider !== undefined;
};

ProviderManager.prototype.startPolling = function (data, pollId) {
    if (!this.provider || !this.provider.poll) {
        return;
    }
    this.polls.push({data: data, id: pollId});
};

ProviderManager.prototype.stopPolling = function (pollId) {
    for (var i = this.polls.length; i--;) {
        var poll = this.polls[i];
        if (poll.id === pollId) {
            this.polls.splice(i, 1);
        }
    }
};

web3.provider = new ProviderManager();

web3.setProvider = function(provider) {
    provider.onmessage = messageHandler;
    web3.provider.set(provider);
    web3.provider.sendQueued();
};

web3.haveProvider = function() {
    return !!web3.provider.provider;
};

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

Filter.prototype.arrived = function(callback) {
    this.changed(callback);
};

Filter.prototype.changed = function(callback) {
    var self = this;
    this.promise.then(function(id) {
        self.callbacks.push(callback);
    });
};

Filter.prototype.trigger = function(messages) {
    for(var i = 0; i < this.callbacks.length; i++) {
        this.callbacks[i].call(this, messages);
    }
};

Filter.prototype.uninstall = function() {
    var self = this;
    this.promise.then(function (id) {
        self.impl.uninstallFilter(id);
        web3.provider.stopPolling(id);
        web3.off(impl.changed, id);
    });
};

Filter.prototype.messages = function() {
    var self = this;
    return this.promise.then(function (id) {
        return self.impl.getMessages(id);
    });
};

Filter.prototype.logs = function () {
    return this.messages();
};

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

module.exports = web3;

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

module.exports = WebSocketProvider;

},{}],"web3":[function(require,module,exports){
var web3 = require('./lib/web3');
web3.providers.WebSocketProvider = require('./lib/websocket');
web3.providers.HttpRpcProvider = require('./lib/httprpc');
web3.providers.QtProvider = require('./lib/qt');
web3.providers.AutoProvider = require('./lib/autoprovider');
web3.contract = require('./lib/contract');

module.exports = web3;

},{"./lib/autoprovider":2,"./lib/contract":3,"./lib/httprpc":4,"./lib/qt":5,"./lib/web3":6,"./lib/websocket":7}]},{},["web3"])


//# sourceMappingURL=ethereum.js.map