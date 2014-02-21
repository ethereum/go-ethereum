import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import GoExtensions 1.0

ApplicationWindow {
	minimumWidth: 500
	maximumWidth: 500
	maximumHeight: 100
	minimumHeight: 100

	title: "Ethereum Dice"

	TextField {
		id: textField
		anchors.verticalCenter: parent.verticalCenter
		anchors.horizontalCenter: parent.horizontalCenter
		placeholderText: "Amount"
	}
	Label {
		id: txHash
		anchors.bottom: textField.top
		anchors.bottomMargin: 5
		anchors.horizontalCenter: parent.horizontalCenter
	}
	Button {
		anchors.top: textField.bottom
		anchors.horizontalCenter: parent.horizontalCenter
		anchors.topMargin: 5
		text: "Place bet"
		onClicked: {
			txHash.text = eth.createTx("e6716f9544a56c530d868e4bfbacb172315bdead", parseInt(textField.text))
		}
	}
}
