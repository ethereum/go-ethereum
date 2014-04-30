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
	// Creates a transaction with the current account
	// If no recipient is set, the Ethereum API will see it as a contract creation
	transact: function(sec, recipient, value, gas, gasPrice, data, cb) {
		postData({call: "transact", args: [sec, recipient, value, gas, gasPrice, data]}, cb);
	},

	create: function(sec, value, gas, gasPrice, init, body, cb) {
		postData({call: "create", args: [sec, value, gas, gasPrice, init, body]}, cb);
	},

	getStorage: function(address, storageAddress, cb) {
		postData({call: "getStorage", args: [address, storageAddress]}, cb);
	},

	getKey: function(cb) {
		postData({call: "getKey"}, cb);
	},

	getBalance: function(address, cb) {
		postData({call: "getBalance", args: [address]}, cb);
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
				callbacks[i](data);
			}
		}
	},
}
window.eth._callbacks = {}
window.eth._onCallbacks = {}

function debug(/**/) {
	var args = arguments;
	var msg = ""
	for(var i = 0; i < args.length; i++){
		if(typeof args[i] == "object") {
			msg += " " + JSON.stringify(args[i])
		} else {
			msg += args[i]
		}
	}

	document.getElementById("debug").innerHTML += "<br>" + msg
}

// Helper function for generating pseudo callbacks and sending data to the QML part of the application
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
				// Call the callback
				cb(data.data);
				// Remove the "trigger" callback
				delete eth._callbacks[ev._seed];
			}
		}
	}
}
