import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import Ethereum 1.0

ApplicationWindow {
	minimumWidth: 500
	maximumWidth: 500
	maximumHeight: 400
	minimumHeight: 400

	function onNewBlockCb(block) {
		console.log("Please overwrite onNewBlock(block):", block)
	}
	function onObjectChangeCb(stateObject) {
		console.log("Please overwrite onObjectChangeCb(object)", stateObject)
	}
	function onStorageChangeCb(storageObject) {
		var ev = ["storage", storageObject.stateAddress, storageObject.address].join(":");
		console.log("Please overwrite onStorageChangeCb(object)", ev)
	}
}
