// Copyright (c) 2013-2014, Jeffrey Wilcke. All rights reserved.
//
// This library is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation; either
// version 2.1 of the License, or (at your option) any later version.
//
// This library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this library; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston,
// MA 02110-1301  USA

// Helper function for generating pseudo callbacks and sending data to the QML part of the application
function postData(data, cb) {
	data._seed = Math.floor(Math.random() * 1000000)
	if(cb) {
		Muted._callbacks[data._seed] = cb;
	}

	if(data.args === undefined) {
		data.args = [];
	}

	navigator.qt.postMessage(JSON.stringify(data));
}

window.Muted = {
	prototype: Object(),
}

window.Muted._callbacks = {}
window.Muted._onCallbacks = {}

function debug(/**/) {
	console.log("hello world")

	var args = arguments;
	var msg = ""
	for(var i = 0; i < args.length; i++){
		if(typeof args[i] == "object") {
			msg += " " + JSON.stringify(args[i])
		} else {
			msg += args[i]
		}
	}

	document.querySelector("#debugger").innerHTML += "<div class='line'><div class='col1'></div><div class='col2'>"+msg+"</div></div>";
}
console.log = function() {
	var args = []
	for(var i = 0; i < arguments.length; i++) {
		args.push(arguments[i]);
	}
	postData({call:"log", args:args})
}

navigator.qt.onmessage = function(ev) {
	var data = JSON.parse(ev.data)

	if(data._event !== undefined) {
		Muted.trigger(data._event, data.data);
	} else {
		if(data._seed) {
			var cb = Muted._callbacks[data._seed];
			if(cb) {
				// Call the callback
				cb(data.data);
				// Remove the "trigger" callback
				delete Muted._callbacks[ev._seed];
			}
		}
	}
}
