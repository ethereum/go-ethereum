import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import GoExtensions 1.0

ApplicationWindow {
	id: root

	width: 800
	height: 600
	minimumHeight: 300

	title: "Ethereal"


	toolBar: ToolBar {
		id: mainToolbar

		RowLayout {
			width: parent.width
			Button {
			      text: "Send"
			      onClicked: console.log("SEND")
			}

			TextField {
			      width: 200
			      placeholderText: "Amount"
			}

			TextField {
			      width: 300
			      placeholderText: "Receiver Address (or empty for contract)"
			      Layout.fillWidth: true
			}

		}
	}

	SplitView {
		id: splitView
		height: 200
		anchors.top: parent.top
		anchors.right: parent.right
		anchors.left: parent.left

		TextArea {
			      id: codeView
			      width: parent.width /2 
		}

		TextArea {
			      readOnly: true
		}
	}

	property var blockModel: ListModel {
		id: blockModel
	}

	TableView {
		id: blockTable
		width: parent.width
		anchors.top: splitView.bottom
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
				id: connectButton
				onClicked: ui.connect()
				text: "Connect"
			}
			Button {
				anchors.left: connectButton.right
				anchors.leftMargin: 5
				onClicked: openAppDialog.open()
				text: "Import App"
			}

			Label { text: "0.0.1" }
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

	function addBlock(block) {
		blockModel.insert(0, {number: block.number, hash: block.hash})
	}

	function addLog(str) {
		console.log(str)
		logModel.insert(0, {description: str})
	}

	function setPeers(text) {
		peerLabel.text = text
	}
}
