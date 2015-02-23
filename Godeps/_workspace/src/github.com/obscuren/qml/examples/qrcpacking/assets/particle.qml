import QtQuick 2.0
import QtQuick.Particles 2.0
import QtGraphicalEffects 1.0;

Rectangle {
	id: root

	width: 640
	height: 480

	gradient: Gradient {
		GradientStop { position: 0.0; color: "#3a2c32"; }
		GradientStop { position: 0.8; color: "#875864"; }
		GradientStop { position: 1.0; color: "#9b616c"; }
	}

	Text {
		text: ctrl.message

		Component.onCompleted: {
			x = parent.width/2 - width/2
			y = parent.height/2 - height/2
		}

		color: "white"
		font.bold: true
		font.pointSize: 20

		MouseArea {
		    id: mouseArea
		    anchors.fill: parent
		    drag.target: parent
                    onReleased: ctrl.textReleased(parent)
		}
	}

	ParticleSystem { id: sys }

	ImageParticle {
		system: sys
		source: "qrc:///assets/particle.png"
		color: "white"
		colorVariation: 1.0
		alpha: 0.1
	}

	property var emitterComponent: Component {
		id: emitterComponent
		Emitter {
			id: container
			system: sys
			Emitter {
				system: sys
				emitRate: 128
				lifeSpan: 600
				size: 16
				endSize: 8
				velocity: AngleDirection { angleVariation:360; magnitude: 60 }
			}

			property int life: 2600
			property real targetX: 0
			property real targetY: 0
			emitRate: 128
			lifeSpan: 600
			size: 24
			endSize: 8
			NumberAnimation on x {
				objectName: "xAnim"
				id: xAnim;
				to: targetX
				duration: life
				running: false
			}
			NumberAnimation on y {
				objectName: "yAnim"
				id: yAnim;
				to: targetY
				duration: life
				running: false
			}
			Timer {
				interval: life
				running: true
				onTriggered: ctrl.done(container)
			}
		}
	}
}
