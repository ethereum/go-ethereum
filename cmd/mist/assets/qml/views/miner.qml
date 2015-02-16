import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	id: root
	property var title: "Miner"
	property var iconSource: "../miner.png"
	property var menuItem

	color: "#00000000"

        Label {
	    visible: false
            id: lastBlockLabel
            objectName: "lastBlockLabel"
            text: "---"
	    onTextChanged: {
		//menuItem.secondaryTitle = text
	    }
        }

        Label {
            objectName: "miningLabel"
            visible: false
            font.pixelSize: 10
            anchors.right: lastBlockLabel.left
            anchors.rightMargin: 5
	    onTextChanged: {
		menuItem.secondaryTitle = text
	    }
        }

	ColumnLayout {
		spacing: 10
		anchors.fill: parent

		Rectangle {
			id: mainPane
			color: "#00000000"
			anchors {
				top: parent.top
				bottom: localTxPane.top
				left: parent.left
				right: parent.right
			}

			Rectangle {
				id: menu
				height: 25
				anchors {
					left: parent.left
				}

				RowLayout {
					id: tools
					anchors {
						left: parent.left
						right: parent.right
					}

					Button {
						text: "Start"
						onClicked: {
							eth.setGasPrice(minGasPrice.text || "10000000000000");
							eth.setExtra(blockExtra.text)
							if (eth.toggleMining()) {
								this.text = "Stop";
							} else {
								this.text = "Start";
							}
						}
					}

					Rectangle {
						id: minGasPriceRect
						anchors.top: parent.top
						anchors.topMargin: 2
						width: 200
						TextField {
							id: minGasPrice
							placeholderText: "Min Gas: 10000000000000"
							width: 200
							validator: RegExpValidator { regExp: /\d*/ }
						}
					}

					Rectangle {
						width: 300
						anchors {
							left: minGasPriceRect.right
							leftMargin: 5
							top: parent.top
							topMargin: 2
						}

						TextField {
							id: blockExtra
							placeholderText: "Extra"
							width: parent.width
							maximumLength: 1024
						}
					}
				}
			}

			Column {
				anchors {
					left: parent.left
					right: parent.right
					top: menu.bottom
					topMargin: 5
				}

				Text {
					text: "<b>Merged mining options</b>"
				}

				TableView {
					id: mergedMiningTable
					height: 300
					anchors {
						left: parent.left
						right: parent.right
					}
					Component {
						id: checkBoxDelegate

						Item {
							id: test
							CheckBox {
								anchors.fill: parent
								checked: styleData.value

								onClicked: {
									var model = mergedMiningModel.get(styleData.row)

									if (this.checked) {
										model.id = txModel.createLocalTx(model.address, "0", "5000", "0", "")
									} else {
										txModel.removeWithId(model.id);
										model.id = 0;
									}
								}
							}
						}
					}
					TableViewColumn{ role: "checked" ; title: "" ; width: 40 ; delegate: checkBoxDelegate }
					TableViewColumn{ role: "name" ; title: "Name" ; width: 480 }
					model: ListModel {
						objectName: "mergedMiningModel"
						id: mergedMiningModel 
						function addMergedMiningOption(model) {
							this.append(model);
						}
					}
					Component.onCompleted: {
						/*
						// XXX Temp. replace with above eventually
						var tmpItems = ["JEVCoin", "Some coin", "Other coin", "Etc coin"];
						var address = "e6716f9544a56c530d868e4bfbacb172315bdead";
						for (var i = 0; i < tmpItems.length; i++) {
							mergedMiningModel.append({checked: false, name: tmpItems[i], address: address, id: 0, itemId: i});
						}
						*/
					}
				}
			}
		}

		Rectangle {
			id: localTxPane
			color: "#ececec"
			border.color: "#cccccc"
			border.width: 1
			anchors {
				left: parent.left
				right: parent.right
				bottom: parent.bottom
			}
			height: 300

			ColumnLayout {
				spacing: 10
				anchors.fill: parent
				RowLayout {
					id: newLocalTx
					anchors {
						left: parent.left
						leftMargin: 5
						top: parent.top
						topMargin: 5
						bottomMargin: 5
					}

					Text {
						text: "Local tx"
					}

					Rectangle {
						width: 250
						color: "#00000000"
						anchors.top: parent.top
						anchors.topMargin: 2

						TextField {
							id: to
							placeholderText: "To"
							width: 250
							validator: RegExpValidator { regExp: /[abcdefABCDEF1234567890]*/ }
						}
					}
					TextField {
						property var defaultGas: "5000"
						id: gas
						placeholderText: "Gas"
						text: defaultGas
						validator: RegExpValidator { regExp: /\d*/ }
					}
					TextField {
						id: gasPrice
						placeholderText: "Price"
						validator: RegExpValidator { regExp: /\d*/ }
					}
					TextField {
						id: value 
						placeholderText: "Amount"
						text: "0"
						validator: RegExpValidator { regExp: /\d*/ }
					}
					TextField {
						id: data
						placeholderText: "Data"
						validator: RegExpValidator { regExp: /[abcdefABCDEF1234567890]*/ }
					}
					Button {
						text: "Create"
						onClicked: {
							if (to.text.length == 40 && gasPrice.text.length != 0 && value.text.length != 0 && gas.text.length != 0) {
								txModel.createLocalTx(to.text, gasPrice.text, gas.text, value.text, data.text);

								to.text = ""; gasPrice.text = "";
								gas.text = gas.defaultGas;
								value.text = "0"
							}
						}
					}
				}

				TableView {
					id: txTableView
					anchors {
						top: newLocalTx.bottom
						topMargin: 5
						left: parent.left
						right: parent.right
						bottom: parent.bottom
					}
					TableViewColumn{ role: "to" ; title: "To" ; width: 480 }
					TableViewColumn{ role: "gas" ; title: "Gas" ; width: 100 }
					TableViewColumn{ role: "gasPrice" ; title: "Gas Price" ; width: 100 }
					TableViewColumn{ role: "value" ; title: "Amount" ; width: 100 }
					TableViewColumn{ role: "data" ; title: "Data" ; width: 100 }

					model: ListModel {
						id: txModel
						Component.onCompleted: {
						}
						function removeWithId(id) {
							for (var i = 0; i < this.count; i++) {
								if (txModel.get(i).id == id) {
									this.remove(i);
									eth.removeLocalTransaction(id)
									break;
								}
							}
						}

						function createLocalTx(to, gasPrice, gas, value, data) {
							var id = eth.addLocalTransaction(to, data, gas, gasPrice, value)
							txModel.insert(0, {to: to, gas: gas, gasPrice: gasPrice, value: value, data: data, id: id});

							return id
						}
					}
				}
			}
		}
	}
}
