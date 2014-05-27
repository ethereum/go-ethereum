import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

ApplicationWindow {
	id: debugWindow
	visible: false
	title: "Debugger"
	minimumWidth: 600
	minimumHeight: 600
	width: 800
	height: 600

	SplitView {
		anchors.fill: parent
		property var asmModel: ListModel {
			id: asmModel
		}
		TableView {
			id: asmTableView
			width: 200
			TableViewColumn{ role: "value" ; title: "" ; width: 100 }
			model: asmModel
		}

		Rectangle {
			anchors.left: asmTableView.right
			anchors.right: parent.right
			SplitView {
				orientation: Qt.Vertical
				anchors.fill: parent

				TableView {
					property var memModel: ListModel {
						id: memModel
					}
					height: parent.height/2
					width: parent.width
					TableViewColumn{ id:mnumColmn ; role: "num" ; title: "#" ; width: 50}
					TableViewColumn{ role: "value" ; title: "Memory" ; width: 750}
					model: memModel
				}

				SplitView {
					orientation: Qt.Horizontal
					TableView {
						property var debuggerLog: ListModel {
							id: debuggerLog
						}
						TableViewColumn{ role: "value"; title: "Debug messages" }
						model: debuggerLog
					}
					TableView {
						property var stackModel: ListModel {
							id: stackModel
						}
						height: parent.height/2
						width: parent.width
						TableViewColumn{ role: "value" ; title: "Stack" ; width: 200 }
						model: stackModel
					}
				}
			}
		}
	}
	statusBar: StatusBar {
		RowLayout {
			anchors.fill: parent
			Button {
				property var enabled: true
				id: debugNextButton
				onClicked: {
					//db.next()
				}
				text: "Next"
			}
		}
	}

	function setAsm(asm) {
		asmModel.append({asm: asm})
	}

	function setInstruction(num) {
		asmTableView.selection.clear()
		asmTableView.selection.select(num-1)
	}

	function clearAsm() {
		asmModel.clear()
	}

	function setMem(mem) {
		memModel.append({num: mem.num, value: mem.value})
	}
	function clearMem(){
		memModel.clear()
	}

	function setStack(stack) {
		stackModel.append({value: stack})
	}
	function addDebugMessage(message){
		debuggerLog.append({value: message})
	}

	function clearStack() {
		stackModel.clear()
	}
}
