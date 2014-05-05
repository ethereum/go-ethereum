import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Component {
	id: newTransaction
	Column {
		id: simpleSendColumn
		states: [
			State{
				name: "ERROR"
			},
			State {
				name: "DONE"
				PropertyChanges { target: txSimpleValue; visible:false}
				PropertyChanges { target: txSimpleRecipient; visible:false}
				PropertyChanges { target:newSimpleTxButton; visible:false}

				PropertyChanges { target: txSimpleResult; visible:true}
				PropertyChanges { target: txSimpleOutput; visible:true}
				PropertyChanges { target:newSimpleTxButton; visible:true}
			},
			State {
				name: "SETUP"
				PropertyChanges { target: txSimpleValue; visible:true; text: ""}
				PropertyChanges { target: txSimpleRecipient; visible:true; text: ""}
				PropertyChanges { target: txSimpleButton; visible:true}
				PropertyChanges { target:newSimpleTxButton; visible:false}
			}
		]
		spacing: 5
		anchors.leftMargin: 5
		anchors.topMargin: 5
		anchors.top: parent.top
		anchors.left: parent.left

		function checkFormState(){
			if(txSimpleRecipient.text.length == 40 && txSimpleValue.text.length > 0) {
				txSimpleButton.state = "READY"
			}else{
				txSimpleButton.state = "NOTREADY"
			}
		}

		TextField {
			id: txSimpleRecipient
			placeholderText: "Recipient address"
			Layout.fillWidth: true
			validator: RegExpValidator { regExp: /[a-f0-9]{40}/ }
			width: 530
			onTextChanged: { checkFormState() }
		}
		TextField {
			id: txSimpleValue
			placeholderText: "Amount"
			anchors.rightMargin: 5
			validator: IntValidator { }
			onTextChanged: { checkFormState() }
		}
		Button {
			id: txSimpleButton
			/*enabled: false*/
			states: [
				State {
					name: "READY"
					PropertyChanges { target: txSimpleButton; /*enabled: true*/}
				},
				State {
					name: "NOTREADY"
					PropertyChanges { target: txSimpleButton; /*enabled: false*/}
				}
			]
			text: "Send"
			onClicked: {
				//this.enabled = false
				var res = eth.createTx(txSimpleRecipient.text, txSimpleValue.text,"","","")
				if(res[1]) {
					txSimpleResult.text = "There has been an error broadcasting your transaction:" + res[1].error()
				} else {
					txSimpleResult.text = "Your transaction has been broadcasted over the network.\nYour transaction id is:"
					txSimpleOutput.text = res[0]
					this.visible = false
					simpleSendColumn.state = "DONE"
				}
			}
		}
		Text {
			id: txSimpleResult
			visible: false

		}
		TextField {
			id: txSimpleOutput
			visible: false
			width: 530
		}
		Button {
			id: newSimpleTxButton
			visible: false
			text: "Create an other transaction"
			onClicked: {
				this.visible = false
				simpleSendColumn.state = "SETUP"
			}
		}
	}
}
