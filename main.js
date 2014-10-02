(function(window) {
    function isPromise(o) {
        return typeof o === "object" && o.then
    }

    var eth = {
        _callbacks: {},
        _events: {},
        prototype: Object(),

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
                var code = hex.charCodeAt(i)
                if(code == 0) {
                    break;
                }

                str += String.fromCharCode(parseInt(hex.substr(i, 2), 16));
            }

            return str;
        },

        fromAscii: function(str, pad) {
            if(pad === undefined) {
                pad = 32
            }

            var hex = this.toHex(str);

            while(hex.length < pad*2)
                hex += "00";

            return hex
        },

        block: function(numberOrHash) {
            return new Promise(function(resolve, reject) {
                var func;
                if(typeof numberOrHash == "string") {
                    func =  "getBlockByHash";
                } else {
                    func =  "getBlockByNumber";
                }

                eth.provider.send({call: func, args: [numberOrHash]}, function(block) {
                    if(block)
                        resolve(block);
                    else
                        reject("not found");

                });
            });
        },

        transact: function(params) {
            if(params === undefined) {
                params = {};
            }

            if(params.endowment !== undefined)
                params.value = params.endowment;
            if(params.code !== undefined)
                params.data = params.code;


            var promises = []
            if(isPromise(params.to)) {
                promises.push(params.to.then(function(_to) { params.to = _to; }));
            }
            if(isPromise(params.from)) {
                promises.push(params.from.then(function(_from) { params.from = _from; }));
            }

            if(typeof params.data !== "object" || isPromise(params.data)) {
                params.data = [params.data]
            }

            var data = params.data;
            for(var i = 0; i < params.data.length; i++) {
                if(isPromise(params.data[i])) {
                    var promise = params.data[i];
                    var _i = i;
                    promises.push(promise.then(function(_arg) { params.data[_i] = _arg; }));
                }
            }

            // Make sure everything is string
            var fields = ["value", "gas", "gasPrice"];
            for(var i = 0; i < fields.length; i++) {
                if(params[fields[i]] === undefined) {
                    params[fields[i]] = "";
                }
                params[fields[i]] = params[fields[i]].toString();
            }

            // Load promises then call the last "transact".
            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    params.data = params.data.join("");
                    eth.provider.send({call: "transact", args: [params]}, function(data) {
                        if(data[1])
                            reject(data[0]);
                        else
                            resolve(data[0]);
                    });
                });
            })
        },

        compile: function(code) {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "compile", args: [code]}, function(data) {
                    if(data[1])
                        reject(data[0]);
                    else
                        resolve(data[0]);
                });
            });
        },

        balanceAt: function(address) {
            var promises = [];

            if(isPromise(address)) {
                promises.push(address.then(function(_address) { address = _address; }));
            }

            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "getBalanceAt", args: [address]}, function(balance) {
                        resolve(balance);
                    });
                });
            });
        },

        countAt: function(address) {
            var promises = [];

            if(isPromise(address)) {
                promises.push(address.then(function(_address) { address = _address; }));
            }

            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "getCountAt", args: [address]}, function(count) {
                        resolve(count);
                    });
                });
            });
        },

        codeAt: function(address) {
            var promises = [];

            if(isPromise(address)) {
                promises.push(address.then(function(_address) { address = _address; }));
            }

            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "getCodeAt", args: [address]}, function(code) {
                        resolve(code);
                    });
                });
            });
        },

        storageAt: function(address, storageAddress) {
            var promises = [];

            if(isPromise(address)) {
                promises.push(address.then(function(_address) { address = _address; }));
            }

            if(isPromise(storageAddress)) {
                promises.push(storageAddress.then(function(_sa) { storageAddress = _sa; }));
            }

            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "getStorageAt", args: [address, storageAddress]}, function(entry) {
                        resolve(entry);
                    });
                });
            });
        },

        stateAt: function(address, storageAddress) {
            return this.storageAt(address, storageAddress);
        },

        call: function(params) {
            if(params === undefined) {
                params = {};
            }

            if(params.endowment !== undefined)
                params.value = params.endowment;
            if(params.code !== undefined)
                params.data = params.code;


            var promises = []
            if(isPromise(params.to)) {
                promises.push(params.to.then(function(_to) { params.to = _to; }));
            }
            if(isPromise(params.from)) {
                promises.push(params.from.then(function(_from) { params.from = _from; }));
            }

            if(isPromise(params.data)) {
                promises.push(params.data.then(function(_code) { params.data = _code; }));
            } else {
                if(typeof params.data === "object") {
                    data = "";
                    for(var i = 0; i < params.data.length; i++) {
                        data += params.data[i]
                    }
                } else {
                    data = params.data;
                }
            }

            // Make sure everything is string
            var fields = ["value", "gas", "gasPrice"];
            for(var i = 0; i < fields.length; i++) {
                if(params[fields[i]] === undefined) {
                    params[fields[i]] = "";
                }
                params[fields[i]] = params[fields[i]].toString();
            }

            // Load promises then call the last "transact".
            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "call", args: params}, function(data) {
                        if(data[1])
                            reject(data[0]);
                        else
                            resolve(data[0]);
                    });
                });
            })
        },

        watch: function(params) {
            return new Filter(params);
        },

        secretToAddress: function(key) {
            var promises = [];
            if(isPromise(key)) {
                promises.push(key.then(function(_key) { key = _key; }));
            }

            return Promise.all(promises).then(function() {
                return new Promise(function(resolve, reject) {
                    eth.provider.send({call: "getSecretToAddress", args: [key]}, function(address) {
                        resolve(address);
                    });
                });
            });
        },

        on: function(event, cb) {
            if(eth._events[event] === undefined) {
                eth._events[event] = [];
            }

            eth._events[event].push(cb);

            return this
        },

        off: function(event, cb) {
            if(eth._events[event] !== undefined) {
                var callbacks = eth._events[event];
                for(var i = 0; i < callbacks.length; i++) {
                    if(callbacks[i] === cb) {
                        delete callbacks[i];
                    }
                }
            }

            return this
        },

        trigger: function(event, data) {
            var callbacks = eth._events[event];
            if(callbacks !== undefined) {
                for(var i = 0; i < callbacks.length; i++) {
                    // Figure out whether the returned data was an array
                    // array means multiple return arguments (multiple params)
                    if(data instanceof Array) {
                        callbacks[i].apply(this, data);
                    } else {
                        callbacks[i].call(this, data);
                    }
                }
            }
        },
    };

    // Eth object properties
    Object.defineProperty(eth, "key", {
        get: function() {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getKey"}, function(k) {
                    resolve(k);
                });
            });
        },
    });

    Object.defineProperty(eth, "gasPrice", {
        get: function() {
            return "10000000000000"
        }
    });

    Object.defineProperty(eth, "coinbase", {
        get: function() {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getCoinBase"}, function(coinbase) {
                    resolve(coinbase);
                });
            });
        },
    });

    Object.defineProperty(eth, "listening", {
        get: function() {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getIsListening"}, function(listening) {
                    resolve(listening);
                });
            });
        },
    });


    Object.defineProperty(eth, "mining", {
        get: function() {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getIsMining"}, function(mining) {
                    resolve(mining);
                });
            });
        },
    });

    Object.defineProperty(eth, "peerCount", {
        get: function() {
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getPeerCount"}, function(peerCount) {
                    resolve(peerCount);
                });
            });
        },
    });

    var ProviderManager = function() {
        this.queued = [];
        this.ready = false;
        this.provider = undefined;
        this.seed = 1;
    };
    ProviderManager.prototype.send = function(data, cb) {
        data.seed = this.seed;
        if(cb) {
            eth._callbacks[data.seed] = cb;
        }

        if(data.args === undefined) {
            data.args = [];
        }

        this.seed++;

        if(this.provider !== undefined) {
            this.provider.send(data);
        } else {
            this.queued = data;
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

    eth.provider = new ProviderManager();

    eth.setProvider = function(provider) {
        eth.provider.set(provider);

        eth.provider.sendQueued();
    };

    var EthWebSocket = function(host) {
        // onmessage handlers
        this.handlers = [];
        // queue will be filled with messages if send is invoked before the ws is ready
        this.queued = [];
        this.ready = false;

        this.ws = new WebSocket(host);

        var self = this;
        this.ws.onmessage = function(event) {
            for(var i = 0; i < self.handlers.length; i++) {
                self.handlers[i].call(self, event.data, event)
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

    EthWebSocket.prototype.send = function(jsonData) {
        if(this.ready) {
            var data = JSON.stringify(jsonData);

            this.ws.send(data);
        } else {
            this.queued.push(jsonData);
        }
    };

    EthWebSocket.prototype.onMessage = function(handler) {
        this.handlers.push(handler);
    };

    EthWebSocket.prototype.unload = function() {
        this.ws.close();
    };
    eth.WebSocket = EthWebSocket;

    var filters = [];
    var Filter = function(options) {
        filters.push(this);

        this.callbacks = [];
        this.options = options;

        var call;
        if(options === "chain") {
            call = "newFilterString"
        } else if(typeof options === "object") {
            call = "newFilter"
        }

        var self = this; // Cheaper than binding
        this.promise = new Promise(function(resolve, reject) {
            eth.provider.send({call: call, args: [options]}, function(id) {
                self.id = id;

                resolve(id);
            });
        });
    };

    Filter.prototype.changed = function(callback) {
        var self = this;
        this.promise.then(function(id) {
            self.callbacks.push(callback);
        });
    };

    Filter.prototype.trigger = function(messages, id) {
        if(id == this.id) {
            for(var i = 0; i < this.callbacks.length; i++) {
                this.callbacks[i].call(this, messages);
            }
        }
    };

    Filter.prototype.uninstall = function() {
        this.promise.then(function(id) {
            eth.provider.send({call: "uninstallFilter", args:[id]});
        });
    };

    Filter.prototype.messages = function() {
        var self=this;
        return Promise.all([this.promise]).then(function() {
            var id = self.id
            return new Promise(function(resolve, reject) {
                eth.provider.send({call: "getMessages", args: [id]}, function(messages) {
                    resolve(messages);
                });
            });
        });
    };

    // Register to the messages callback. "messages" will be emitted when new messages
    // from the client have been created.
    eth.on("messages", function(messages, id) {
        for(var i = 0; i < filters.length; i++) {
            filters[i].trigger(messages, id);
        }
    });


    // Install default provider
    if(!eth.provider.installed()) {
        var sock = new eth.WebSocket("ws://localhost:40404/eth");
        sock.onMessage(function(ev) {
            var data = JSON.parse(ev)

            if(data._event !== undefined) {
                eth.trigger(data._event, data.data);
            } else {
                if(data.seed) {
                    var cb = eth._callbacks[data.seed];
                    if(cb) {
                        cb.call(this, data.data)

                        // Remove the "trigger" callback
                        delete eth._callbacks[ev.seed];
                    }
                }
            }
        });

        eth.setProvider(sock);
    }

    window.eth = eth;
})(this);

/*
   function eth.provider.send(data, cb) {
   data.seed = g_seed;
   if(cb) {
   eth._callbacks[data.seed] = cb;
   }

   if(data.args === undefined) {
   data.args = [];
   }

   g_seed++;

   window._messagingAdapter.call(this, data)
   }
   */
