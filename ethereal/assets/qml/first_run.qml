import QtQuick 2.0
import Ethereum 1.0

// Which ones do we actually need?
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import QtQuick.Dialogs 1.1

ApplicationWindow {
  id: wizardRoot
  width: 500
  height: 400
  title: "Ethereal first run setup"

  Column {
    spacing: 5
    anchors.leftMargin: 10
    anchors.left: parent.left

    Text {
      visible: true
      text: "<h2>Ethereal setup</h2>"
    }

    Column {
      id: restoreColumn
      spacing: 5
      Text {
        visible: true
        font.pointSize: 14
        text: "Restore your Ethereum account"
        id: restoreLabel
      }

      TextField {
        id: txPrivKey
        width: 480
        placeholderText: "Private key or mnemonic words"
        focus: true
        onTextChanged: {
          if(this.text.length == 64){
            detailLabel.text = "Private (hex) key detected."
            actionButton.enabled = true
          }
          else if(this.text.split(" ").length == 24){
            detailLabel.text = "Mnemonic key detected."
            actionButton.enabled = true
          }else{
            detailLabel.text = ""
            actionButton.enabled = false
          }
        }
      }
      Row {
        spacing: 10
        Button {
          id: actionButton
          text: "Restore"
          enabled: false
          onClicked: {
           var success = eth.importAndSetPrivKey(txPrivKey.text)
           if(success){
             importedDetails.visible = true
             restoreColumn.visible = false
             newKey.visible = false
             wizardRoot.height = 120
           }
          }
        }
        Text {
          id: detailLabel
          font.pointSize: 12
          anchors.topMargin: 10
        }
      }
    }
    Column {
      id: importedDetails
      visible: false
      Text {
        text: "<b>Your account has been imported. Please close the application and restart it again to let the changes take effect.</b>"
        wrapMode: Text.WordWrap
        width: 460
      }
    }
    Column {
      spacing: 5
      id: newDetailsColumn
      visible: false
      Text {
        font.pointSize: 14
        text: "Your account details"
      }
      Label {
        text: "Address"
      }
      TextField {
        id: addressInput
        readOnly:true
        width: 480
      }
      Label {
        text: "Private key"
      }
      TextField {
        id: privkeyInput
        readOnly:true
        width: 480
      }
      Label {
        text: "Mnemonic words"
      }
      TextField {
        id: mnemonicInput
        readOnly:true
        width: 480
      }
      Label {
        text: "<b>A new account has been created. Please take the time to write down the <i>24 words</i>. You can use those to restore your account at a later date.</b>"
        wrapMode: Text.WordWrap
        width: 480
      }
      Label {
        text: "Please restart the application once you have completed the steps above."
        wrapMode: Text.WordWrap
        width: 480
      }
    }

  }
  Button {
    anchors.right: parent.right
    anchors.bottom: parent.bottom
    anchors.rightMargin: 10
    anchors.bottomMargin: 10
    id: newKey
    text: "I don't have an account yet"
    onClicked: {
      var res = eth.createAndSetPrivKey()
      mnemonicInput.text = res[0]
      addressInput.text = res[1]
      privkeyInput.text = res[2]

      // Hide restore
      restoreColumn.visible = false

      // Show new details
      newDetailsColumn.visible = true
      newKey.visible = false
    }
  }
}
