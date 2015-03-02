import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Controls.Styles 1.0
import QtQuick.Layouts 1.0;
import QtWebEngine 1.0
import QtWebEngine.experimental 1.0
import QtQuick.Window 2.0;
import Ethereum 1.0
import Qt.WebSockets 1.0
//import "qwebchannel.js" as WebChannel
	


Rectangle {
	id: window
	anchors.fill: parent
	color: "#00000000"

	property var title: "Network"
	property var iconSource: "../mining-icon.png"
	property var menuItem
	property var hideUrl: true

	property alias url: webview.url
	property alias windowTitle: webview.title
	property alias webView: webview

	property var cleanPath: false
	property var open: function(url) {
		if(!window.cleanPath) {
			var uri = url;
			if(!/.*\:\/\/.*/.test(uri)) {
				uri = "http://" + uri;
			}

			var reg = /(^https?\:\/\/(?:www\.)?)([a-zA-Z0-9_\-]*\.eth)(.*)/

			if(reg.test(uri)) {
				uri.replace(reg, function(match, pre, domain, path) {
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

			window.cleanPath = true;

			webview.url = uri;

			//uriNav.text = uri.text.replace(/(^https?\:\/\/(?:www\.)?)([a-zA-Z0-9_\-]*\.\w{2,3})(.*)/, "$1$2<span style='color:#CCC'>$3</span>");
			uriNav.text = uri;

		} else {
			// Prevent inf loop.
			window.cleanPath = false;
		}
	}

	Label {
        objectName: "miningLabel"
        visible: false
        font.pixelSize: 10
        anchors.right: lastBlockLabel.left
        anchors.rightMargin: 5
    	onTextChanged: {
			menuItem.secondaryTitle =  eth.miner().mining()? eth.miner().hashRate() + " Khash"  : ""
    	}
    }

	Item {
		objectName: "root"
		id: root
		anchors.fill: parent
		state: "inspectorShown"

		Timer {
	         interval: 1000; running: true; repeat: true
	         onTriggered: {
	         	webview.runJavaScript("Miner.mining", function(miningSliderValue) {  

	         			// Check if it's mining and set it accordingly       				
         				if (miningSliderValue > 0 && !eth.miner().mining()) {
							eth.setGasPrice("10000000000000");
							
	         				eth.miner().start();
	         			} else if (miningSliderValue == 0 && eth.miner().mining()) {
	         				eth.miner().stop();
	         			} else if (eth.miner().mining()) {
	         				
	         				webview.runJavaScript('console.log(localStorage.timeSpent); Miner.timeSpentMining++; Miner.hashrate = ' + eth.miner().hashRate() );


	         			} else if (miningSliderValue == "undefined") {
	         				
	         				webview.runJavaScript('Miner.mining = 0' );
	         				
	         			}					
				});

	         }
	    }

		WebEngineView {
			objectName: "webView"
			id: webview
			anchors.fill: parent

			url: "http://localhost:3000/"

			experimental.settings.javascriptCanAccessClipboard: true


			onJavaScriptConsoleMessage: {
				console.log(sourceID + ":" + lineNumber + ":" + JSON.stringify(message));
			}

			onLoadingChanged: {
				if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
                    webview.runJavaScript(eth.readFile("mist.js"));
				}
			}
		}






		WebEngineView {
			id: inspector
			visible: false
			z:10
			anchors {
				left: root.left
				right: root.right
				top: root.top
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
