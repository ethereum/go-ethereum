import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var iconFile: "../new.png"
	property var title: "New transaction"

	objectName: "newTxView"
	visible: false
	anchors.fill: parent
	color: "#00000000"

	Column {
		id: mainContractColumn
		anchors.fill: parent


		states: [
			State{
				name: "ERROR"

				PropertyChanges { target: txResult; visible:true}
				PropertyChanges { target: codeView; visible:true}
			},
			State {
				name: "DONE"

				PropertyChanges { target: txValue; visible:false}
				PropertyChanges { target: txGas; visible:false}
				PropertyChanges { target: txGasPrice; visible:false}
				PropertyChanges { target: codeView; visible:false}
				PropertyChanges { target: txButton; visible:false}
				PropertyChanges { target: txDataLabel; visible:false}
				PropertyChanges { target: atLabel; visible:false}
				PropertyChanges { target: txFuelRecipient; visible:false}
				PropertyChanges { target: valueDenom; visible:false}
				PropertyChanges { target: gasDenom; visible:false}

				PropertyChanges { target: txResult; visible:true}
				PropertyChanges { target: txOutput; visible:true}
				PropertyChanges { target: newTxButton; visible:true}
			},
			State {
				name: "SETUP"

				PropertyChanges { target: txValue; visible:true; text: ""}
				PropertyChanges { target: txGas; visible:true;}
				PropertyChanges { target: txGasPrice; visible:true;}
				PropertyChanges { target: codeView; visible:true; text: ""}
				PropertyChanges { target: txButton; visible:true}
				PropertyChanges { target: txDataLabel; visible:true}
				PropertyChanges { target: valueDenom; visible:true}
				PropertyChanges { target: gasDenom; visible:true}

				PropertyChanges { target: txResult; visible:false}
				PropertyChanges { target: txOutput; visible:false}
				PropertyChanges { target: newTxButton; visible:false}
			}
		]
		width: 400
		spacing: 5
		anchors.left: parent.left
		anchors.top: parent.top
		anchors.leftMargin: 5
		anchors.topMargin: 5

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


		TextField {
			id: txFuelRecipient
			placeholderText: "Address / Name or empty for contract"
			//validator: RegExpValidator { regExp: /[a-f0-9]{40}/ }
			width: 400
		}

		RowLayout {
			TextField {
				id: txValue
				width: 222
				placeholderText: "Amount"
				validator: RegExpValidator { regExp: /\d*/ }
				onTextChanged: {
					contractFormReady()
				}
			}

			ComboBox {
				id: valueDenom
				currentIndex: 6
				model: denomModel
			}
		}

		RowLayout {
			TextField {
				id: txGas
				width: 50
				validator: RegExpValidator { regExp: /\d*/ }
				placeholderText: "Gas"
				text: "500"
			}
			Label {
				id: atLabel
				text: "@"
			}

			TextField {
				id: txGasPrice
				width: 200
				placeholderText: "Gas price"
				text: "10"
				validator: RegExpValidator { regExp: /\d*/ }
			}

			ComboBox {
				id: gasDenom
				currentIndex: 4
				model: denomModel
			}
		}

		Label {
			id: txDataLabel
			text: "Data"
		}

		TextArea {
			id: codeView
			height: 300
			anchors.topMargin: 5
			width: 400
			onTextChanged: {
				contractFormReady()
			}
		}


		Button {
			id: txButton
			/* enabled: false */
			states: [
				State {
					name: "READY"
					PropertyChanges { target: txButton; /*enabled: true*/}
				},
				State {
					name: "NOTREADY"
					PropertyChanges { target: txButton; /*enabled:false*/}
				}
			]
			text: "Send"
			onClicked: {
				var value = txValue.text + denomModel.get(valueDenom.currentIndex).zeros;
				var gasPrice = txGasPrice.text + denomModel.get(gasDenom.currentIndex).zeros;
				var res = gui.create(txFuelRecipient.text, value, txGas.text, gasPrice, codeView.text)
				if(res[1]) {
					txResult.text = "Your contract <b>could not</b> be sent over the network:\n<b>"
					txResult.text += res[1].error()
					txResult.text += "</b>"
					mainContractColumn.state = "ERROR"
				} else {
					txResult.text = "Your transaction has been submitted:\n"
					txOutput.text = res[0].address
					mainContractColumn.state = "DONE"
				}
			}
		}
		Text {
			id: txResult
			visible: false
		}
		TextField {
			id: txOutput
			visible: false
			width: 530
		}
		Button {
			id: newTxButton
			visible: false
			text: "Create a new transaction"
			onClicked: {
				this.visible = false
				txResult.text = ""
				txOutput.text = ""
				mainContractColumn.state = "SETUP"
			}
		}
	}

	function contractFormReady(){
		if(codeView.text.length > 0 && txValue.text.length > 0 && txGas.text.length > 0 && txGasPrice.length > 0) {
			txButton.state = "READY"
		}else{
			txButton.state = "NOTREADY"
		}
	}
}
