window._messagingAdapter = function(data) {
	navigator.qt.postMessage(data);
};

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
