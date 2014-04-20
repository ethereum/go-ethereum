import QtQuick 2.0
import QtWebKit 3.0
import QtWebKit.experimental 1.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Window 2.1;
import Ethereum 1.0

ApplicationWindow {
  id: window
  title: "Webapp"
  width: 900
  height: 600
  minimumHeight: 300
  property alias url: webview.url

  Item {
    id: root
    anchors.fill: parent
    state: "inspectorShown"

    WebView {
      id: webview
      anchors {
        left: parent.left
        right: parent.right
        bottom: sizeGrip.top
        top: parent.top
      }
      onTitleChanged: { window.title = title }
      experimental.preferences.javascriptEnabled: true
      experimental.preferences.navigatorQtObjectEnabled: true
      experimental.preferences.developerExtrasEnabled: true
      experimental.userScripts: [ui.assetPath("ethereum.js")]
      experimental.onMessageReceived: {
        console.log("[onMessageReceived]: ", message.data)
        var data = JSON.parse(message.data)

        webview.experimental.postMessage(JSON.stringify({data: {message: data.message}, _seed: data._seed}))
      }
    }

    Rectangle {
      id: sizeGrip
      color: "gray"
      visible: true
      height: 10
      anchors {
        left: root.left
        right: root.right
      }
      y: Math.round(root.height * 2 / 3)

      MouseArea {
        anchors.fill: parent
        drag.target: sizeGrip
        drag.minimumY: 0
        drag.maximumY: root.height
        drag.axis: Drag.YAxis
      }
    }

    WebView {
      id: inspector
      visible: true
      url: webview.experimental.remoteInspectorUrl
      anchors {
        left: root.left
        right: root.right
        top: sizeGrip.bottom
        bottom: root.bottom
      }
    }

    states: [
      State {
        name: "inspectorShown"
        PropertyChanges {
          target: inspector
          url: webview.experimental.remoteInspectorUrl
        }
      }
    ]
  }
}
