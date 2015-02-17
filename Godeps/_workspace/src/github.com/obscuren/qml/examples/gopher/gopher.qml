import QtQuick 2.0
import GoExtensions 1.0

Rectangle {
	width: 640; height: 400
	color: "black"

	Gopher {
		id: gopher
		width: 300; height: 300
		anchors.centerIn: parent

		NumberAnimation on rotation {
			id: anim
			from: 360; to: 0
			duration: 5000
			loops: Animation.Infinite
		}

		MouseArea {
			anchors.fill: parent

			property real startX
			property real startR

			onPressed: {
				startX = mouse.x
				startR = gopher.rotation
				anim.running = false
			}
			onReleased: {
				anim.from = gopher.rotation + 360
				anim.to = gopher.rotation
				anim.running = true
			}
			onPositionChanged: {
				gopher.rotation = (36000 + (startR - (mouse.x - startX))) % 360
			}
		}

	}
}
