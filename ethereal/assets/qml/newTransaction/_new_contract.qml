import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Component {
	id: newContract
	Column {
		id: mainContractColumn
		function contractFormReady(){
			if(codeView.text.length > 0 && txValue.text.length > 0 && txGas.text.length > 0 && txGasPrice.length > 0) {
				txButton.state = "READY"
			}else{
				txButton.state = "NOTREADY"
			}
		}
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

				PropertyChanges { target: txResult; visible:true}
				PropertyChanges { target: txOutput; visible:true}
				PropertyChanges { target: newTxButton; visible:true}
			},
			State {
				name: "SETUP"
				PropertyChanges { target: txValue; visible:true; text: ""}
				PropertyChanges { target: txGas; visible:true; text: ""}
				PropertyChanges { target: txGasPrice; visible:true; text: ""}
				PropertyChanges { target: codeView; visible:true; text: ""}
				PropertyChanges { target: txButton; visible:true}
				PropertyChanges { target: txDataLabel; visible:true}

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

		TextField {
			id: txValue
			width: 200
			placeholderText: "Amount"
			validator: RegExpValidator { regExp: /\d*/ }
			onTextChanged: {
				contractFormReady()
			}
		}
		TextField {
			id: txGas
			width: 200
			validator: RegExpValidator { regExp: /\d*/ }
			placeholderText: "Gas"
			onTextChanged: {
				contractFormReady()
			}
		}
		TextField {
			id: txGasPrice
			width: 200
			placeholderText: "Gas price"
			validator: RegExpValidator { regExp: /\d*/ }
			onTextChanged: {
				contractFormReady()
			}
		}

		Row {
			id: rowContract
			ExclusiveGroup { id: contractTypeGroup }
			RadioButton {
				id: createContractRadio
				text: "Create contract"
				checked: true
				exclusiveGroup: contractTypeGroup
				onClicked: {
					txFuelRecipient.visible = false
					txDataLabel.text = "Contract code"
				}
			}
			RadioButton {
				id: runContractRadio
				text: "Run contract"
				exclusiveGroup: contractTypeGroup
				onClicked: {
					txFuelRecipient.visible = true
					txDataLabel.text = "Contract arguments"
				}
			}
		}


		Label {
			id: txDataLabel
			text: "Contract code"
		}

		TextArea {
			id: codeView
			height: 300
			anchors.topMargin: 5
			Layout.fillWidth: true
			width: parent.width /2
			onTextChanged: {
				contractFormReady()
			}
		}

		TextField {
			id: txFuelRecipient
			placeholderText: "Contract address"
			validator: RegExpValidator { regExp: /[a-f0-9]{40}/ }
			visible: false
			width: 530
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
				//this.enabled = false
				var res = eth.create(txFuelRecipient.text, txValue.text, txGas.text, txGasPrice.text, codeView.text)
				if(res[1]) {
					txResult.text = "Your contract <b>could not</b> be send over the network:\n<b>"
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
			text: "Create an other contract"
			onClicked: {
				this.visible = false
				txResult.text = ""
				txOutput.text = ""
				mainContractColumn.state = "SETUP"
			}
		}

		Button {
			id: debugButton
			text: "Debug"
			onClicked: {
				var res = ui.debugTx("", txValue.text, txGas.text, txGasPrice.text, codeView.text)
				debugWindow.visible = true
			}
		}
	}
}
