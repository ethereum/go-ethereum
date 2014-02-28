import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

ApplicationWindow {
	id: root

	width: 900
	height: 600
	minimumHeight: 300

	title: "Ethereal"

	MenuBar {
		Menu {
			title: "File"
			MenuItem {
				text: "Import App"
				shortcut: "Ctrl+o"
				onTriggered: openAppDialog.open()
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
				text: "Start"
				onTriggered: ui.connect()
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


	property var blockModel: ListModel {
		id: blockModel
	}

	function setView(view) {
		networkView.visible = false
		historyView.visible = false
		newTxView.visible = false
		view.visible = true
		//root.title = "Ethereal - " = view.title
	}

	SplitView {
		anchors.fill: parent
		resizing: false

		Rectangle {
			id: menu
			Layout.minimumWidth: 80
			Layout.maximumWidth: 80
			anchors.bottom: parent.bottom
			anchors.top: parent.top
			//color: "#D9DDE7"
			color: "#252525"

			ColumnLayout {
				y: 50
				anchors.left: parent.left
				anchors.right: parent.right
				height: 200
				Image {
					source: "tx.png"
					anchors.horizontalCenter: parent.horizontalCenter
					MouseArea {
						anchors.fill: parent
						onClicked: {
							setView(historyView)
						}
					}
				}
				Image {
					source: "new.png"
					anchors.horizontalCenter: parent.horizontalCenter
					MouseArea {
						anchors.fill: parent
						onClicked: {
							setView(newTxView)
						}
					}
				}
				Image {
					source: "net.png"
					anchors.horizontalCenter: parent.horizontalCenter
					MouseArea {
						anchors.fill: parent
						onClicked: {
							setView(networkView)
						}
					}
				}
			}
						
		}

		property var txModel: ListModel {
			id: txModel
		}

		Rectangle {
			id: historyView
			property var title: "Transactions"
			anchors.right: parent.right
			anchors.left: menu.right
			anchors.bottom: parent.bottom
			anchors.top: parent.top
			TableView {
				id: txTableView
				anchors.fill: parent
				TableViewColumn{ role: "value" ; title: "Value" ; width: 100 }
				TableViewColumn{ role: "address" ; title: "Address" ; width: 430 }

				model: txModel
			}
		}

		Rectangle {
			id: newTxView
			property var title: "New transaction"
			visible: false
			anchors.right: parent.right
			anchors.left: menu.right
			anchors.bottom: parent.bottom
			anchors.top: parent.top
			color: "#00000000"

			ColumnLayout {
				width: 400
				anchors.left: parent.left
				anchors.top: parent.top
				anchors.leftMargin: 5
				anchors.topMargin: 5
				TextField {
					id: txAmount
					width: 200
					placeholderText: "Amount"
				}

				TextField {
					id: txReceiver
					placeholderText: "Receiver Address (or empty for contract)"
					Layout.fillWidth: true
				}

				Label {
					text: "Transaction data"
				}
				TextArea {
					id: codeView
					anchors.topMargin: 5
					Layout.fillWidth: true
					width: parent.width /2 
				}

				Button {
					text: "Send"
					onClicked: {
						console.log(eth.createTx(txReceiver.text, txAmount.text, codeView.text))
					}
				}
			}
		}


		Rectangle {
			id: networkView
			property var title: "Network"
			visible: false
			anchors.right: parent.right
			anchors.bottom: parent.bottom
			anchors.top: parent.top

			TableView {
				id: blockTable
				width: parent.width
				anchors.top: parent.top
				anchors.bottom: logView.top
				TableViewColumn{ role: "number" ; title: "#" ; width: 100 }
				TableViewColumn{ role: "hash" ; title: "Hash" ; width: 560 }

				model: blockModel

				onDoubleClicked: {
					popup.visible = true
					popup.block = eth.getBlock(blockModel.get(row).hash)
					popup.hashLabel.text = popup.block.hash
				}
			}

			property var logModel: ListModel {
				id: logModel
			}

			TableView {
				id: logView
				width: parent.width
				height: 150
				anchors.bottom: parent.bottom
				TableViewColumn{ role: "description" ; title: "log" }

				model: logModel
			}
		}
	}

	FileDialog {
		id: openAppDialog
		title: "Open QML Application"
		onAccepted: {
			ui.open(openAppDialog.fileUrl.toString())
		}
	}

	statusBar: StatusBar {
		RowLayout {
			anchors.fill: parent
			Button {
				property var enabled: true
				id: connectButton
				onClicked: {
					if(this.enabled) {
						ui.connect(this)
					}
				}
				text: "Connect"
			}
			Button {
				id: importAppButton
				anchors.left: connectButton.right
				anchors.leftMargin: 5
				onClicked: openAppDialog.open()
				text: "Import App"
			}

			Label {
				anchors.left: importAppButton.right
				anchors.leftMargin: 5
				id: walletValueLabel
			}

			Label {
				anchors.right: peerImage.left
				anchors.rightMargin: 5
				id: peerLabel
				font.pixelSize: 8
				text: "0 / 0"
			}
			Image {
				id: peerImage
				anchors.right: parent.right
				width: 10; height: 10
				source: "network.png"
			}
		}
	}

	Window {
		id: popup
		visible: false
		property var block
		Label {
			id: hashLabel
			anchors.horizontalCenter: parent.horizontalCenter
			anchors.verticalCenter: parent.verticalCenter
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
		}
		Button {
			anchors.left: addrField.right
			anchors.verticalCenter: parent.verticalCenter
			anchors.leftMargin: 5
			text: "Add"
			onClicked: {
				ui.connectToPeer(addrField.text)
				addPeerWin.visible = false
			}
		}
	}

	Window {
		id: aboutWin
		visible: false
		title: "About"
		minimumWidth: 300
		maximumWidth: 300
		maximumHeight: 200
		minimumHeight: 200

		Text {
			font.pointSize: 18
			text: "Eth Go"
		}

	}

	function setWalletValue(value) {
		walletValueLabel.text = value
	}

	function addTx(tx) {
		txModel.insert(0, {hash: tx.hash, address: tx.address, value: tx.value})
	}

	function addBlock(block) {
		blockModel.insert(0, {number: block.number, hash: block.hash})
	}

	function addLog(str) {
		if(str.len != 0) {
			logModel.append({description: str})
		}
	}

	function setPeers(text) {
		peerLabel.text = text
	}
}
