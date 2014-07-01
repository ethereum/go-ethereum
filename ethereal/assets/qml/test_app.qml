import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import Ethereum 1.0

QmlApp {
	minimumWidth: 350
	maximumWidth: 350
	maximumHeight: 80
	minimumHeight: 80

	title: "Generic Coin"

	property string contractAddr: "f299f6c74515620e4c4cd8fe3d205b5c4f2e25c8"
	property string addr: "2ef47100e0787b915105fd5e3f4ff6752079d5cb"

	Component.onCompleted: {
		eth.watch(contractAddr, addr)
		eth.watch(addr, contractAddr)
		setAmount()
	}

	function onStorageChangeCb(storageObject) {
		setAmount()
	}

	function setAmount(){
		var state = eth.getStateObject(contractAddr)
		var storage = state.getStorage(addr)
		amountLabel.text = storage
	}
	Column {
		spacing: 5
		Row {
			spacing: 20
			Label {
				id: genLabel
				text: "Generic coin balance:"
			}
			Label {
				id: amountLabel
			}
		}
		Row {
			spacing: 20
			TextField {
				id: address
				placeholderText: "Address"
			}
			TextField {
				id: amount
				placeholderText: "Amount"
			}
		}
		Button {
			text: "Send coins"
			onClicked: {
				var privKey = eth.getKey().privateKey
				if(privKey){
					var result = eth.transact(privKey, contractAddr, 0,"100000","250", "0x" + address.text + "\n" + amount.text)
					resultTx.text = result.hash
				}
			}
		}
		Label {
			id: resultTx
		}
	}

}
