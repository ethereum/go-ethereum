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

	getBalanceAt: function(address, cb) {
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
		if(typeof args[i] === "object") {
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

window.eth._0 = "\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0"
String.prototype.pad = function(len) {
    var bin = this.bin();
    var l = bin.length;
    if(l < 32) {
        return eth._0.substr(0, 32 - bin.length) + bin;
    }

    return bin;
}

String.prototype.unpad = function() {
    var i, l;
    for(i = 0, l = this.length; i < l; i++) {
        if(this[i] != "\0") {
            return this.substr(i, this.length);
        }
    }

    return this.substr(i, this.length);
}

String.prototype.bin = function() {
    if(this.substr(0, 2) == "0x") {
        return this.hex2bin();
    } else if(/^\d+$/.test(this)) {
        return this.num2bin()
    }

    // Otherwise we'll return the "String" object instead of an actual string
    return this.substr(0, this.length)
}

String.prototype.unbin = function() {
    var i, l, o = '';
    for(i = 0, l = this.length; i < l; i++) {
        var n = this.charCodeAt(i).toString(16);
        o += n.length < 2 ? '0' + n : n;
    }

    return "0x" + o;
}

String.prototype.hex2bin = function() {
    bytes = []

    for(var i=2; i< this.length-1; i+=2) {
        bytes.push(parseInt(this.substr(i, 2), 16));
    }

    return String.fromCharCode.apply(String, bytes);
}

String.prototype.num2bin = function() {
    return ("0x"+parseInt(this).toString(16)).bin()
}
