import QtQuick 2.0
import QtWebKit 3.0
import QtWebKit.experimental 1.0
import QtQuick.Controls 1.0;
import QtQuick.Controls.Styles 1.0
import QtQuick.Layouts 1.0;
import QtQuick.Window 2.1;
import Ethereum 1.0

Rectangle {
    id: window
    property var title: "Browser"
    property var iconSource: "../browser.png"
    property var menuItem

    property alias url: webview.url
    property alias webView: webview

    property var cleanPath: false
    property var open: function(url) {
        if(!window.cleanPath) {
            var uri = url;
            if(!/.*\:\/\/.*/.test(uri)) {
                uri = "http://" + uri;
            }

            var reg = /(^https?\:\/\/(?:www\.)?)([a-zA-Z0-9_\-]*\.eth)(.*)/

            if(reg.test(uri)) {
                uri.replace(reg, function(match, pre, domain, path) {
                    uri = pre;

                    var lookup = eth.lookupDomain(domain.substring(0, domain.length - 4));
                    var ip = [];
                    for(var i = 0, l = lookup.length; i < l; i++) {
                        ip.push(lookup.charCodeAt(i))
                    }

                    if(ip.length != 0) {
                        uri += lookup;
                    } else {
                        uri += domain;
                    }

                    uri += path;
                });
            }

            window.cleanPath = true;

            webview.url = uri;

            //uriNav.text = uri.text.replace(/(^https?\:\/\/(?:www\.)?)([a-zA-Z0-9_\-]*\.\w{2,3})(.*)/, "$1$2<span style='color:#CCC'>$3</span>");
            uriNav.text = uri;
        } else {
            // Prevent inf loop.
            window.cleanPath = false;
        }
    }

    Component.onCompleted: {
        webview.url = "http://etherian.io"
    }

    signal messages(var messages, int id);
    onMessages: {
        // Bit of a cheat to get proper JSON
        var m = JSON.parse(JSON.parse(JSON.stringify(messages)))
        webview.postEvent("messages", [m, id]);
    }

    Item {
        objectName: "root"
        id: root
        anchors.fill: parent
        state: "inspectorShown"

        RowLayout {
            id: navBar
            height: 40
            anchors {
                left: parent.left
                right: parent.right
                leftMargin: 7
            }

            Button {
                id: back
                onClicked: {
                    webview.goBack()
                }
                style: ButtonStyle {
                    background: Image {
                        source: "../back.png"
                        width: 30
                        height: 30
                    }
                }
            }

            TextField {
                anchors {
                    left: back.right
                    right: toggleInspector.left
                    leftMargin: 5
                    rightMargin: 5
                }
                text: "http://etherian.io"
                id: uriNav
                y: parent.height / 2 - this.height / 2

                Keys.onReturnPressed: {
                    webview.url = this.text;
                }
            }

            Button {
                id: toggleInspector
                anchors {
                    right: parent.right
                }
                iconSource: "../bug.png"
                onClicked: {
                    if(inspector.visible == true){
                        inspector.visible = false
                    }else{
                        inspector.visible = true
                        inspector.url = webview.experimental.remoteInspectorUrl
                    }
                }
            }
        }


        WebView {
            objectName: "webView"
            id: webview
            anchors {
                left: parent.left
                right: parent.right
                bottom: parent.bottom
                top: navBar.bottom
            }

            //property var cleanPath: false
            onNavigationRequested: {
                window.open(request.url.toString());					
            }

            function sendMessage(data) {
                webview.experimental.postMessage(JSON.stringify(data))
            }


            experimental.preferences.javascriptEnabled: true
            experimental.preferences.navigatorQtObjectEnabled: true
            experimental.preferences.developerExtrasEnabled: true
            experimental.userScripts: ["../ext/qt_messaging_adapter.js", "../ext/q.js", "../ext/big.js", "../ext/string.js", "../ext/html_messaging.js"]
            experimental.onMessageReceived: {
                console.log("[onMessageReceived]: ", message.data)
                // TODO move to messaging.js
                var data = JSON.parse(message.data)

                try {
                    switch(data.call) {
                        case "compile":
                        postData(data._seed, eth.compile(data.args[0]))
                        break

                        case "getCoinBase":
                        postData(data._seed, eth.coinBase())

                        break

                        case "getIsListening":
                        postData(data._seed, eth.isListening())

                        break

                        case "getIsMining":
                        postData(data._seed, eth.isMining())

                        break

                        case "getPeerCount":
                        postData(data._seed, eth.peerCount())

                        break

                        case "getCountAt":
                        require(1)
                        postData(data._seed, eth.txCountAt(data.args[0]))

                        break

                        case "getCodeAt":
                        require(1)
                        var code = eth.codeAt(data.args[0])
                        postData(data._seed, code);

                        break

                        case "getBlockByNumber":
                        var block = eth.blockByNumber(data.args[0])
                        postData(data._seed, block)

                        break

                        case "getBlockByHash":
                        var block = eth.blockByHash(data.args[0])
                        postData(data._seed, block)

                        break

                        case "transact":
                        require(5)

                        var tx = eth.transact(data.args)
                        postData(data._seed, tx)

                        break

                        case "getStorageAt":
                        require(2);

                        var storage = eth.storageAt(data.args[0], data.args[1]);
                        postData(data._seed, storage)

                        break

                        case "call":
                        require(1);
                        var ret = eth.call(data.args)
                        postData(data._seed, ret)

                        break

                        case "getEachStorage":
                        require(1);
                        var storage = JSON.parse(eth.eachStorage(data.args[0]))
                        postData(data._seed, storage)

                        break

                        case "getTransactionsFor":
                        require(1);
                        var txs = eth.transactionsFor(data.args[0], true)
                        postData(data._seed, txs)

                        break

                        case "getBalanceAt":
                        require(1);

                        postData(data._seed, eth.balanceAt(data.args[0]));

                        break

                        case "getKey":
                        var key = eth.key().privateKey;

                        postData(data._seed, key)
                        break

                        case "watch":
                        require(2)
                        eth.watch(data.args[0], data.args[1])

                        case "disconnect":
                        require(1)
                        postData(data._seed, null)

                        break;

                        case "getSecretToAddress":
                        require(1)

                        var addr = eth.secretToAddress(data.args[0])
                        console.log("getsecret", addr)
                        postData(data._seed, addr)

                        break;

                        case "messages":
                        require(1);

                        var messages = JSON.parse(eth.getMessages(data.args[0]))
                        postData(data._seed, messages)

                        break

                        case "mutan":
                        require(1)

                        var code = eth.compileMutan(data.args[0])
                        postData(data._seed, "0x"+code)

                        break;

                        case "newFilterString":
                        require(1)
                        var id = eth.newFilterString(data.args[0])
                        postData(data._seed, id);
                        break;
                        case "newFilter":
                        require(1)
                        var id = eth.newFilter(data.args[0])

                        postData(data._seed, id);
                        break;

                        case "getMessages":
                        require(1);

                        var messages = eth.messages(data.args[0]);
                        var m = JSON.parse(JSON.parse(JSON.stringify(messages)))
                        postData(data._seed, m);

                        break;

                        case "deleteFilter":
                        require(1);
                        eth.uninstallFilter(data.args[0])
                        break;
                    }
                } catch(e) {
                    console.log(data.call + ": " + e)

                    postData(data._seed, null);
                }
            }


            function post(seed, data) {
                postData(data._seed, data)
            }

            function require(args, num) {
                if(args.length < num) {
                    throw("required argument count of "+num+" got "+args.length);
                }
            }
            function postData(seed, data) {
                webview.experimental.postMessage(JSON.stringify({data: data, _seed: seed}))
            }
            function postEvent(event, data) {
                webview.experimental.postMessage(JSON.stringify({data: data, _event: event}))
            }

            function onWatchedCb(data, id) {
                var messages = JSON.parse(data)
                postEvent("watched:"+id, messages)
            }

            function onNewBlockCb(block) {
                postEvent("block:new", block)
            }
            function onObjectChangeCb(stateObject) {
                postEvent("object:"+stateObject.address(), stateObject)
            }
            function onStorageChangeCb(storageObject) {
                var ev = ["storage", storageObject.stateAddress, storageObject.address].join(":");
                postEvent(ev, [storageObject.address, storageObject.value])
            }
        }


        Rectangle {
            id: sizeGrip
            color: "gray"
            visible: false
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
            visible: false
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
                }
            }
        ]
    }
}
