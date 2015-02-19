
import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	id: root
	property var title: "Whisper Traffic"
	property var menuItem

	objectName: "whisperView"
	anchors.fill: parent

	property var identity: ""
	Component.onCompleted: {
		identity = shh.newIdentity()

		var t = shh.watch({}, root)
	}

	function onShhMessage(message, i) {
		whisperModel.insert(0, {from: message.from, payload: eth.toAscii(message.payload)})
	}

	RowLayout {
		id: input
		anchors {
			left: parent.left
			leftMargin: 20
			top: parent.top
			topMargin: 20
		}

		TextField {
			id: to
			placeholderText: "To"
		}
		TextField {
			id: data
			placeholderText: "Data"
		}
		TextField {
			id: topics
			placeholderText: "topic1, topic2, topic3, ..."
		}
		Button {
			text: "Send"
			onClicked: {
				shh.post([eth.toHex(data.text)], "", identity, topics.text.split(","), 500, 50)
			}
		}
	}

	TableView {
		id: txTableView
		anchors {
			top: input.bottom
			topMargin: 10
			bottom: parent.bottom
			left: parent.left
			right: parent.right
		}
		TableViewColumn{ id: fromRole; role: "from" ; title: "From"; width: 300 }
		TableViewColumn{ role: "payload" ; title: "Payload" ; width: parent.width -  fromRole.width - 2 }

		model: ListModel {
			id: whisperModel
		}
	}
}
