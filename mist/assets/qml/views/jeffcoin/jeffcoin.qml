import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1

Rectangle {
	id: root
	property var title: "JeffCoin"
	property var iconSource: "./views/jeffcoin/jeff.png"
	property var menuItem
	property var filter
	property var address: "fc0a9436890478bb9b1c6ed7455c2535366f4a99"

	function insertTx(message, blockNumber) {
		if(!message) return;

		var from = message.from
		var to = message.input.substr(24, 40)
		var value = eth.fromNumber(message.input.substr(64, 64))

		var me = eth.key().address;
		if((to == me|| from == me) && message.input.length == 128) {
			var to = eth.lookupName(to)
			var from = eth.lookupName(from)
			txModel.insert(0, {confirmations: blockNumber - message.number, from: from, to: to, value: value})
		}
	}

	function setBalance() {
		var jeffCoinAmount = eth.fromNumber(eth.storageAt(address, eth.key().address)) + " JΞF"
		menuItem.secondaryTitle = jeffCoinAmount

		balance.text = "<b>Balance</b>: " + jeffCoinAmount;
	}

	function onReady() {
		setBalance()

		filter = new ethx.watch({latest: -1, to: address})
		filter.changed(function(messages) {
			setBalance()

			var blockNumber = eth.block(-1).number;
			for(var i = 0; i < messages.length; i++) {
				insertTx(messages.get(i), blockNumber);
			}
		});

		var blockNumber = eth.block(-1).number;
		var msgs = filter.messages()
		for(var i = msgs.length-1; i >= 0; i--) {
			var message = JSON.parse(msgs.getAsJson(i))

			insertTx(message, blockNumber)
		}

		var chainChanged = ethx.watch("chain")
		chainChanged.changed(function() {
			for(var i = 0; i < txModel.count; i++) {
				var entry = txModel.get(i);
				entry.confirmations++;
			}
		});
	}

	function onDestroy() {
		filter.uninstall()
	}

	ColumnLayout {
		spacing: 10
		y: 40
		anchors.fill: parent

		Text {
			id: balance
			text: "<b>Balance</b>: " + eth.fromNumber(eth.storageAt(address, eth.key().address)) + " JΞF"
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
					text: "JΞF  "
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
						var lookup = eth.lookupAddress(address)
						if(lookup.length == 0)
							lookup = address

						eth.transact({from: eth.key().privateKey, to:lookup, gas: "9000", gasPrice: "10000000000000", data: ["0x"+txTo.text, txValue.text]})
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
				TableViewColumn{ role: "value" ; title: "Amount" ; width: 100 }
				TableViewColumn{ role: "from" ; title: "From" ; width: 280 }
				TableViewColumn{ role: "to" ; title: "To" ; width: 280 }
				TableViewColumn{ role: "confirmations" ; title: "Confirmations" ; width: 100 }

				model: ListModel {
					id: txModel
					Component.onCompleted: {
					}
				}
			}
		}
	}
}
