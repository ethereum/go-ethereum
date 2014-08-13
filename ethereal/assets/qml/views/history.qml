import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var iconFile: "../tx.png"
	property var title: "Transactions"

	property var txModel: ListModel {
		id: txModel
	}

	id: historyView
	anchors.fill: parent
	objectName: "transactionView"

	TableView {
		id: txTableView
		anchors.fill: parent
		TableViewColumn{ role: "inout" ; title: "" ; width: 40 }
		TableViewColumn{ role: "value" ; title: "Value" ; width: 100 }
		TableViewColumn{ role: "address" ; title: "Address" ; width: 430 }
		TableViewColumn{ role: "contract" ; title: "Contract" ; width: 100 }

		model: txModel
	}

	function addTx(tx, inout) {
		var isContract
		if (tx.contract == true){
			isContract = "Yes"
		}else{
			isContract = "No"
		}


		var address;
		if(inout == "recv") {
			address = tx.sender;
		} else {
			address = tx.address;
		}

		txModel.insert(0, {inout: inout, hash: tx.hash, address: address, value: tx.value, contract: isContract})
	}
}
