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

	property var title: ""
	property var iconSource: ""
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
			height: 184
			z: 20

			anchors {
				left: parent.left
				right: parent.right
			}

			Rectangle {
				id: appInfoPane
			    height: 28
			    color: "#efefef"
			    radius: 6
			    z:25

			   MouseArea {
			    	anchors.fill: parent
			    	z: 10
			    	hoverEnabled: true
			    	onEntered: {
			    		uriNav.visible = true
			    		appTitle.visible = false
			    		appDomain.visible = false
			    	}	    	
			    }

			    anchors {
					left: parent.left
					right: parent.right
					leftMargin: 10
					rightMargin: 10
					top: parent.verticalCenter 
					topMargin: 23
				}

				TextField {
				    id: uriNav
				    anchors {
				    	left: parent.left
				    	right: parent.right
				    	leftMargin: 16
						top: parent.verticalCenter 
						topMargin: -10
				    }

				    horizontalAlignment: Text.AlignHCenter
                    
                    style: TextFieldStyle {
                        textColor: "#928484"
                        background: Rectangle {
                            border.width: 0
                            color: "transparent"
                        }
                    }
    				text: "Type the address of a new Dapp";
				    y: parent.height / 2 - this.height / 2
				    z: 30
				    activeFocusOnPress: true
				    Keys.onReturnPressed: {
        				newBrowserTab(this.text);
        				this.text = "Type the address of a new Dapp";
				    }

			    }   				
			}
			
			Rectangle {
				id: appInfoPaneShadow
			    width: 10
			    height: 30
			    color: "#BDB6B6"
			    radius: 6
			    z: 15

			    anchors {
					left: parent.left
					right: parent.right
					leftMargin:10
					rightMargin:10
					top: parent.verticalCenter 
					topMargin: 23
				}

				
			}

		}


		WebEngineView {
			objectName: "webView"
			id: webview
			anchors.fill: parent

			onLoadingChanged: {
				if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
					webview.runJavaScript(eth.readFile("bignumber.min.js"));
					webview.runJavaScript(eth.readFile("ethereum.js/dist/ethereum.js"));
				}
			}
			onJavaScriptConsoleMessage: {
				console.log(sourceID + ":" + lineNumber + ":" + JSON.stringify(message));
			}
		}



		WebEngineView {
			id: inspector
			visible: false
			z:10
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
