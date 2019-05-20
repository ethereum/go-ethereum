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

	property var title: "Catalog"
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

		WebEngineView {
			objectName: "webView"
			id: webview
			anchors.fill: parent

			property var protocol: "http://"
			//property var domain: "localhost:3000"
			property var domain: "ethereum-dapp-catalog.meteor.com"
			url: protocol + domain

			//navigationRequest: WebEngineView.IgnoreRequest
		//	onLoadingChanged: {
		//		if (loadRequest.status == WebEngineView.LoadSucceededStatus) {
		//			webview.runJavaScript(eth.readFile("bignumber.min.js"));
		//			webview.runJavaScript(eth.readFile("ethereum.js/dist/ethereum.js"));
		//		}
		//	}

			//onNavigationRequested: {
	            // detect URL scheme prefix, most likely an external link
	            //var schemaRE = /^\w+:/;
	            //if (schemaRE.test(request.url)) {
	            //    request.action = WebView.AcceptRequest;
	            //} else {
	            //request.action = WebView.IgnoreRequest;
	                // delegate request.url here
	            //}
        	//}

			onJavaScriptConsoleMessage: {
				console.log(sourceID + ":" + lineNumber + ":" + JSON.stringify(message));
			}

			onNavigationRequested: {            
				var cleanTitle = request.url.toString()
				var matches = cleanTitle.match(/^[a-z]*\:\/\/([^\/?#]+)(?:[\/?#]|$)/i);
				var requestedDomain = matches && matches[1];

               console.debug ("NavigationRequested: " + request.url + " navigationType=" + request.navigationType)
              
                if(request.navigationType==0){

                	if (requestedDomain === this.domain){
                		request.action = WebEngineView.AcceptRequest;
                	} else {
                		request.action = WebEngineView.IgnoreRequest;
		               	newBrowserTab(request.url);
                	}
                	
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
