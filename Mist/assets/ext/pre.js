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
				// Figure out whether the returned data was an array
				// array means multiple return arguments (multiple params)
				if(data.data instanceof Array) {
					cb.apply(this, data.data)
				} else {
					cb.call(this, data.data)
				}

				// Remove the "trigger" callback
				delete eth._callbacks[ev._seed];
			}
		}
	}
}
