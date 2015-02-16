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

	function showFullUrlBar(on){
        if (uriNav.focus == false ) {
        	if (on == false) {
                clickAnywhereOnApp.visible = false
                navBar.state = "titleVisible"
        	} else {
                clickAnywhereOnApp.visible = true
                navBar.state = "fullUrlVisible"
       		}
        }

    }

	Component.onCompleted: {
	}

	Item {
		objectName: "root"
		id: root
		anchors {
            fill: parent
        }

		state: "inspectorShown"

		MouseArea {
			id: clickAnywhereOnApp
			z:15
			// Using a secondary screen to catch on mouse exits for the area, because 
            // there are many hover actions conflicting

            anchors {
                top: parent.top
                topMargin: 50
                right: parent.right
                bottom: parent.bottom
                left: parent.left
            }
			hoverEnabled: true
			
			onEntered: {
			  	showFullUrlBar(false);
			}

            onClicked: {
                uriNav.focus = false
                showFullUrlBar(false);
            }

			// Rectangle {
			//     anchors.fill: parent
			//     color: "#88888888"
			// }
		}

		RowLayout {
			id: navBar
			height: 74
			z: 20
			anchors {
				left: parent.left
				right: parent.right
			}

			Button {
				id: back
                z: 30
				onClicked: {					
                    webview.goBack()
				}

				anchors {
					left: parent.left
					leftMargin: 6
				}

				style: ButtonStyle {
                    background: Image {
                         source: (webview.canGoBack) ? 
                            (control.hovered ? "../../backButtonHover.png" : "../../backButton.png") : 
                            "../../backButtonDisabled.png"
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
                z:2
	           MouseArea {
			    	anchors.fill: parent
			    	z: 10
			    	hoverEnabled: true
			    	
			    	onEntered: {
                        showFullUrlBar(true);
                    }
                    /*onExited: {
                        showFullUrlBar(false);
                    }*/	 
			    	   	
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
                     elide: Text.ElideRight

                     anchors {
                         left: parent.left
                         right: parent.horizontalCenter
                         top: parent.top
                         bottom: parent.bottom
                         leftMargin: 32
                     }
                     color: "#928484"
                 }

                 Text {
                 	 id: appDomain
                     text: "loading domain"
                     font.bold: false
                     horizontalAlignment: Text.AlignLeft
                     verticalAlignment: Text.AlignVCenter
                     elide: Text.ElideLeft
                     
                     anchors {
                         left: parent.horizontalCenter
                         right: parent.right
                         top: parent.top
                         bottom: parent.bottom
                         leftMargin: 32

                     }
                     color: "#C0AFAF"
                 }


				TextField {
				    id: uriNav
				    opacity: 0.0

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
                        // if there's no http, add it.
                        var url = this.text,
                        matches = url.match(/^([a-z]*\:\/\/)?([^\/.]+)(:?\/)(.*|$)/i),
                        requestedProtocol = (matches && matches[1] != "undefined")? "" : "http://";

                        webview.url = requestedProtocol + url;
				    }
			    }
   				
			    
			}
			
			Rectangle {
				id: appInfoPaneShadow
			    width: 10
			    height: 30
			    color: "#BDB6B6"
			    radius: 6
                z:1

			    anchors {
					left: back.right
					right: parent.right
					leftMargin:10
					rightMargin:10
					top: parent.top 
					topMargin: 23
				}				
			}

			Rectangle {
				id: navBarBackground
                anchors.fill: parent
                z:-1
                gradient: Gradient {
                    GradientStop { position: 0.0; color: "#F6F1F2" }
                    GradientStop { position: 1.0; color: "#DED5D5" }
                }
            }

            states: [
            	State {
            		name: "fullUrlVisible"
            		PropertyChanges {
                		target: appTitle
                		anchors.rightMargin: -50
                		opacity: 0.0
            		}            		
            		PropertyChanges {
                		target: appDomain
                		anchors.leftMargin: -50
                		opacity: 0.0
            		}
            		PropertyChanges {
                		target: uriNav
                		anchors.leftMargin: 0
                		opacity: 1.0
            		}            		
            	},           	
            	State {
            		name: "titleVisible"

            		PropertyChanges {
            			target: appTitle
                		anchors.rightMargin: 10
                		opacity: 1.0
            		}
            		PropertyChanges {
            			target: appDomain
                		anchors.leftMargin: 10
                		opacity: 1.0
            		}
            		PropertyChanges {
                		target: uriNav
                		anchors.leftMargin: -50
                		opacity: 0.0
            		}              		
            	}

            ]

			transitions: [
      		  // This adds a transition that defaults to applying to all state changes

     		   Transition {
		
     		       // This applies a default NumberAnimation to any changes a state change makes to x or y properties
     		       NumberAnimation { 
     		       		properties: "anchors.leftMargin, anchors.rightMargin, opacity" 
     		       		easing.type: Easing.InOutQuad //Easing.InOutBack
     		       		duration: 300
     		       }
     		   }
    		]            

		}

		WebEngineView {
			objectName: "webView"
			id: webview
			//experimental.settings.javascriptCanAccessClipboard: true
			//experimental.settings.localContentCanAccessRemoteUrls: true
			anchors {
				left: parent.left
				right: parent.right
				bottom: parent.bottom
				top: navBar.bottom
			}
			z: 10

			Timer {
				interval: 2000; running: true; repeat: true
				onTriggered: {
					webview.runJavaScript("try{document.querySelector('meta[name=ethereum-dapp-info]').getAttribute('content')}catch(e){}", function(extraInfo) {
                        if (extraInfo) {
                            menuItem.secondaryTitle = extraInfo;
                        }
                    });
                    webview.runJavaScript("try{document.querySelector('meta[name=ethereum-dapp-badge]').getAttribute('content')}catch(e){}", function(badge) {
                        if (badge) {
                            if (Number(badge)>0 && Number(badge)<999) {
                                menuItem.badgeNumber = Number(badge);
                                menuItem.badgeContent = "number"
                            } else if (badge == "warning") {
                                menuItem.badgeIcon = "\ue00e"
                                menuItem.badgeContent = "icon"

                            } else if (badge == "ghost") {
                                menuItem.badgeIcon = "\ue01a"
                                menuItem.badgeContent = "icon"

                            } else if (badge == "question") {
                                menuItem.badgeIcon = "\ue05d"
                                menuItem.badgeContent = "icon"

                            } else if (badge == "info") {
                                menuItem.badgeIcon = "\ue08b"
                                menuItem.badgeContent = "icon"

                            } else if (badge == "check") {
                                menuItem.badgeIcon = "\ue080"
                                menuItem.badgeContent = "icon"

                            } else if (badge == "gear") {
                                menuItem.badgeIcon = "\ue09a"
                                menuItem.badgeContent = "icon"

                            }


                            console.log(menuItem.badgeContent);
                        } else {
                            menuItem.badgeContent = ""
                        } 
                    });
				}
			}
			
			onLoadingChanged: {
				if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
					webview.runJavaScript("document.title", function(pageTitle) {
						menuItem.title = pageTitle;	
					});

                    webView.runJavaScript("try{document.querySelector(\"link[rel='icon']\").getAttribute(\"href\")}catch(e){}", function(sideIcon){
                            if(sideIcon){
                                menuItem.icon = webview.url + sideIcon;
                                console.log("icon: " + webview.url + sideIcon );
                            }; 
                            console.log("no icon!" );
                    });
                    
					webView.runJavaScript("try{document.querySelector(\"meta[name='ethereum-dapp-url-bar-style']\").getAttribute(\"content\")}catch(e){}", function(topBarStyle){
						if (!topBarStyle) {
							showFullUrlBar(true);
							navBarBackground.visible = true;
							back.visible = true;
							appInfoPane.anchors.leftMargin = 0;
							appInfoPaneShadow.anchors.leftMargin = 0;
							webview.anchors.topMargin = 0;
							return;
						}

						if (topBarStyle=="transparent") {
							// Adjust for a transparent sidebar Dapp
							navBarBackground.visible = false;
							back.visible = false;
							appInfoPane.anchors.leftMargin = -16;
							appInfoPaneShadow.anchors.leftMargin = -16;
							webview.anchors.topMargin = -74;
							webview.runJavaScript("document.querySelector('body').classList.add('ethereum-dapp-url-bar-style-transparent')")

						} else {
							navBarBackground.visible = true;
							back.visible = true;
							appInfoPane.anchors.leftMargin = 0;
							appInfoPaneShadow.anchors.leftMargin = 0;
							webview.anchors.topMargin = 0;
						};	
					});


					webview.runJavaScript(eth.readFile("bignumber.min.js"));
                    webview.runJavaScript(eth.readFile("ethereum.js/dist/ethereum.js"));

					var cleanTitle = webview.url.toString()
					var matches = cleanTitle.match(/^[a-z]*\:\/\/([^\/?#]+)(?:[\/?#]|$)/i);
					var domain = matches && matches[1];


					if (domain)
						appDomain.text = domain //webview.url.replace("a", "z")
					if (webview.title)
						appTitle.text = webview.title

					showFullUrlBar(false);
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
