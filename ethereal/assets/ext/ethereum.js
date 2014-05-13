// Main Ethereum library
window.eth = {
	prototype: Object(),

	// Retrieve block
	//
	// Either supply a number or a string. Type is determent for the lookup method
	// string - Retrieves the block by looking up the hash
	// number - Retrieves the block by looking up the block number
        getBlock: function(numberOrHash, cb) {
                var func;
                if(typeof numberOrHash == "string") {
                        func =  "getBlockByHash";
                } else {
                        func =  "getBlockByNumber";
                }
                postData({call: func, args: [numberOrHash]}, cb);
        },

	// Create transaction
	//
	// Transact between two state objects
	transact: function(sec, recipient, value, gas, gasPrice, data, cb) {
		postData({call: "transact", args: [sec, recipient, value, gas, gasPrice, data]}, cb);
	},

	create: function(sec, value, gas, gasPrice, init, body, cb) {
		postData({call: "create", args: [sec, value, gas, gasPrice, init, body]}, cb);
	},

	getStorageAt: function(address, storageAddress, cb) {
		postData({call: "getStorage", args: [address, storageAddress]}, cb);
	},

	getKey: function(cb) {
		postData({call: "getKey"}, cb);
	},

	getTxCountAt: function(address, cb) {
		postData({call: "getTxCountAt", args: [address]}, cb);
	},
	getIsMining: function(cb){
		postData({call: "getIsMining"}, cb)
	},
	getIsListening: function(cb){
		postData({call: "getIsListening"}, cb)
	},
	getCoinBase: function(cb){
		postData({call: "getCoinBase"}, cb);
	},
	getPeerCount: function(cb){
		postData({call: "getPeerCount"}, cb);
	},
	getBalanceAt: function(address, cb) {
		postData({call: "getBalance", args: [address]}, cb);
	},

	getSecretToAddress: function(sec, cb) {
		postData({call: "getSecretToAddress", args: [sec]}, cb);
	},

	watch: function(address, storageAddrOrCb, cb) {
		var ev;
		if(cb === undefined) {
			cb = storageAddrOrCb;
			storageAddrOrCb = "";
			ev = "object:"+address;
		} else {
			ev = "storage:"+address+":"+storageAddrOrCb;
		}

		eth.on(ev, cb)

		postData({call: "watch", args: [address, storageAddrOrCb]});
	},

	disconnect: function(address, storageAddrOrCb, cb) {
		var ev;
		if(cb === undefined) {
			cb = storageAddrOrCb;
			storageAddrOrCb = "";
			ev = "object:"+address;
		} else {
			ev = "storage:"+address+":"+storageAddrOrCb;
		}

		eth.off(ev, cb)

		postData({call: "disconnect", args: [address, storageAddrOrCb]});
	},

	set: function(props) {
		postData({call: "set", args: props});
	},

	on: function(event, cb) {
		if(eth._onCallbacks[event] === undefined) {
			eth._onCallbacks[event] = [];
		}

		eth._onCallbacks[event].push(cb);

		return this
	},

	off: function(event, cb) {
		if(eth._onCallbacks[event] !== undefined) {
			var callbacks = eth._onCallbacks[event];
			for(var i = 0; i < callbacks.length; i++) {
				if(callbacks[i] === cb) {
					delete callbacks[i];
				}
			}
		}

		return this
	},

	trigger: function(event, data) {
		var callbacks = eth._onCallbacks[event];
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


}
window.eth._callbacks = {}
window.eth._onCallbacks = {}

