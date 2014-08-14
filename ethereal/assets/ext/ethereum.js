// Main Ethereum library
window.eth = {
	prototype: Object(),

	mutan: function(code) {
	},

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
	transact: function(params, cb) {
		if(params === undefined) {
			params = {};
		}

		if(params.endowment !== undefined)
			params.value = params.endowment;
		if(params.code !== undefined)
			params.data = params.code;

		// Make sure everything is string
		var fields = ["to", "from", "value", "gas", "gasPrice"];
		for(var i = 0; i < fields.length; i++) {
			if(params[fields[i]] === undefined) {
				params[fields[i]] = "";
			}
			params[fields[i]] = params[fields[i]].toString();
		}

		var data;
		if(typeof params.data === "object") {
			data = "";
			for(var i = 0; i < params.data.length; i++) {
				data += params.data[i]
			}
		} else {
			data = params.data;
		}

		postData({call: "transact", args: [params.from, params.to, params.value, params.gas, params.gasPrice, "0x"+data]}, cb);
	},

	getMessages: function(filter, cb) {
		postData({call: "messages", args: [filter]}, cb);
	},

	getStorageAt: function(address, storageAddress, cb) {
		postData({call: "getStorage", args: [address, storageAddress]}, cb);
	},

	getStateKeyVals: function(address, cb){
		postData({call: "getStateKeyVals", args: [address]}, cb);
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
	getTransactionsFor: function(address, cb) {
		postData({call: "getTransactionsFor", args: [address]}, cb);
	},

	getSecretToAddress: function(sec, cb) {
		postData({call: "getSecretToAddress", args: [sec]}, cb);
	},

	/*
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
	*/

       watch: function(options) {
	       var filter = new Filter(options);
	       filter.number = newWatchNum().toString()

	       postData({call: "watch", args: [options, filter.number]})

	       return filter;
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

var Filter = function(options) {
	this.options = options;
};

Filter.prototype.changed = function(callback) {
	eth.on("watched:"+this.number, callback)
}

Filter.prototype.getMessages = function(cb) {
	return eth.getMessages(this.options, cb)
}

var watchNum = 0;
function newWatchNum() {
	return watchNum++;
}

function postData(data, cb) {
	data._seed = Math.floor(Math.random() * 1000000)
	if(cb) {
		eth._callbacks[data._seed] = cb;
	}

	if(data.args === undefined) {
		data.args = [];
	}

	navigator.qt.postMessage(JSON.stringify(data));
}

navigator.qt.onmessage = function(ev) {
	var data = JSON.parse(ev.data)

	if(data._event !== undefined) {
		eth.trigger(data._event, data.data);
	} else {
		if(data._seed) {
			var cb = eth._callbacks[data._seed];
			if(cb) {
				cb.call(this, data.data)

				// Remove the "trigger" callback
				delete eth._callbacks[ev._seed];
			}
		}
	}
}
