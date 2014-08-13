import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var title: "Information"
	property var iconFile: "../heart.png"

	objectName: "infoView"
	visible: false
	anchors.fill: parent

	color: "#00000000"

	Column {
		spacing: 3
		anchors.fill: parent
		anchors.topMargin: 5
		anchors.leftMargin: 5

		Label {
			id: addressLabel
			text: "Address"
		}
		TextField {
			text: pub.getKey().address
			width: 500
		}

		Label {
			text: "Client ID"
		}
		TextField {
			text: gui.getCustomIdentifier()
			width: 500
			placeholderText: "Anonymous"
			onTextChanged: {
				gui.setCustomIdentifier(text)
			}
		}
	}

	property var addressModel: ListModel {
		id: addressModel
	}
	TableView {
		id: addressView
		width: parent.width - 200
		height: 200
		anchors.bottom: logLayout.top
		TableViewColumn{ role: "name"; title: "name" }
		TableViewColumn{ role: "address"; title: "address"; width: 300}

		model: addressModel
	}

	Rectangle {
		anchors.top: addressView.top
		anchors.left: addressView.right
		anchors.leftMargin: 20

		TextField {
			placeholderText: "Name to register"
			id: nameToReg
			width: 150
		}

		Button {
			anchors.top: nameToReg.bottom
			text: "Register"
			MouseArea{
				anchors.fill: parent
				onClicked: {
					gui.registerName(nameToReg.text)
					nameToReg.text = ""
				}
			}
		}
	}

	property var logModel: ListModel {
		id: logModel
	}
	RowLayout {
		id: logLayout
		width: parent.width
		height: 200
		anchors.bottom: parent.bottom
		TableView {
			id: logView
			headerVisible: false
			anchors {
				right: logLevelSlider.left
				left: parent.left
				bottom: parent.bottom
				top: parent.top
			}

			TableViewColumn{ role: "description" ; title: "log" }

			model: logModel
		}

		Slider {
			id: logLevelSlider
			value: gui.getLogLevelInt()
			anchors {
				right: parent.right
				top: parent.top
				bottom: parent.bottom

				rightMargin: 5
				leftMargin: 5
				topMargin: 5
				bottomMargin: 5
			}

			orientation: Qt.Vertical
			maximumValue: 5
			stepSize: 1

			onValueChanged: {
				gui.setLogLevel(value)
			}
		}
	}

	function addDebugMessage(message){
		debuggerLog.append({value: message})
	}

	function addAddress(address) {
		addressModel.append({name: address.name, address: address.address})
	}

	function clearAddress() {
		addressModel.clear()
	}

	function addLog(str) {
		// Remove first item once we've reached max log items
		if(logModel.count > 250) {
			logModel.remove(0)
		}

		if(str.len != 0) {
			if(logView.flickableItem.atYEnd) {
				logModel.append({description: str})
				logView.positionViewAtRow(logView.rowCount - 1, ListView.Contain)
			} else {
				logModel.append({description: str})
			}
		}

	}
}
