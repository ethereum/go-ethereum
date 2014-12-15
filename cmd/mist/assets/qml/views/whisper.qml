
import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	id: root
	property var title: "Whisper"
	property var iconSource: "../facet.png"
	property var menuItem

	objectName: "whisperView"
	anchors.fill: parent

	property var identity: ""
	Component.onCompleted: {
		identity = shh.newIdentity()
		console.log("New identity:", identity)
	}

	RowLayout {
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
				shh.post(eth.toHex(data.text), "", identity, topics.text.split(","), 500, 50)
			}
		}
	}
}
