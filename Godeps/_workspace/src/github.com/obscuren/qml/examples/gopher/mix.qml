import QtQuick 2.0
import GoExtensions 1.0

Rectangle {
	id: root

	width: 640
	height: 220
	color: "black"

	Rectangle {
		x: 20; y: 20; width: 100; height: 100
		color: "red"

		SequentialAnimation on x {
			loops: Animation.Infinite
			NumberAnimation { from: 20; to: 120; duration: 4000; easing.type: Easing.InOutQuad }
			NumberAnimation { from: 120; to: 20; duration: 4000; easing.type: Easing.InOutQuad }
		}
	}

	Rectangle {
		x: 40; y: 40; width: 100; height: 100
		color: "yellow"
		opacity: 0.7

		SequentialAnimation on x {
			loops: Animation.Infinite
			NumberAnimation { from: 40; to: 220; duration: 4000; easing.type: Easing.InOutQuad }
			NumberAnimation { from: 220; to: 40; duration: 4000; easing.type: Easing.InOutQuad }
		}
	}

	Gopher {
		id: gopher

		x: 60; y: 60; width: 100; height: 100

		SequentialAnimation on x {
			loops: Animation.Infinite
			NumberAnimation { from: 60; to: 320; duration: 4000; easing.type: Easing.InOutQuad }
			NumberAnimation { from: 320; to: 60; duration: 4000; easing.type: Easing.InOutQuad }
		}
	}

	Rectangle {
		x: 80; y: 80; width: 100; height: 100
		color: "yellow"
		opacity: 0.7

		SequentialAnimation on x {
			loops: Animation.Infinite
			NumberAnimation { from: 80; to: 420; duration: 4000; easing.type: Easing.InOutQuad }
			NumberAnimation { from: 420; to: 80; duration: 4000; easing.type: Easing.InOutQuad }
		}
	}

	Rectangle {
		x: 100; y: 100; width: 100; height: 100
		color: "red"

		SequentialAnimation on x {
			loops: Animation.Infinite
			NumberAnimation { from: 100; to: 520; duration: 4000; easing.type: Easing.InOutQuad }
			NumberAnimation { from: 520; to: 100; duration: 4000; easing.type: Easing.InOutQuad }
		}
	}
}
