import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

ApplicationWindow {
	id: root

	property alias miningButtonText: miningButton.text

	width: 900
	height: 600
	minimumHeight: 300

	title: "Ethereal"

	// Takes care of loading all default plugins
	Component.onCompleted: {
		var historyView = addPlugin("./views/history.qml", {title: "History"})
		var newTxView = addPlugin("./views/transaction.qml", {title: "New Transaction"})
		var chainView = addPlugin("./views/chain.qml", {title: "Block chain"})
		var infoView = addPlugin("./views/info.qml", {title: "Info"})
		var pendingTxView = addPlugin("./views/pending_tx.qml", {title: "Pending", canClose: true})
		var pendingTxView = addPlugin("./views/javascript.qml", {title: "JavaScript", canClose: true})

		// Call the ready handler
		gui.done()
	}

	function addPlugin(path, options) {
		var component = Qt.createComponent(path);
		if(component.status != Component.Ready) {
			if(component.status == Component.Error) {
				console.debug("Error:"+ component.errorString());
			}
			return
		}

		return mainSplit.addComponent(component, options)
	}

	MenuBar {
		Menu {
			title: "File"
			MenuItem {
				text: "Import App"
				shortcut: "Ctrl+o"
				onTriggered: {
					generalFileDialog.callback = importApp;
					generalFileDialog.open()
				}
			}

			MenuItem {
				text: "Browser"
				onTriggered: eth.openBrowser()
			}

			MenuItem {
				text: "Add plugin"
				onTriggered: {
					generalFileDialog.callback = function(path) {
						addPlugin(path, {canClose: true})
					}
					generalFileDialog.open()
				}
			}

			MenuSeparator {}

			MenuItem {
				text: "Import key"
				shortcut: "Ctrl+i"
				onTriggered: {
					generalFileDialog.callback = function(path) {
						ui.importKey(path)
					}
					generalFileDialog.open()
				}
			}

			MenuItem {
				text: "Export keys"
				shortcut: "Ctrl+e"
				onTriggered: {
					generalFileDialog.callback = function(path) {
					}
					generalFileDialog.open()
				}
			}
		}

		Menu {
			title: "Developer"
			MenuItem {
				text: "Debugger"
				shortcut: "Ctrl+d"
				onTriggered: ui.startDebugger()
			}

			MenuItem {
				text: "Import Tx"
				onTriggered: {
					txImportDialog.visible = true
				}
			}

			MenuItem {
				text: "Run JS file"
				onTriggered: {
					generalFileDialog.callback = function(path) {
						eth.evalJavascriptFile(path)
					}
					generalFileDialog.open()
				}
			}
		}

		Menu {
			title: "Network"
			MenuItem {
				text: "Add Peer"
				shortcut: "Ctrl+p"
				onTriggered: {
					addPeerWin.visible = true
				}
			}
			MenuItem {
				text: "Show Peers"
				shortcut: "Ctrl+e"
				onTriggered: {
					peerWindow.visible = true
				}
			}
		}

		Menu {
			title: "Help"
			MenuItem {
				text: "About"
				onTriggered: {
					aboutWin.visible = true
				}
			}
		}

	}

	statusBar: StatusBar {
		height: 32
		RowLayout {
			Button {
				id: miningButton
				text: "Start Mining"
				onClicked: {
					gui.toggleMining()
				}
			}

			Button {
				id: importAppButton
				text: "Browser"
				onClicked: {
					eth.openBrowser()
				}
			}

			RowLayout {
				Label {
					id: walletValueLabel

					font.pixelSize: 10
					styleColor: "#797979"
				}
			}
		}

		Label {
			y: 6
			id: lastBlockLabel
			objectName: "lastBlockLabel"
			visible: true
			text: ""
			font.pixelSize: 10
			anchors.right: peerGroup.left
			anchors.rightMargin: 5
		}

		ProgressBar {
			id: syncProgressIndicator
			visible: false
			objectName: "syncProgressIndicator"
			y: 3
			width: 140
			indeterminate: true
			anchors.right: peerGroup.left
			anchors.rightMargin: 5
		}

		RowLayout {
			id: peerGroup
			y: 7
			anchors.right: parent.right
			MouseArea {
				onDoubleClicked:  peerWindow.visible = true
				anchors.fill: parent
			}

			Label {
				id: peerLabel
				font.pixelSize: 8
				text: "0 / 0"
			}
			Image {
				id: peerImage
				width: 10; height: 10
				source: "../network.png"
			}
		}
	}


	property var blockModel: ListModel {
		id: blockModel
	}

	SplitView {
		property var views: [];

		id: mainSplit
		anchors.fill: parent
		resizing: false

		function setView(view) {
			for(var i = 0; i < views.length; i++) {
				views[i].visible = false
			}

			view.visible = true
		}

		function addComponent(component, options) {
			var view = mainView.createView(component, options)
			if(!view.hasOwnProperty("iconFile")) {
				console.log("Could not load plugin. Property 'iconFile' not found on view.");
				return;
			}

			menu.createMenuItem(view.iconFile, view, options);
			mainSplit.views.push(view);

			return view
		}

		/*********************
		 * Main menu.
		 ********************/
		Rectangle {
			id: menu
			Layout.minimumWidth: 80
			Layout.maximumWidth: 80
			anchors.top: parent.top
			color: "#252525"

			Component {
				id: menuItemTemplate
				Image {
					property var view;
					anchors.horizontalCenter: parent.horizontalCenter
					MouseArea {
						anchors.fill: parent
						onClicked: {
							mainSplit.setView(view)
						}
					}
				}
			}

		       /*
			Component {
				id: menuItemTemplate
				Rectangle {
					property var view;
					property var source;
					property alias title: title.text
					height: 25

					id: tab

					anchors {
						left: parent.left
						right: parent.right
					}

					Label {
						id: title
						y: parent.height / 2 - this.height / 2
						x: 5
						font.pixelSize: 10
					}

					MouseArea {
						anchors.fill: parent
						onClicked: {
							mainSplit.setView(view)
						}
					}

					Image {
						id: closeButton
						y: parent.height / 2 - this.height / 2
						visible: false

						source: "../close.png"
						anchors {
							right: parent.right
							rightMargin: 5
						}

						MouseArea {
							anchors.fill: parent
							onClicked: {
								console.log("should close")
							}
						}
					}
				}
			}
			*/


			function createMenuItem(icon, view, options) {
				if(options === undefined) {
					options = {};
				}

				var comp = menuItemTemplate.createObject(menuColumn)
				comp.view = view
				comp.source = icon
				/*
				comp.title = options.title
				if(options.canClose) {
					//comp.closeButton.visible = options.canClose
				}
				*/
			}

			ColumnLayout {
				id: menuColumn
				y: 50
				anchors.left: parent.left
				anchors.right: parent.right
				spacing: 10
			}
		}

		/*********************
		 * Main view
		 ********************/
		Rectangle {
			id: mainView
			color: "#00000000"

			anchors.right: parent.right
			anchors.left: menu.right
			anchors.bottom: parent.bottom
			anchors.top: parent.top

			function createView(component) {
				var view = component.createObject(mainView)

				return view;
			}
		}


	}


	/******************
	 * Dialogs
	 *****************/
	FileDialog {
		id: generalFileDialog
		property var callback;
		onAccepted: {
			var path = this.fileUrl.toString()
			callback.call(this, path)
		}
	}


	/******************
	 * Wallet functions
	 *****************/
	function importApp(path) {
		var ext = path.split('.').pop()
		if(ext == "html" || ext == "htm") {
			eth.openHtml(path)
		}else if(ext == "qml"){
			eth.openQml(path)
		}
	}

	function setWalletValue(value) {
		walletValueLabel.text = value
	}

	function loadPlugin(name) {
		console.log("Loading plugin" + name)
		mainView.addPlugin(name)
	}

	function setPeers(text) {
		peerLabel.text = text
	}

	function addPeer(peer) {
		// We could just append the whole peer object but it cries if you try to alter them
		peerModel.append({ip: peer.ip, port: peer.port, lastResponse:timeAgo(peer.lastSend), latency: peer.latency, version: peer.version})
	}

	function resetPeers(){
		peerModel.clear()
	}

	function timeAgo(unixTs){
		var lapsed = (Date.now() - new Date(unixTs*1000)) / 1000
		return  (lapsed + " seconds ago")
	}

	function convertToPretty(unixTs){
		var a = new Date(unixTs*1000);
		var months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
		var year = a.getFullYear();
		var month = months[a.getMonth()];
		var date = a.getDate();
		var hour = a.getHours();
		var min = a.getMinutes();
		var sec = a.getSeconds();
		var time = date+' '+month+' '+year+' '+hour+':'+min+':'+sec ;
		return time;
	}

	/**********************
	 * Windows
	 *********************/
	Window {
		id: peerWindow
		//flags: Qt.CustomizeWindowHint | Qt.Tool | Qt.WindowCloseButtonHint
		height: 200
		width: 700
		Rectangle {
			anchors.fill: parent
			property var peerModel: ListModel {
				id: peerModel
			}
			TableView {
				anchors.fill: parent
				id: peerTable
				model: peerModel
				TableViewColumn{width: 100; role: "ip" ; title: "IP" }
				TableViewColumn{width: 60; role: "port" ; title: "Port" }
				TableViewColumn{width: 140; role: "lastResponse"; title: "Last event" }
				TableViewColumn{width: 100; role: "latency"; title: "Latency" }
				TableViewColumn{width: 260; role: "version" ; title: "Version" }
			}
		}
	}

	Window {
		id: aboutWin
		visible: false
		title: "About"
		minimumWidth: 350
		maximumWidth: 350
		maximumHeight: 200
		minimumHeight: 200

		Image {
			id: aboutIcon
			height: 150
			width: 150
			fillMode: Image.PreserveAspectFit
			smooth: true
			source: "../facet.png"
			x: 10
			y: 10
		}

		Text {
			anchors.left: aboutIcon.right
			anchors.leftMargin: 10
			font.pointSize: 12
			text: "<h2>Ethereal - Adrastea</h2><br><h3>Development</h3>Jeffrey Wilcke<br>Maran Hidskes<br>Viktor Trón<br>"
		}
	}

	Window {
		id: txImportDialog
		minimumWidth: 270
		maximumWidth: 270
		maximumHeight: 50
		minimumHeight: 50
		TextField {
			id: txImportField
			width: 170
			anchors.verticalCenter: parent.verticalCenter
			anchors.left: parent.left
			anchors.leftMargin: 10
			onAccepted: {
			}
		}
		Button {
			anchors.left: txImportField.right
			anchors.verticalCenter: parent.verticalCenter
			anchors.leftMargin: 5
			text: "Import"
			onClicked: {
				eth.importTx(txImportField.text)
				txImportField.visible = false
			}
		}
		Component.onCompleted: {
			addrField.focus = true
		}
	}

	Window {
		id: addPeerWin
		visible: false
		minimumWidth: 230
		maximumWidth: 230
		maximumHeight: 50
		minimumHeight: 50

		TextField {
			id: addrField
			anchors.verticalCenter: parent.verticalCenter
			anchors.left: parent.left
			anchors.leftMargin: 10
			placeholderText: "address:port"
			onAccepted: {
				eth.connectToPeer(addrField.text)
				addPeerWin.visible = false
			}
		}
		Button {
			anchors.left: addrField.right
			anchors.verticalCenter: parent.verticalCenter
			anchors.leftMargin: 5
			text: "Add"
			onClicked: {
				eth.connectToPeer(addrField.text)
				addPeerWin.visible = false
			}
		}
		Component.onCompleted: {
			addrField.focus = true
		}
	}
}
