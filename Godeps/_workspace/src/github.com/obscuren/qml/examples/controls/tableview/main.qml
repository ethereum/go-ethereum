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
import QtQuick.Window 2.1
import QtQuick.Controls 1.1
import QtQuick.XmlListModel 2.0

Window {
    visible: true
    width: 538 + frame.margins * 2
    height: 360 + frame.margins * 2

    ToolBar {
        id: toolbar
        width: parent.width

        ListModel {
            id: delegatemenu
            ListElement { text: "Shiny delegate" }
            ListElement { text: "Scale selected" }
            ListElement { text: "Editable items" }
        }

        ComboBox {
            id: delegateChooser
            enabled: frame.currentIndex === 3 ? 1 : 0
            model: delegatemenu
            width: 150
            anchors.left: parent.left
            anchors.leftMargin: 8
            anchors.verticalCenter: parent.verticalCenter
        }

        CheckBox {
            id: enabledCheck
            text: "Enabled"
            checked: true
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    SystemPalette {id: syspal}
    color: syspal.window

    XmlListModel {
        id: flickerModel
        source: "http://api.flickr.com/services/feeds/photos_public.gne?format=rss2&tags=" + "Qt"
        query: "/rss/channel/item"
        namespaceDeclarations: "declare namespace media=\"http://search.yahoo.com/mrss/\";"
        XmlRole { name: "title"; query: "title/string()" }
        XmlRole { name: "imagesource"; query: "media:thumbnail/@url/string()" }
        XmlRole { name: "credit"; query: "media:credit/string()" }
    }

    ListModel {
        id: nestedModel
        ListElement{content: ListElement { description: "Core" ; color:"#ffaacc"}}
        ListElement{content: ListElement { description: "Second" ; color:"#ffccaa"}}
        ListElement{content: ListElement { description: "Third" ; color:"#ffffaa"}}
    }

    ListModel {
        id: largeModel
        Component.onCompleted: {
            for (var i=0 ; i< 500 ; ++i)
                largeModel.append({"name":"Person "+i , "age": Math.round(Math.random()*100), "gender": Math.random()>0.5 ? "Male" : "Female"})
        }
    }

    Column {
        anchors.top: toolbar.bottom
        anchors.right: parent.right
        anchors.left: parent.left
        anchors.bottom:  parent.bottom
        anchors.margins: 8

        TabView {
            id:frame
            focus:true
            enabled: enabledCheck.checked

            property int margins: Qt.platform.os === "osx" ? 16 : 0

            height: parent.height - 34
            anchors.right: parent.right
            anchors.left: parent.left
            anchors.margins: margins

            Tab {
                title: "XmlListModel"

                TableView {
                    model: flickerModel
                    anchors.fill: parent
                    anchors.margins: 12

                    TableViewColumn {
                        role: "title"
                        title: "Title"
                        width: 120
                    }
                    TableViewColumn {
                        role: "credit"
                        title: "Credit"
                        width: 120
                    }
                    TableViewColumn {
                        role: "imagesource"
                        title: "Image source"
                        width: 200
                        visible: true
                    }

                    frameVisible: frameCheckbox.checked
                    headerVisible: headerCheckbox.checked
                    sortIndicatorVisible: sortableCheckbox.checked
                    alternatingRowColors: alternateCheckbox.checked
                }
            }
            Tab {
                title: "Multivalue"

                TableView {
                    model: nestedModel
                    anchors.fill: parent
                    anchors.margins: 12

                    TableViewColumn {
                        role: "content"
                        title: "Text and Color"
                        width: 220
                    }

                    itemDelegate: Item {
                        Rectangle{
                            color: styleData.value.get(0).color
                            anchors.top:parent.top
                            anchors.right:parent.right
                            anchors.bottom:parent.bottom
                            anchors.margins: 4
                            width:32
                            border.color:"#666"
                        }
                        Text {
                            width: parent.width
                            anchors.margins: 4
                            anchors.left: parent.left
                            anchors.verticalCenter: parent.verticalCenter
                            elide: styleData.elideMode
                            text: styleData.value.get(0).description
                            color: styleData.textColor
                        }
                    }

                    frameVisible: frameCheckbox.checked
                    headerVisible: headerCheckbox.checked
                    sortIndicatorVisible: sortableCheckbox.checked
                    alternatingRowColors: alternateCheckbox.checked
                }
            }
            Tab {
                title: "Generated"

                TableView {
                    model: largeModel
                    anchors.margins: 12
                    anchors.fill: parent
                    TableViewColumn {
                        role: "name"
                        title: "Name"
                        width: 120
                    }
                    TableViewColumn {
                        role: "age"
                        title: "Age"
                        width: 120
                    }
                    TableViewColumn {
                        role: "gender"
                        title: "Gender"
                        width: 120
                    }
                    frameVisible: frameCheckbox.checked
                    headerVisible: headerCheckbox.checked
                    sortIndicatorVisible: sortableCheckbox.checked
                    alternatingRowColors: alternateCheckbox.checked
                }
            }

            Tab {
                title: "Delegates"
                Item {
                    anchors.fill: parent

                    Component {
                        id: delegate1
                        Item {
                            clip: true
                            Text {
                                width: parent.width
                                anchors.margins: 4
                                anchors.left: parent.left
                                anchors.verticalCenter: parent.verticalCenter
                                elide: styleData.elideMode
                                text: styleData.value !== undefined  ? styleData.value : ""
                                color: styleData.textColor
                            }
                        }
                    }

                    Component {
                        id: delegate2
                        Text {
                            width: parent.width
                            anchors.margins: 4
                            anchors.left: parent.left
                            anchors.verticalCenter: parent.verticalCenter
                            elide: styleData.elideMode
                            text: styleData.value !== undefined  ? styleData.value : ""
                            color: styleData.textColor
                        }
                    }

                    Component {
                        id: editableDelegate
                        Item {

                            Text {
                                width: parent.width
                                anchors.margins: 4
                                anchors.left: parent.left
                                anchors.verticalCenter: parent.verticalCenter
                                elide: styleData.elideMode
                                text: styleData.value !== undefined ? styleData.value : ""
                                color: styleData.textColor
                                visible: !styleData.selected
                            }
                            Loader { // Initialize text editor lazily to improve performance
                                id: loaderEditor
                                anchors.fill: parent
                                anchors.margins: 4
                                Connections {
                                    target: loaderEditor.item
                                    onAccepted: {
                                        if (typeof styleData.value === 'number')
                                            largeModel.setProperty(styleData.row, styleData.role, Number(parseFloat(loaderEditor.item.text).toFixed(0)))
                                        else
                                            largeModel.setProperty(styleData.row, styleData.role, loaderEditor.item.text)
                                    }
                                }
                                sourceComponent: styleData.selected ? editor : null
                                Component {
                                    id: editor
                                    TextInput {
                                        id: textinput
                                        color: styleData.textColor
                                        text: styleData.value
                                        MouseArea {
                                            id: mouseArea
                                            anchors.fill: parent
                                            hoverEnabled: true
                                            onClicked: textinput.forceActiveFocus()
                                        }
                                    }
                                }
                            }
                        }
                    }
                    TableView {
                        model: largeModel
                        anchors.margins: 12
                        anchors.fill:parent
                        frameVisible: frameCheckbox.checked
                        headerVisible: headerCheckbox.checked
                        sortIndicatorVisible: sortableCheckbox.checked
                        alternatingRowColors: alternateCheckbox.checked

                        TableViewColumn {
                            role: "name"
                            title: "Name"
                            width: 120
                        }
                        TableViewColumn {
                            role: "age"
                            title: "Age"
                            width: 120
                        }
                        TableViewColumn {
                            role: "gender"
                            title: "Gender"
                            width: 120
                        }

                        headerDelegate: BorderImage{
                            source: "images/header.png"
                            border{left:2;right:2;top:2;bottom:2}
                            Text {
                                text: styleData.value
                                anchors.centerIn:parent
                                color:"#333"
                            }
                        }

                        rowDelegate: Rectangle {
                            height: (delegateChooser.currentIndex == 1 && styleData.selected) ? 30 : 20
                            Behavior on height{ NumberAnimation{} }

                            color: styleData.selected ? "#448" : (styleData.alternate? "#eee" : "#fff")
                            BorderImage{
                                id: selected
                                anchors.fill: parent
                                source: "images/selectedrow.png"
                                visible: styleData.selected
                                border{left:2; right:2; top:2; bottom:2}
                                SequentialAnimation {
                                    running: true; loops: Animation.Infinite
                                    NumberAnimation { target:selected; property: "opacity"; to: 1.0; duration: 900}
                                    NumberAnimation { target:selected; property: "opacity"; to: 0.5; duration: 900}
                                }
                            }
                        }

                        itemDelegate: {
                            if (delegateChooser.currentIndex == 2)
                                return editableDelegate;
                            else
                                return delegate1;
                        }
                    }
                }
            }
        }
        Row{
            x: 12
            height: 34
            CheckBox{
                id: alternateCheckbox
                checked: true
                text: "Alternate"
                anchors.verticalCenter: parent.verticalCenter
            }
            CheckBox{
                id: sortableCheckbox
                checked: false
                text: "Sort indicator"
                anchors.verticalCenter: parent.verticalCenter
            }
            CheckBox{
                id: frameCheckbox
                checked: true
                text: "Frame"
                anchors.verticalCenter: parent.verticalCenter
            }
            CheckBox{
                id: headerCheckbox
                checked: true
                text: "Headers"
                anchors.verticalCenter: parent.verticalCenter
            }
        }
    }
}
