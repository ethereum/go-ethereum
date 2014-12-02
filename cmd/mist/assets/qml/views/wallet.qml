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
	property var iconSource: "../facet.png"
	property var menuItem

	objectName: "walletView"
	anchors.fill: parent

	function onReady() {
		setBalance()
	}

	function setBalance() {
		balance.text = "<b>Balance</b>: " + eth.numberToHuman(eth.balanceAt(eth.key().address))
		if(menuItem)
			menuItem.secondaryTitle = eth.numberToHuman(eth.balanceAt(eth.key().address))
	}

	ListModel {
		id: denomModel
		ListElement { text: "Wei" ;     zeros: "" }
		ListElement { text: "Ada" ;     zeros: "000" }
		ListElement { text: "Babbage" ; zeros: "000000" }
		ListElement { text: "Shannon" ; zeros: "000000000" }
		ListElement { text: "Szabo" ;   zeros: "000000000000" }
		ListElement { text: "Finney" ;  zeros: "000000000000000" }
		ListElement { text: "Ether" ;   zeros: "000000000000000000" }
		ListElement { text: "Einstein" ;zeros: "000000000000000000000" }
		ListElement { text: "Douglas" ; zeros: "000000000000000000000000000000000000000000" }
	}

	ColumnLayout {
		spacing: 10
		y: 40
		anchors.fill: parent

		Text {
			id: balance
			font.pixelSize: 24
			anchors {
				horizontalCenter: parent.horizontalCenter
				top: parent.top
				topMargin: 20
			}
		}

		Rectangle {
			id: newTxPane
			color: "#ececec"
			border.color: "#cccccc"
			border.width: 1
			anchors {
				top: balance.bottom
				topMargin: 10
				left: parent.left
				leftMargin: 5
				right: parent.right
				rightMargin: 5
			}
			height: 100

			RowLayout {
				id: amountFields
				spacing: 10
				anchors {
					top: parent.top
					topMargin: 20
					left: parent.left
					leftMargin: 20
				}

				Text {
					text: "Ξ  "
				}

				// There's something off with the row layout where textfields won't listen to the width setting
				Rectangle {
					width: 50
					height: 20
					TextField {
						id: txValue
						width: parent.width
						placeholderText: "0.00"
					}
				}

				ComboBox {
					id: valueDenom
					currentIndex: 6
					model: denomModel
				}

			}

			RowLayout {
				id: toFields
				spacing: 10
				anchors {
					top: amountFields.bottom
					topMargin: 5
					left: parent.left
					leftMargin: 20
				}

				Text {
					text: "To"
				}

				Rectangle {
					width: 200
					height: 20
					TextField {
						id: txTo
						width: parent.width
						placeholderText: "Address or name"
					}
				}

				Button {
					text: "Send"
					onClicked: {
						var value = txValue.text + denomModel.get(valueDenom.currentIndex).zeros;
						var gasPrice = "10000000000000"
						var res = eth.transact({from: eth.key().privateKey, to: txTo.text, value: value, gas: "500", gasPrice: gasPrice})
					}
				}
			}
		}

		Rectangle {
			anchors {
				left: parent.left
				right: parent.right
				top: newTxPane.bottom
				topMargin: 10
				bottom: parent.bottom
			}
			TableView {
				id: txTableView
				anchors.fill : parent
				TableViewColumn{ role: "num" ; title: "#" ; width: 30 }
				TableViewColumn{ role: "from" ; title: "From" ; width: 280 }
				TableViewColumn{ role: "to" ; title: "To" ; width: 280 }
				TableViewColumn{ role: "value" ; title: "Amount" ; width: 100 }

				model: ListModel {
					id: txModel
					Component.onCompleted: {
						var filter = ethx.watch({latest: -1, from: eth.key().address});
						filter.changed(addTxs)

						addTxs(filter.messages())
					}

					function addTxs(messages) {
						setBalance()

						for(var i = 0; i < messages.length; i++) {
							var message = messages.get(i);
							var to = eth.lookupName(message.to);
							var from = eth.lookupName(message.from);
							txModel.insert(0, {num: txModel.count, from: from, to: to, value: eth.numberToHuman(message.value)})
						}
					}
				}
			}
		}

	}
}
