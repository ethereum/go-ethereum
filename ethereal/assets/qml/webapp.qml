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

		WebView {
			objectName: "webView"
			id: webview
			anchors.fill: parent
			/*
			 anchors {
				 left: parent.left
				 right: parent.right
				 bottom: sizeGrip.top
				 top: parent.top
			 }
			 */

			onTitleChanged: { window.title = title }
			experimental.preferences.javascriptEnabled: true
			experimental.preferences.navigatorQtObjectEnabled: true
			experimental.preferences.developerExtrasEnabled: true
			experimental.userScripts: [ui.assetPath("ethereum.js")]
			experimental.onMessageReceived: {
				console.log("[onMessageReceived]: ", message.data)
				// TODO move to messaging.js
				var data = JSON.parse(message.data)

				try {
					switch(data.call) {
					case "getBlockByNumber":
						var block = eth.getBlock("b9b56cf6f907fbee21db0cd7cbc0e6fea2fe29503a3943e275c5e467d649cb06")	
						postData(data._seed, block)
						break
					case "getBlockByHash":
						var block = eth.getBlock("b9b56cf6f907fbee21db0cd7cbc0e6fea2fe29503a3943e275c5e467d649cb06")	
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
						for(var key in data.args) {
							if(webview.hasOwnProperty(key)) {
								window[key] = data.args[key];
							}
						}
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
			url: webview.experimental.remoteInspectorUrl
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
					url: webview.experimental.remoteInspectorUrl
				}
			}
		]
	}
}
