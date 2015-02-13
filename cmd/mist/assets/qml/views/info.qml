import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var title: "Debug Info"
	property var menuItem

	objectName: "infoView"
	visible: false
	anchors.fill: parent

	color: "#00000000"

	Column {
		id: info
		spacing: 3
		anchors.fill: parent
		anchors.topMargin: 5
		anchors.leftMargin: 5

		Label {
			id: addressLabel
			text: "Address"
		}
		TextField {
			text: eth.coinbase()
			width: 500
		}

		TextArea {
			objectName: "statsPane"
			width: parent.width
			height: 200
			selectByMouse: true
			readOnly: true
			font.family: "Courier"
		}
	}

	RowLayout {
		id: logLayout
		width: parent.width
		height: 200
		anchors.bottom: parent.bottom

		TableView {
			id: addressView
			width: parent.width
			height: 200
			anchors {
				left: parent.left
				right: logLevelSlider.left
				bottom: parent.bottom
				top: parent.top
			}
			TableViewColumn{ role: "name"; title: "name" }
			TableViewColumn{ role: "address"; title: "address"; width: 300}

			property var addressModel: ListModel {
				id: addressModel
			}

			model: addressModel
			itemDelegate: Item {
				Text {
					anchors {
						left: parent.left
						right: parent.right
						leftMargin: 10
						verticalCenter: parent.verticalCenter
					}
					color: styleData.textColor
					elide: styleData.elideMode
					text: styleData.value
					font.pixelSize: 11
					MouseArea {
						acceptedButtons: Qt.LeftButton | Qt.RightButton
						propagateComposedEvents: true
						anchors.fill: parent
						onClicked: {
							addressView.selection.clear()
							addressView.selection.select(styleData.row)

							if(mouse.button == Qt.RightButton) {
								contextMenu.row = styleData.row;
								contextMenu.popup()
							}
						}
					}
				}
			}

			Menu {
				id: contextMenu
				property var row;

				MenuItem {
					text: "Copy"
					onTriggered: {
						copyToClipboard(addressModel.get(this.row).address)
					}
				}
			}
		}

		/*
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
		*/

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

	property var logModel: ListModel {
		id: logModel
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
