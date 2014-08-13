function handleMessage(message) {
	console.log("[onMessageReceived]: ", message.data)
	// TODO move to messaging.js
	var data = JSON.parse(message.data)

	try {
		switch(data.call) {
			case "getCoinBase":
				postData(data._seed, eth.getCoinBase())

			break
			case "getIsListening":
				postData(data._seed, eth.getIsListening())

			break
			case "getIsMining":
				postData(data._seed, eth.getIsMining())

			break
			case "getPeerCount":
				postData(data._seed, eth.getPeerCount())

			break

			case "getTxCountAt":
				require(1)
			postData(data._seed, eth.getTxCountAt(data.args[0]))

			break
			case "getBlockByNumber":
				var block = eth.getBlock(data.args[0])
			postData(data._seed, block)

			break
			case "getBlockByHash":
				var block = eth.getBlock(data.args[0])
			postData(data._seed, block)

			break
			case "transact":
				require(5)

			var tx = eth.transact(data.args[0], data.args[1], data.args[2],data.args[3],data.args[4],data.args[5])
			postData(data._seed, tx)

			break
			case "create":
				postData(data._seed, null)

			break
			case "getStorage":
				require(2);

			var stateObject = eth.getStateObject(data.args[0])
			var storage = stateObject.getStorage(data.args[1])
			postData(data._seed, storage)

			break
			case "getStateKeyVals":
				require(1);
			var stateObject = eth.getStateObject(data.args[0]).stateKeyVal(true)
			postData(data._seed,stateObject)

			break
			case "getTransactionsFor":
				require(1);
			var txs = eth.getTransactionsFor(data.args[0], true)
			postData(data._seed, txs)

			break
			case "getBalance":
				require(1);

			postData(data._seed, eth.getStateObject(data.args[0]).value());

			break
			case "getKey":
				var key = eth.getKey().privateKey;

			postData(data._seed, key)
			break
			case "watch":
				require(1)
				eth.watch(data.args[0], data.args[1]);
			break
			case "disconnect":
				require(1)
			postData(data._seed, null)
			break;
			case "set":
				console.log("'Set' has been depcrecated")
			/*
			   for(var key in data.args) {
			   if(webview.hasOwnProperty(key)) {
			   window[key] = data.args[key];
			   }
			   }
			   */
			break;
			case "getSecretToAddress":
				require(1)
			postData(data._seed, eth.secretToAddress(data.args[0]))
			break;
			case "debug":
				console.log(data.args[0]);
			break;
		}
	} catch(e) {
		console.log(data.call + ": " + e)

		postData(data._seed, null);
	}
}

function postData(seed, data) {
	webview.experimental.postMessage(JSON.stringify({data: data, _seed: seed}))
}
