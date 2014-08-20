import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

Rectangle {
	property var title: "Pending Transactions"
	property var iconSource: "../tx.png"
	property var menuItem

	objectName: "pendingTxView"
	anchors.fill: parent
	visible: false
	id: pendingTxView

	property var pendingTxModel: ListModel {
		id: pendingTxModel
	}

	TableView {
		id: pendingTxTableView
		anchors.fill: parent
		TableViewColumn{ role: "value" ; title: "Value" ; width: 100 }
		TableViewColumn{ role: "from" ; title: "sender" ; width: 230 }
		TableViewColumn{ role: "to" ; title: "Reciever" ; width: 230 }
		TableViewColumn{ role: "contract" ; title: "Contract" ; width: 100 }

		model: pendingTxModel
	}

	function addTx(tx, inout) {
		var isContract
		if (tx.contract == true){
			isContract = "Yes"
		}else{
			isContract = "No"
		}


		pendingTxModel.insert(0, {hash: tx.hash, to: tx.address, from: tx.sender, value: tx.value, contract: isContract})
	}
}
