// The magic return variable. The magic return variable will be set during the execution of the QML call.
(function(window) {
	function message(type, data) {
		document.title = JSON.stringify({type: type, data: data});

		return window.____returnData;
	}

	function isPromise(o) {
		return typeof o === "object" && o.then
	}

	window.eth = {
		_callbacks: {},
		_onCallbacks: {},
		prototype: Object(),

		coinbase: function() {
			return new Promise(function(resolve, reject) {
				postData({call: "getCoinBase"}, function(coinbase) {
					resolve(coinbase);
				});
			});
		},

		block: function(numberOrHash) {
			return new Promise(function(resolve, reject) {
				var func;
				if(typeof numberOrHash == "string") {
					func =  "getBlockByHash";
				} else {
					func =  "getBlockByNumber";
				}

				postData({call: func, args: [numberOrHash]}, function(block) {
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
			return Q.all(promises).then(function() {
				return new Promise(function(resolve, reject) {
					postData({call: "transact", args: params}, function(data) {
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
				postData({call: "compile", args: [code]}, function(data) {
					if(data[1])
						reject(data[0]);
					else
						resolve(data[0]);
				});
			});
		},

		key: function() {
			return new Promise(function(resolve, reject) {
				postData({call: "getKey"}, function(k) {
					resolve(k);
				});
			});
		}
	};

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
})(this);
