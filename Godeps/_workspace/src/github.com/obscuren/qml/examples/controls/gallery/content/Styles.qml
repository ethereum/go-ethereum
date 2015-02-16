/****************************************************************************
**
** Copyright (C) 2013 Digia Plc and/or its subsidiary(-ies).
** Contact: http://www.qt-project.org/legal
**
** This file is part of the Qt Quick Controls module of the Qt Toolkit.
**
** $QT_BEGIN_LICENSE:BSD$
** You may use this file under the terms of the BSD license as follows:
**
** "Redistribution and use in source and binary forms, with or without
** modification, are permitted provided that the following conditions are
** met:
**   * Redistributions of source code must retain the above copyright
**     notice, this list of conditions and the following disclaimer.
**   * Redistributions in binary form must reproduce the above copyright
**     notice, this list of conditions and the following disclaimer in
**     the documentation and/or other materials provided with the
**     distribution.
**   * Neither the name of Digia Plc and its Subsidiary(-ies) nor the names
**     of its contributors may be used to endorse or promote products derived
**     from this software without specific prior written permission.
**
**
** THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
** "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
** LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
** A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
** OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
** SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
** LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
** DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
** THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
** (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
** OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE."
**
** $QT_END_LICENSE$
**
****************************************************************************/





import QtQuick 2.2
import QtQuick.Controls 1.1
import QtQuick.Controls.Styles 1.1
import QtQuick.Particles 2.0
import QtQuick.Layouts 1.0

