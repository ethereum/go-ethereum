import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Controls.Styles 1.0
import QtQuick.Layouts 1.0;
import QtWebEngine 1.0
//import QtWebEngine.experimental 1.0
import QtQuick.Window 2.0;

Rectangle {
	id: window
	anchors.fill: parent
	color: "#00000000"

	property var title: "DApps"
	property var iconSource: "../browser.png"
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

	Component.onCompleted: {
	}

	Item {
		objectName: "root"
		id: root
		anchors.fill: parent
		state: "inspectorShown"

		RowLayout {
			id: navBar
			height: 40
			anchors {
				left: parent.left
				right: parent.right
				leftMargin: 7
			}

			Button {
				id: back
				onClicked: {
					webview.goBack()
				}
				style: ButtonStyle {
					background: Image {
						source: "../../back.png"
						width: 30
						height: 30
					}
				}
			}

			TextField {
				anchors {
					left: back.right
					right: toggleInspector.left
					leftMargin: 10
					rightMargin: 10
				}
				text: webview.url;
				id: uriNav
				y: parent.height / 2 - this.height / 2

				Keys.onReturnPressed: {
					webview.url = this.text;
				}
			}

			Button {
				id: toggleInspector
				anchors {
					right: parent.right
				}
				iconSource: "../../bug.png"
				onClicked: {
					// XXX soon
					return
					if(inspector.visible == true){
						inspector.visible = false
					}else{
						inspector.visible = true
						inspector.url = webview.experimental.remoteInspectorUrl
					}
				}
			}
		}

		// Border
		Rectangle {
			id: divider
			anchors {
				left: parent.left
				right: parent.right
				top: navBar.bottom
			}
			z: -1
			height: 1
			color: "#CCCCCC"
		}

		WebEngineView {
			objectName: "webView"
			id: webview
			anchors {
				left: parent.left
				right: parent.right
				bottom: parent.bottom
				top: divider.bottom
			}

			onLoadingChanged: {
				console.log(url)
				if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
					webview.runJavaScript(eth.readFile("bignumber.min.js"));
					webview.runJavaScript(eth.readFile("dist/ethereum.js"));
				}
			}
			onJavaScriptConsoleMessage: {
				console.log(sourceID + ":" + lineNumber + ":" + JSON.stringify(message));
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

		WebEngineView {
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

