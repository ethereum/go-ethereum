// Helper function for generating pseudo callbacks and sending data to the QML part of the application
function postData(data, cb) {
	data._seed = Math.floor(Math.random() * 1000000)
	if(cb) {
		eth._callbacks[data._seed] = cb;
	}

	if(data.args === undefined) {
		data.args = []
	}

	navigator.qt.postMessage(JSON.stringify(data));
}

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
                        func =  "getBlockByHash"
                } else {
                        func =  "getBlockByNumber"
                }
                postData({call: func, args: [numberOrHash]}, cb)
        },

	// Create transaction
	//
	// Creates a transaction with the current account
	// If no recipient is set, the Ethereum API will see it as a contract creation
	createTx: function(recipient, value, gas, gasPrice, data, cb) {
		postData({call: "createTx", args: [recipient, value, gas, gasPrice, data]}, cb)
	},

	getStorage: function(address, storageAddress, cb) {
		postData({call: "getStorage", args: [address, storageAddress]}, cb)
	},

	getKey: function(cb) {
		postData({call: "getKey"}, cb)
	},
}
window.eth._callbacks = {}

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

navigator.qt.onmessage = function(ev) {
	var data = JSON.parse(ev.data)

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
