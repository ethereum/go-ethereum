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
	property alias debugUrl: debugView.url
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
				bottom: sizeGrip.top
			}
		}

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

		WebView {
			id: debugView
			objectName: "debugView"
			anchors {
				left: root.left
				right: root.right
				bottom: root.bottom
				top: sizeGrip.bottom
			}
		}
	}
}