Item {
    id: root
    width: 300
    height: 200

    property int columnWidth: 120
    GridLayout {
        rowSpacing: 12
        columnSpacing: 30
        anchors.top: parent.top
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.margins: 30

        Button {
            text: "Push me"
            style: ButtonStyle { }
            implicitWidth: columnWidth
        }
        Button {
            text: "Push me"
            style: ButtonStyle {
                background: BorderImage {
                    source: control.pressed ? "../images/button-pressed.png" : "../images/button.png"
                    border.left: 4 ; border.right: 4 ; border.top: 4 ; border.bottom: 4
                }
            }
            implicitWidth: columnWidth
        }
        Button {
            text: "Push me"
            style: buttonStyle
            implicitWidth: columnWidth
        }

        TextField {
            Layout.row: 1
            style: TextFieldStyle { }
            implicitWidth: columnWidth
        }
        TextField {
            style: TextFieldStyle {
                background: BorderImage {
                    source: "../images/textfield.png"
                    border.left: 4 ; border.right: 4 ; border.top: 4 ; border.bottom: 4
                }
            }
            implicitWidth: columnWidth
        }
        TextField {
            style: textfieldStyle
            implicitWidth: columnWidth
        }

        Slider {
            id: slider1
            Layout.row: 2
            value: 0.5
            implicitWidth: columnWidth
            style: SliderStyle { }
        }
        Slider {
            id: slider2
            value: 0.5
            implicitWidth: columnWidth
            style: SliderStyle {
                groove: BorderImage {
                    height: 6
                    border.top: 1
                    border.bottom: 1
                    source: "../images/progress-background.png"
                    border.left: 6
                    border.right: 6
                    BorderImage {
                        anchors.verticalCenter: parent.verticalCenter
                        source: "../images/progress-fill.png"
                        border.left: 5 ; border.top: 1
                        border.right: 5 ; border.bottom: 1
                        width: styleData.handlePosition
                        height: parent.height
                    }
                }
                handle: Item {
                    width: 13
                    height: 13
                    Image {
                        anchors.centerIn: parent
                        source: "../images/slider-handle.png"
                    }
                }
            }
        }
        Slider {
            id: slider3
            value: 0.5
            implicitWidth: columnWidth
            style: sliderStyle
        }

        ProgressBar {
            Layout.row: 3
            value: slider1.value
            implicitWidth: columnWidth
            style: ProgressBarStyle{ }
        }
        ProgressBar {
            value: slider2.value
            implicitWidth: columnWidth
            style: progressBarStyle
        }
        ProgressBar {
            value: slider3.value
            implicitWidth: columnWidth
            style: progressBarStyle2
        }

        CheckBox {
            text: "CheckBox"
            style: CheckBoxStyle{}
            Layout.row: 4
            implicitWidth: columnWidth
        }
        RadioButton {
            style: RadioButtonStyle{}
            text: "RadioButton"
            implicitWidth: columnWidth
        }

        ComboBox {
            model: ["Paris", "Oslo", "New York"]
            style: ComboBoxStyle{}
            implicitWidth: columnWidth
        }

        TabView {
            Layout.row: 5
            Layout.columnSpan: 3
            Layout.fillWidth: true
            implicitHeight: 30
            Tab { title: "One" ; Item {}}
            Tab { title: "Two" ; Item {}}
            Tab { title: "Three" ; Item {}}
            Tab { title: "Four" ; Item {}}
            style: TabViewStyle {}
        }

        TabView {
            Layout.row: 6
            Layout.columnSpan: 3
            Layout.fillWidth: true
            implicitHeight: 30
            Tab { title: "One" ; Item {}}
            Tab { title: "Two" ; Item {}}
            Tab { title: "Three" ; Item {}}
            Tab { title: "Four" ; Item {}}
            style: tabViewStyle
        }
    }

    // Style delegates:

    property Component buttonStyle: ButtonStyle {
        background: Rectangle {
            implicitHeight: 22
            implicitWidth: columnWidth
            color: control.pressed ? "darkGray" : control.activeFocus ? "#cdd" : "#ccc"
            antialiasing: true
            border.color: "gray"
            radius: height/2
            Rectangle {
                anchors.fill: parent
                anchors.margins: 1
                color: "transparent"
                antialiasing: true
                visible: !control.pressed
                border.color: "#aaffffff"
                radius: height/2
            }
        }
    }

    property Component textfieldStyle: TextFieldStyle {
        background: Rectangle {
            implicitWidth: columnWidth
            implicitHeight: 22
            color: "#f0f0f0"
            antialiasing: true
            border.color: "gray"
            radius: height/2
            Rectangle {
                anchors.fill: parent
                anchors.margins: 1
                color: "transparent"
                antialiasing: true
                border.color: "#aaffffff"
                radius: height/2
            }
        }
    }

    property Component sliderStyle: SliderStyle {
        handle: Rectangle {
            width: 18
            height: 18
            color: control.pressed ? "darkGray" : "lightGray"
            border.color: "gray"
            antialiasing: true
            radius: height/2
            Rectangle {
                anchors.fill: parent
                anchors.margins: 1
                color: "transparent"
                antialiasing: true
                border.color: "#eee"
                radius: height/2
            }
        }

        groove: Rectangle {
            height: 8
            implicitWidth: columnWidth
            implicitHeight: 22

            antialiasing: true
            color: "#ccc"
            border.color: "#777"
            radius: height/2
            Rectangle {
                anchors.fill: parent
                anchors.margins: 1
                color: "transparent"
                antialiasing: true
                border.color: "#66ffffff"
                radius: height/2
            }
        }
    }

    property Component progressBarStyle: ProgressBarStyle {
        background: BorderImage {
            source: "../images/progress-background.png"
            border.left: 2 ; border.right: 2 ; border.top: 2 ; border.bottom: 2
        }
        progress: Item {
            clip: true
            BorderImage {
                anchors.fill: parent
                anchors.rightMargin: (control.value < control.maximumValue) ? -4 : 0
                source: "../images/progress-fill.png"
                border.left: 10 ; border.right: 10
                Rectangle {
                    width: 1
                    color: "#a70"
                    opacity: 0.8
                    anchors.top: parent.top
                    anchors.bottom: parent.bottom
                    anchors.bottomMargin: 1
                    anchors.right: parent.right
                    visible: control.value < control.maximumValue
                    anchors.rightMargin: -parent.anchors.rightMargin
                }
            }
            ParticleSystem{ id: bubbles; running: visible }
            ImageParticle{
                id: fireball
                system: bubbles
                source: "../images/bubble.png"
                opacity: 0.7
            }
            Emitter{
                system: bubbles
                anchors.bottom: parent.bottom
                anchors.margins: 4
                anchors.bottomMargin: -4
                anchors.left: parent.left
                anchors.right: parent.right
                size: 4
                sizeVariation: 4
                acceleration: PointDirection{ y: -6; xVariation: 3 }
                emitRate: 6 * control.value
                lifeSpan: 3000
            }
        }
    }

    property Component progressBarStyle2: ProgressBarStyle {
        background: Rectangle {
            implicitWidth: columnWidth
            implicitHeight: 24
            color: "#f0f0f0"
            border.color: "gray"
        }
        progress: Rectangle {
            color: "#ccc"
            border.color: "gray"
            Rectangle {
                color: "transparent"
                border.color: "#44ffffff"
                anchors.fill: parent
                anchors.margins: 1
            }
        }
    }

    property Component tabViewStyle: TabViewStyle {
        tabOverlap: 16
        frameOverlap: 4
        tabsMovable: true

        frame: Rectangle {
            gradient: Gradient{
                GradientStop { color: "#e5e5e5" ; position: 0 }
                GradientStop { color: "#e0e0e0" ; position: 1 }
            }
            border.color: "#898989"
            Rectangle { anchors.fill: parent ; anchors.margins: 1 ; border.color: "white" ; color: "transparent" }
        }
        tab: Item {
            property int totalOverlap: tabOverlap * (control.count - 1)
            implicitWidth: Math.min ((styleData.availableWidth + totalOverlap)/control.count - 4, image.sourceSize.width)
            implicitHeight: image.sourceSize.height
            BorderImage {
                id: image
                anchors.fill: parent
                source: styleData.selected ? "../images/tab_selected.png" : "../images/tab.png"
                border.left: 30
                smooth: false
                border.right: 30
            }
            Text {
                text: styleData.title
                anchors.centerIn: parent
            }
        }
        leftCorner: Item { implicitWidth: 12 }
    }
}

