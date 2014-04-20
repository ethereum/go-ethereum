// Helper function for generating pseudo callbacks and sending data to the QML part of the application
function postData(data, cb) {
	data._seed = Math.floor(Math.random() * 1000000)
	if(cb) {
		eth._callbacks[data._seed] = cb;
	}

	navigator.qt.postMessage(JSON.stringify(data));
}

// Main Ethereum library
window.eth = {
	prototype: Object(),

	send: function(cb) {
		document.getElementById("out").innerHTML = "clicked";
		postData({message: "Hello world"}, cb);
	}
}
window.eth._callbacks = {}

function debug(/**/) {
	var args = arguments;
	var msg = ""
	for(var i=0; i<args.length; i++){
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
