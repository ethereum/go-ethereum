import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	id: root
	property var title: "Wallet"
	property var iconFile: "../wallet.png"
	property var menuItem

	objectName: "walletView"
	anchors.fill: parent

	function onReady() {
		menuItem.secondary = eth.numberToHuman(eth.balanceAt(eth.key().address))

	}

	ColumnLayout {
		spacing: 10
		y: 40
		anchors {
			left: parent.left
			right: parent.right
		}

		Text {
			text: "<b>Balance</b>: " + eth.numberToHuman(eth.balanceAt(eth.key().address))
			font.pixelSize: 24
			anchors {
				horizontalCenter: parent.horizontalCenter
			}
		}

		TableView {
			id: txTableView
			anchors {
				left: parent.left
				right: parent.right
			}
			TableViewColumn{ role: "num" ; title: "#" ; width: 30 }
			TableViewColumn{ role: "from" ; title: "From" ; width: 280 }
			TableViewColumn{ role: "to" ; title: "To" ; width: 280 }
			TableViewColumn{ role: "value" ; title: "Amount" ; width: 100 }

			model: ListModel {
				id: txModel
				Component.onCompleted: {
					var messages = JSON.parse(eth.messages({latest: -1, from: "e6716f9544a56c530d868e4bfbacb172315bdead"}))
					for(var i = 0; i < messages.length; i++) {
						var message = messages[i];
						this.insert(0, {num: i, from: message.from, to: message.to, value: eth.numberToHuman(message.value)})
					}
				}
			}
		}

	}
}
