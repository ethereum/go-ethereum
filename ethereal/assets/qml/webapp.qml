import QtQuick 2.0
import QtWebKit 3.0
import QtWebKit.experimental 1.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Window 2.1;
import Ethereum 1.0

ApplicationWindow {
	id: window
	title: "Ethereum"
	width: 900
	height: 600
	minimumHeight: 300

	property alias url: webview.url
	property alias webView: webview

	Item {
		objectName: "root"
		id: root
		anchors.fill: parent
		state: "inspectorShown"

		RowLayout {
			id: navBar
			anchors {
				left: parent.left
				right: parent.right
			}

			Button {
				id: back
				iconSource: "../back.png"
				onClicked: {
					webview.goBack()
				}
			}

			TextField {
				anchors {
					top: parent.top
					left: back.right
					right: parent.right
				}
				id: uriNav

				Keys.onReturnPressed: {
					var uri = this.text;
					if(!/.*\:\/\/.*/.test(uri)) {
						uri = "http://" + uri;
					}

					var reg = /(^https?\:\/\/(?:www\.)?)([a-zA-Z0-9_\-]*\.eth)(.*)/

					if(reg.test(uri)) {
						this.text.replace(reg, function(match, pre, domain, path) {
							uri = pre;

							var lookup = eth.lookupDomain(domain.substring(0, domain.length - 4));
							var ip = [];
							for(var i = 0, l = lookup.length; i < l; i++) {
								ip.push(lookup.charCodeAt(i))
							}

							if(ip.length != 0) {
								uri += lookup;
							} else {
								uri += domain;
							}

							uri += path;
						});
					}

					console.log("connecting to ...", uri)

					webview.url = uri;
				}
			}
		}

		WebView {
			objectName: "webView"
			id: webview
			anchors {
				left: parent.left
				right: parent.right
				bottom: parent.bottom
				top: navBar.bottom
			}
			onTitleChanged: { window.title = title }
			experimental.preferences.javascriptEnabled: true
			experimental.preferences.navigatorQtObjectEnabled: true
			experimental.preferences.developerExtrasEnabled: true
			experimental.userScripts: ["../ext/pre.js", "../ext/big.js", "../ext/string.js", "../ext/ethereum.js"]
			experimental.onMessageReceived: {
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

			function require(args, num) {
				if(args.length < num) {
					throw("required argument count of "+num+" got "+args.length);
				}
			}
			function postData(seed, data) {
				webview.experimental.postMessage(JSON.stringify({data: data, _seed: seed}))
			}
			function postEvent(event, data) {
				webview.experimental.postMessage(JSON.stringify({data: data, _event: event}))
			}

			function onNewBlockCb(block) {
				postEvent("block:new", block)
			}
			function onObjectChangeCb(stateObject) {
				postEvent("object:"+stateObject.address(), stateObject)
			}
			function onStorageChangeCb(storageObject) {
				var ev = ["storage", storageObject.stateAddress, storageObject.address].join(":");
				postEvent(ev, [storageObject.address, storageObject.value])
			}
		}

		Rectangle {
			id: toggleInspector
			color: "#bcbcbc"
			visible: true
			height: 20
			width: 20
			anchors {
				right: root.right
			}
			MouseArea {
				onClicked: {
					if(inspector.visible == true){
						inspector.visible = false
					}else{
						inspector.visible = true
						inspector.url = webview.experimental.remoteInspectorUrl
					}
				}

				onDoubleClicked: {
					webView.reload()
				}
				anchors.fill: parent
			}
		}

		Rectangle {
			id: sizeGrip
			color: "gray"
			visible: false
			height: 10
			anchors {
				left: root.left
				right: root.right
			}
			y: Math.round(root.height * 2 / 3)

			MouseArea {
				anchors.fill: parent
				drag.target: sizeGrip
				drag.minimumY: 0
				drag.maximumY: root.height
				drag.axis: Drag.YAxis
			}
		}

		WebView {
			id: inspector
			visible: false
			anchors {
				left: root.left
				right: root.right
				top: sizeGrip.bottom
				bottom: root.bottom
			}
		}

		states: [
			State {
				name: "inspectorShown"
				PropertyChanges {
					target: inspector
				}
			}
		]
	}
}
