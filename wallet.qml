import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
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
			      onClicked: tester.compile(codeView)
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
		width: parent.width
		height: 100
		anchors.bottom: parent.bottom
		anchors.top: splitView.bottom
		TableViewColumn{ role: "number" ; title: "#" ; width: 100 }
		TableViewColumn{ role: "hash" ; title: "Hash" ; width: 560 }

		model: blockModel
	}


	statusBar: StatusBar {
			RowLayout {
				Label { text: "0.0.1" }
			}
	}

	function addBlock(block) {
			blockModel.append({number: block.number, hash: block.hash})
	}
}
