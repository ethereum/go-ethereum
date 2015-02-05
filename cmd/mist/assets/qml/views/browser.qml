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
			height: 74
			

			anchors {
				left: parent.left
				right: parent.right
			}

			Button {
				id: back

				onClicked: {
					webview.goBack()
				}

				anchors{
					left: parent.left
					leftMargin: 6
				}

				style: ButtonStyle {
					background: Image {
						source: "../../backButton.png"
						width: 20
						height: 30
					}
				}
			}

			Rectangle {
				id: appInfoPane
			    height: 28
			    color: "#FFFFFF"
			    radius: 6

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
					left: back.right
					right: parent.right
					leftMargin: 10
					rightMargin: 10
				}

				Text {
   					 id: appTitle
                     text: "LOADING"
                     font.bold: true
                     font.capitalization: Font.AllUppercase 
                     horizontalAlignment: Text.AlignRight
                     verticalAlignment: Text.AlignVCenter
                     
                     anchors {
                         left: parent.left
                         right: parent.horizontalCenter
                         top: parent.top
                         bottom: parent.bottom
                         rightMargin: 10
                     }
                     color: "#928484"
                 }

                 Text {
                 	 id: appDomain
                     text: "loading domain"
                     font.bold: false
                     horizontalAlignment: Text.AlignLeft
                     verticalAlignment: Text.AlignVCenter
                     
                     anchors {
                         left: parent.horizontalCenter
                         right: parent.right
                         top: parent.top
                         bottom: parent.bottom
                         leftMargin: 10
                     }
                     color: "#C0AFAF"
                 }


				TextField {
				    id: uriNav
				    visible: false
				    anchors {
				    	left: parent.left
				    	right: parent.right
				    	leftMargin: 16
				    }

				    horizontalAlignment: Text.AlignHCenter
                    
                    style: TextFieldStyle {
                        textColor: "#928484"
                        background: Rectangle {
                            border.width: 0
                            color: "transparent"
                        }
                    }
    				text: webview.url;
				    y: parent.height / 2 - this.height / 2
				    z: 20
				    activeFocusOnPress: true
				    Keys.onReturnPressed: {
				    	webview.url = this.text;
				    }

			    }
   				
   				



                 			    /*text {
			    	id: appTitle
			    	anchors.left: parent.left
			    	anchors.right: parent.horizontalCenter
			    	text: "APP TITLE"
			    	font.bold: true
			    	color: "#928484"
			    }*/
			    z:2
			}
			
			Rectangle {
				id: appInfoPaneShadow
			    width: 10
			    height: 30
			    color: "#BDB6B6"
			    radius: 6

			    anchors {
					left: back.right
					right: parent.right
					leftMargin:10
					rightMargin:10
					top: parent.top 
					topMargin: 23
				}

				z:1
			}
		/*
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
			}*/

			Rectangle {
                anchors.fill: parent
                gradient: Gradient {
                    GradientStop { position: 0.0; color: "#F6F1F2" }
                    GradientStop { position: 1.0; color: "#DED5D5" }
                }
                z:-1
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
				if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
					webview.runJavaScript("document.title", function(pageTitle) {
						menuItem.title = pageTitle;	
					});
					webview.runJavaScript(eth.readFile("bignumber.min.js"));
					webview.runJavaScript(eth.readFile("ethereum.js/dist/ethereum.js"));

					var cleanTitle = webview.url.toString()
					var matches = cleanTitle.match(/^[a-z]*\:\/\/([^\/?#]+)(?:[\/?#]|$)/i);
					var domain = matches && matches[1];

					uriNav.visible = false
			    	appDomain.visible = true
			    	appDomain.text = domain //webview.url.replace("a", "z")

			    	appTitle.visible = true
			    	appTitle.text = webview.title
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
