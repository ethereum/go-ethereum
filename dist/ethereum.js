require=(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){

/**
 * Module dependencies.
 */

var global = (function() { return this; })(); // jshint ignore:line

/**
 * XMLHttpRequest constructor.
 */

var XMLHttpRequest = window.XMLHttpRequest; // jshint ignore:line

/**
 * Module exports.
 */

module.exports.XMLHttpRequest = XMLHttpRequest ? xhr : null;

/**
 * XMLHttpRequest constructor.
 *
 * @param {Object) opts (optional)
 * @api public
 */

function xhr(obj) {
  var instance;

  instance = new XMLHttpRequest(obj);

  return instance;
}

if (XMLHttpRequest) xhr.prototype = XMLHttpRequest.prototype;

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
/** @file httprpc.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */


var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line


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
            data: object.result
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
            if (parsed.result instanceof Array ? parsed.result.length === 0 : !parsed.result) {
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

},{"xmlhttprequest":1}],3:[function(require,module,exports){
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
* @date 2014
*/


    function isPromise(o) {
        return o instanceof Promise;
    }

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

    var ethMethods = function () {
        var blockCall = function (args) {
            return typeof args[0] === "string" ? "blockByHash" : "blockByNumber";
        };

        var transactionCall = function (args) {
            return typeof args[0] === "string" ? 'transactionByHash' : 'transactionByNumber';
        };

        var uncleCall = function (args) {
            return typeof args[0] === "string" ? 'uncleByHash' : 'uncleByNumber';
        };

        var methods = [
        { name: 'balanceAt', call: 'balanceAt' },
        { name: 'stateAt', call: 'stateAt' },
        { name: 'countAt', call: 'countAt'},
        { name: 'codeAt', call: 'codeAt' },
        { name: 'transact', call: 'transact' },
        { name: 'call', call: 'call' },
        { name: 'block', call: blockCall },
        { name: 'transaction', call: transactionCall },
        { name: 'uncle', call: uncleCall },
        { name: 'compile', call: 'compile' }
        ];
        return methods;
    };

    var ethProperties = function () {
        return [
        { name: 'coinbase', getter: 'coinbase', setter: 'setCoinbase' },
        { name: 'listening', getter: 'listening', setter: 'setListening' },
        { name: 'mining', getter: 'mining', setter: 'setMining' },
        { name: 'gasPrice', getter: 'gasPrice' },
        { name: 'account', getter: 'account' },
        { name: 'accounts', getter: 'accounts' },
        { name: 'peerCount', getter: 'peerCount' },
        { name: 'defaultBlock', getter: 'defaultBlock', setter: 'setDefaultBlock' },
        { name: 'number', getter: 'number'}
        ];
    };

    var dbMethods = function () {
        return [
        { name: 'put', call: 'put' },
        { name: 'get', call: 'get' },
        { name: 'putString', call: 'putString' },
        { name: 'getString', call: 'getString' }
        ];
    };

    var shhMethods = function () {
        return [
        { name: 'post', call: 'post' },
        { name: 'newIdentity', call: 'newIdentity' },
        { name: 'haveIdentity', call: 'haveIdentity' },
        { name: 'newGroup', call: 'newGroup' },
        { name: 'addToGroup', call: 'addToGroup' }
        ];
    };

    var ethWatchMethods = function () {
        var newFilter = function (args) {
            return typeof args[0] === 'string' ? 'newFilterString' : 'newFilter';
        };

        return [
        { name: 'newFilter', call: newFilter },
        { name: 'uninstallFilter', call: 'uninstallFilter' },
        { name: 'getMessages', call: 'getMessages' }
        ];
    };

    var shhWatchMethods = function () {
        return [
        { name: 'newFilter', call: 'shhNewFilter' },
        { name: 'uninstallFilter', call: 'shhUninstallFilter' },
        { name: 'getMessage', call: 'shhGetMessages' }
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
                        web3.provider.send(request, function (result) {
                            if (result || typeof result === "boolean") {
                                resolve(result);
                                return;
                            }
                            reject(result);
                        });
                    });
                }).catch(function( err) {
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
                    web3.provider.send({call: property.getter}, function(result) {
                        resolve(result);
                    });
                });
            };
            if (property.setter) {
                proto.set = function (val) {
                    return flattenPromise([val]).then(function (args) {
                        return new Promise(function (resolve) {
                            web3.provider.send({call: property.setter, args: args}, function (result) {
                                if (result) {
                                    resolve(result);
                                } else {
                                    reject(result);
                                }
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
            for(; i < l; i+=2) {
                var code = hex.charCodeAt(i);
                if(code === 0) {
                    break;
                }

                str += String.fromCharCode(parseInt(hex.substr(i, 2), 16));
            }

            return str;
        },

        toDecimal: function (val) {
            return parseInt(val, 16);
        },

        fromAscii: function(str, pad) {
            pad = pad === undefined ? 32 : pad;
            var hex = this.toHex(str);
            while(hex.length < pad*2)
                hex += "00";
            return hex;
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

    var eth = web3.eth;
    setupMethods(eth, ethMethods());
    setupProperties(eth, ethProperties());
    setupMethods(web3.db, dbMethods());
    setupMethods(web3.shh, shhMethods());

    var ethWatch = {
        changed: 'changed'
    };
    setupMethods(ethWatch, ethWatchMethods());
    var shhWatch = {
        changed: 'shhChanged'
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

    function messageHandler(data) {
        if(data._event !== undefined) {
            web3.trigger(data._event, data._id, data.data);
            return;
        }

        if(data._id) {
            var cb = web3._callbacks[data._id];
            if (cb) {
                cb.call(this, data.data);
                delete web3._callbacks[data._id];
            }
        }
    }


module.exports = web3;

},{}],4:[function(require,module,exports){
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
/** @file websocket.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

var WebSocket = require('ws'); // jshint ignore:line


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

},{"ws":6}],6:[function(require,module,exports){

/**
 * Module dependencies.
 */

var global = (function() { return this; })();

/**
 * WebSocket constructor.
 */

var WebSocket = global.WebSocket || global.MozWebSocket;

/**
 * Module exports.
 */

module.exports = WebSocket ? ws : null;

/**
 * WebSocket constructor.
 *
 * The third `opts` options object gets ignored in web browsers, since it's
 * non-standard, and throws a TypeError if passed to the constructor.
 * See: https://github.com/einaros/ws/issues/227
 *
 * @param {String} uri
 * @param {Array} protocols (optional)
 * @param {Object) opts (optional)
 * @api public
 */

function ws(uri, protocols, opts) {
  var instance;
  if (protocols) {
    instance = new WebSocket(uri, protocols);
  } else {
    instance = new WebSocket(uri);
  }
  return instance;
}

if (WebSocket) ws.prototype = WebSocket.prototype;

},{}],"web3":[function(require,module,exports){
var web3 = require('./lib/main');
web3.providers.WebSocketProvider = require('./lib/websocket');
web3.providers.HttpRpcProvider = require('./lib/httprpc');
web3.providers.QtProvider = require('./lib/qt');

module.exports = web3;
},{"./lib/httprpc":2,"./lib/main":3,"./lib/qt":4,"./lib/websocket":5}]},{},[]);
