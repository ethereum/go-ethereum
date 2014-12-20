import QtQuick 2.0
import QtWebKit 3.0
import QtWebKit.experimental 1.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Window 2.1;
import Ethereum 1.0

ApplicationWindow {
    id: window
    title: "muted"
    width: 900
    height: 600
    minimumHeight: 300

    property alias url: webView.url
    property alias webView: webView


    Item {
        id: root
        anchors.fill: parent
        WebView {
            objectName: "webView"
            id: webView
            anchors {
                top: root.top
                right: root.right
                left: root.left
                bottom: root.bottom
                //bottom: sizeGrip.top
            }

            experimental.preferences.javascriptEnabled: true
            experimental.preferences.navigatorQtObjectEnabled: true
            experimental.onMessageReceived: {
                var data = JSON.parse(message.data)

                switch(data.call) {
                case "log":
                    console.log.apply(this, data.args)
                    break;
                }
            }
            function postData(seed, data) {
                webview.experimental.postMessage(JSON.stringify({data: data, _seed: seed}))
            }
            function postEvent(event, data) {
                webview.experimental.postMessage(JSON.stringify({data: data, _event: event}))
            }
        }

        /*
        Rectangle {
            id: sizeGrip
            color: "gray"
            height: 5
            anchors {
                left: root.left
                right: root.right
            }
            y: Math.round(root.height * 2 / 3)

            MouseArea {
                anchors.fill: parent
                drag.target: sizeGrip
                drag.minimumY: 0
                drag.maximumY: root.height - sizeGrip.height
                drag.axis: Drag.YAxis
            }
        }
        */
    }
}
