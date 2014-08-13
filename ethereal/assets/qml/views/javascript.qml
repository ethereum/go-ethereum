import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var title: "JavaScript"
	property var iconFile: "../tx.png"

	objectName: "javascriptView"
	visible: false
	anchors.fill: parent

	TextField {
		id: input
		anchors {
			left: parent.left
			right: parent.right
			bottom: parent.bottom
		}
		height: 20

		Keys.onReturnPressed: {
			var res = eth.evalJavascriptString(this.text);
			this.text = "";

			output.append(res)
		}
	}

	TextArea {
		id: output
		verticalAlignment: TextEdit.AlignBottom
		text: "> JSRE Ready..."
		anchors {
			top: parent.top
			left: parent.left
			right: parent.right
			bottom: input.top
		}
	}
}
