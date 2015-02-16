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
//import QtQuick.XmlListModel 2.1

Item {
    id: root
    width: 600
    height: 300
    anchors.fill: parent
    anchors.margins: Qt.platform.os === "osx" ? 12 : 6

//    XmlListModel {
//        id: flickerModel
//        source: "http://api.flickr.com/services/feeds/photos_public.gne?format=rss2&tags=" + "Cat"
//        query: "/rss/channel/item"
//        namespaceDeclarations: "declare namespace media=\"http://search.yahoo.com/mrss/\";"
//        XmlRole { name: "title"; query: "title/string()" }
//        XmlRole { name: "imagesource"; query: "media:thumbnail/@url/string()" }
//        XmlRole { name: "credit"; query: "media:credit/string()" }
//    }

    ListModel {
        id: dummyModel
        Component.onCompleted: {
            for (var i = 0 ; i < 100 ; ++i) {
                append({"index": i, "title": "A title " + i, "imagesource" :"http://someurl.com", "credit" : "N/A"})
            }
        }
    }

    TableView{
        model: dummyModel
        anchors.fill: parent

        TableViewColumn {
            role: "index"
            title: "#"
            width: 36
            resizable: false
            movable: false
        }
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
    }
}
