import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1
import Ethereum 1.0

import "../ext/filter.js" as Eth
import "../ext/http.js" as Http


ApplicationWindow {
    id: root
    
    //flags: Qt.FramelessWindowHint
    // Use this to make the window frameless. But then you'll need to do move and resize by hand

    property var ethx : Eth.ethx
    property var catalog;

    width: 1200
    height: 820
    minimumHeight: 600
    minimumWidth: 800

    title: "Mist"

    TextField {
        id: copyElementHax
        visible: false
    }

    function copyToClipboard(text) {
        copyElementHax.text = text
        copyElementHax.selectAll()
        copyElementHax.copy()
    }

    // Takes care of loading all default plugins
    Component.onCompleted: {

        catalog = addPlugin("./views/catalog.qml", {noAdd: true, close: false, section: "begin", active: true});

        var walletWeb = addPlugin("./views/browser.qml", {noAdd: true, close: false, section: "ethereum", active: false});
        walletWeb.view.url = "http://ethereum-dapp-wallet.meteor.com/";
        walletWeb.menuItem.title = "Wallet";

        addPlugin("./views/wallet.qml", {noAdd: true, close: false, section: "legacy"});        
        addPlugin("./views/miner.qml", {noAdd: true, close: false, section: "ethereum", active: false});
        addPlugin("./views/transaction.qml", {noAdd: true, close: false, section: "legacy"});
        addPlugin("./views/whisper.qml", {noAdd: true, close: false, section: "legacy"});
        addPlugin("./views/chain.qml", {noAdd: true, close: false, section: "legacy"});
        addPlugin("./views/pending_tx.qml", {noAdd: true, close: false, section: "legacy"});
        addPlugin("./views/info.qml", {noAdd: true, close: false, section: "legacy"});

        mainSplit.setView(catalog.view, catalog.menuItem);

        //newBrowserTab("http://ethereum-dapp-catalog.meteor.com");

        // Command setup
        gui.sendCommand(0)
    }

    function activeView(view, menuItem) {
        mainSplit.setView(view, menuItem)
        /*if (view.hideUrl) {
            urlPane.visible = false;
            mainView.anchors.top = rootView.top
        } else {
            urlPane.visible = true;
            mainView.anchors.top = divider.bottom
        }*/

    }

    function addViews(view, path, options) {
        var views = mainSplit.addComponent(view, options)
        views.menuItem.path = path

        mainSplit.views.push(views);

        if(!options.noAdd) {
            gui.addPlugin(path)
        }

        return views
    }

    function addPlugin(path, options) {
        try {
            if(typeof(path) === "string" && /^https?/.test(path)) {
                console.log('load http')
                Http.request(path, function(o) {
                    if(o.status === 200) {
                        var view = Qt.createQmlObject(o.responseText, mainView, path)
                        addViews(view, path, options)
                    }
                })

                return
            }

            var component = Qt.createComponent(path);
            if(component.status != Component.Ready) {
                if(component.status == Component.Error) {
                    ethx.note("error: ", component.errorString());
                }

                return
            }

            var view = mainView.createView(component, options)
            var views = addViews(view, path, options)

            return views
        } catch(e) {
            console.log(e)
        }
    }

    function newBrowserTab(url) {
        
        var urlMatches = url.toString().match(/^[a-z]*\:\/\/([^\/?#]+)(?:[\/?#]|$)/i);
        var requestedDomain = urlMatches && urlMatches[1];

        var domainAlreadyOpen = false;

        for(var i = 0; i < mainSplit.views.length; i++) {
            if (mainSplit.views[i].view.url) {
                var matches = mainSplit.views[i].view.url.toString().match(/^[a-z]*\:\/\/(?:www\.)?([^\/?#]+)(?:[\/?#]|$)/i);
                var existingDomain = matches && matches[1];
                if (requestedDomain == existingDomain) {
                    domainAlreadyOpen = true;
                    mainSplit.views[i].view.url = url;
                    activeView(mainSplit.views[i].view, mainSplit.views[i].menuItem);
                }
            }
        }  

        if (!domainAlreadyOpen) {            
            var window = addPlugin("./views/browser.qml", {noAdd: true, close: true, section: "apps", active: true});
            window.view.url = url;
            window.menuItem.title = "Mist";
            activeView(window.view, window.menuItem);
        }
    }



    menuBar: MenuBar {
        Menu {
            title: "File"
            MenuItem {
                text: "New tab"
                shortcut: "Ctrl+t"
                onTriggered: {
	            activeView(catalog.view, catalog.menuItem);
                }
            }

            MenuSeparator {}

            MenuItem {
                text: "Import key"
                shortcut: "Ctrl+i"
                onTriggered: {
                    generalFileDialog.show(true, function(path) {
                        gui.importKey(path)
                    })
                }
            }

            MenuItem {
                text: "Export keys"
                shortcut: "Ctrl+e"
                onTriggered: {
                    generalFileDialog.show(false, function(path) {
                    })
                }
            }

        }

        Menu {
            title: "Developer"
            MenuItem {
                iconSource: "../icecream.png"
                text: "Debugger"
                shortcut: "Ctrl+d"
                onTriggered: eth.startDebugger()
            }

            MenuItem {
                text: "Import Tx"
                onTriggered: {
                    txImportDialog.visible = true
                }
            }

            MenuItem {
                text: "Run JS file"
                onTriggered: {
                    generalFileDialog.show(true, function(path) {
                        eth.evalJavascriptFile(path)
                    })
                }
            }

            MenuItem {
                text: "Dump state"
                onTriggered: {
                    generalFileDialog.show(false, function(path) {
                        // Empty hash for latest
                        gui.dumpState("", path)
                    })
                }
            }

            MenuSeparator {}
        }

        Menu {
            title: "Network"
            MenuItem {
                text: "Add Peer"
                shortcut: "Ctrl+p"
                onTriggered: {
                    addPeerWin.visible = true
                }
            }
            MenuItem {
                text: "Show Peers"
                shortcut: "Ctrl+e"
                onTriggered: {
                    peerWindow.visible = true
                }
            }
        }

        Menu {
            title: "Help"
            MenuItem {
                text: "About"
                onTriggered: {
                    aboutWin.visible = true
                }
            }
        }
    }

    property var blockModel: ListModel {
        id: blockModel
    }

    SplitView {
        property var views: [];

        id: mainSplit
        anchors.fill: parent
        //resizing: false  // this is NOT where we remove that damning resizing handle..
        handleDelegate: Item {
            //This handle is a way to remove the line between the split views
            Rectangle {
                anchors.fill: parent
            }
         }

        function setView(view, menu) {
            for(var i = 0; i < views.length; i++) {
                views[i].view.visible = false
                views[i].menuItem.setSelection(false)
            }
            view.visible = true
            menu.setSelection(true)
        }

        function addComponent(view, options) {
            view.visible = false
            view.anchors.fill = mainView

            var menuItem = menu.createMenuItem(view, options);
            if( view.hasOwnProperty("menuItem") ) {
                view.menuItem = menuItem;
            }

            if( view.hasOwnProperty("onReady") ) {
                view.onReady.call(view)
            }

            if( options.active ) {
                setView(view, menuItem)
            }


            return {view: view, menuItem: menuItem}
        }

        /*********************
         * Main menu.
         ********************/
         Rectangle {
             id: menu
             Layout.minimumWidth: 192
             Layout.maximumWidth: 192

            FontLoader { 
               id: sourceSansPro
               source: "fonts/SourceSansPro-Regular.ttf" 
            }
            FontLoader { 
               source: "fonts/SourceSansPro-Semibold.ttf" 
            }            
            FontLoader { 
               source: "fonts/SourceSansPro-Bold.ttf" 
            } 
            FontLoader { 
               source: "fonts/SourceSansPro-Black.ttf" 
            }            
            FontLoader { 
               source: "fonts/SourceSansPro-Light.ttf" 
            }              
            FontLoader { 
               source: "fonts/SourceSansPro-ExtraLight.ttf" 
            }  
            FontLoader { 
               id: simpleLineIcons
               source: "fonts/Simple-Line-Icons.ttf" 
            }

            Rectangle {
                color: "steelblue"
                anchors.fill: parent

                MouseArea {
                    anchors.fill: parent
                    property real lastMouseX: 0
                    property real lastMouseY: 0
                    onPressed: {
                        lastMouseX = mouseX
                        lastMouseY = mouseY
                    }
                    onPositionChanged: {
                        root.x += (mouseX - lastMouseX)
                        root.y += (mouseY - lastMouseY)
                    }
                    /*onDoubleClicked: {
                        //!maximized ? view.set_max() : view.set_normal()}
                        visibility = "Minimized"
                    }*/
                }
            }



             anchors.top: parent.top
             Rectangle {
                     width: parent.height
                     height: parent.width
                     anchors.centerIn: parent
                     rotation: 90

                     gradient: Gradient {
                         GradientStop { position: 0.0; color: "#E2DEDE" }
                         GradientStop { position: 0.1; color: "#EBE8E8" }
                         GradientStop { position: 1.0; color: "#EBE8E8" }
                     }
             }

             Component {
                 id: menuItemTemplate
                 Rectangle {
                     id: menuItem
                     property var view;
                     property var path;
                     property var closable;
                     property var badgeContent;

                     property alias title: label.text
                     property alias icon: icon.source
                     property alias secondaryTitle: secondary.text
                     property alias badgeNumber: badgeNumberLabel.text
                     property alias badgeIcon: badgeIconLabel.text

                     function setSelection(on) {
                         sel.visible = on
                         
                         if (this.closable == true) {
                                closeIcon.visible = on
                         }
                     }

                     function setAsBigButton(on) {
                        newAppButton.visible = on
                        label.visible = !on
                        buttonLabel.visible = on
                     }
 
                     width: 192
                     height: 55
                     color: "#00000000"

                     anchors {
                         left: parent.left
                         leftMargin: 4
                     }

                     Rectangle {
                         // New App Button
                         id: newAppButton
                         visible: false 
                         anchors.fill: parent
                         anchors.rightMargin: 8
                         border.width: 0
                         radius: 5
                         height: 55
                         width: 180
                         color: "#F3F1F3"
                     }

                     Rectangle {
                         id: sel
                         visible: false
                         anchors.fill: parent
                         color: "#00000000"
                         Rectangle {
                             id: r
                             anchors.fill: parent
                             border.width: 0
                             radius: 5
                             color: "#FAFAFA"
                         }
                         Rectangle {
                             anchors {
                                 top: r.top
                                 bottom: r.bottom
                                 right: r.right
                             }
                             width: 10
                             color: "#FAFAFA"
                             border.width:0

                             Rectangle {
                                // Small line on top of selection. What's this for?
                                 anchors {
                                     left: parent.left
                                     right: parent.right
                                     top: parent.top
                                 }
                                 height: 1
                                 color: "#FAFAFA"
                             }

                             Rectangle {
                                // Small line on bottom of selection. What's this for again?
                                 anchors {
                                     left: parent.left
                                     right: parent.right
                                     bottom: parent.bottom
                                 }
                                 height: 1
                                 color: "#FAFAFA"
                             }
                         }
                     }

                     MouseArea {
                         anchors.fill: parent
                         hoverEnabled: true
                         onClicked: {
                             activeView(view, menuItem);
                         }
                         onEntered: {
                            if (parent.closable == true) {
                                closeIcon.visible = sel.visible
                            }
                         }
                         onExited:  {
                            closeIcon.visible = false
                         }
                     }

                     Image {
                         id: icon
                         height: 28
                         width: 28
                         anchors {
                             left: parent.left
                             verticalCenter: parent.verticalCenter
                             leftMargin: 6
                         }
                     }

                     Text {
                        id: buttonLabel
                        visible: false
                        text: "GO TO NEW APP"
                        font.family: sourceSansPro.name 
                        font.weight: Font.DemiBold
                        anchors.horizontalCenter: parent.horizontalCenter
                        anchors.verticalCenter: parent.verticalCenter
                        color: "#AAA0A0"
                     }   

                    Text {
                         id: label
                         font.family: sourceSansPro.name 
                         font.weight: Font.DemiBold
                         elide: Text.ElideRight
                         x:250
                         color: "#665F5F"
                         font.pixelSize: 14
                         anchors {
                             left: icon.right
                             right: parent.right
                             verticalCenter: parent.verticalCenter
                             leftMargin: 6
                             rightMargin: 8
                             verticalCenterOffset: (secondaryTitle == "") ? 0 : -10;
                         }


                         
                         
                     }

                     Text {
                         id: secondary
                         font.family: sourceSansPro.name 
                         font.weight: Font.Light
                         anchors {
                             left: icon.right
                             leftMargin: 6
                             top: label.bottom
                         }
                         color: "#6691C2"
                         font.pixelSize: 12
                     }

                     Rectangle {
                        id: closeIcon
                        visible: false
                        width: 10
                        height: 10
                        color: "#FAFAFA"
                        anchors {
                            fill: icon
                        }

                        MouseArea {
                             anchors.fill: parent
                             onClicked: {
                                 menuItem.closeApp()
                             }
                         }

                        Text {
                             
                             font.family: simpleLineIcons.name 
                             anchors {
                                 centerIn: parent
                             }
                             color: "#665F5F"
                             font.pixelSize: 20
                             text: "\ue082"
                         }
                     }                     

                     Rectangle {
                        id: badge
                        visible: (badgeContent == "icon" || badgeContent == "number" )? true : false 
                        width: 32
                        color: "#05000000"
                        anchors {
                            right: parent.right;
                            top: parent.top;
                            bottom: parent.bottom;
                            rightMargin: 4;
                        }
                                      
                        Text {
                             id: badgeIconLabel
                             visible: (badgeContent == "icon") ? true : false;
                             font.family: simpleLineIcons.name 
                             anchors {
                                 centerIn: parent
                             }
                             horizontalAlignment: Text.AlignCenter
                             color: "#AAA0A0"
                             font.pixelSize: 20
                             text: badgeIcon
                         }                       

                        Text {
                             id: badgeNumberLabel
                             visible: (badgeContent == "number") ? true : false;
                             anchors {
                                 centerIn: parent
                             }
                             horizontalAlignment: Text.AlignCenter
                             font.family: sourceSansPro.name 
                             font.weight: Font.Light
                             color: "#AAA0A0"
                             font.pixelSize: 18
                             text: badgeNumber
                         }
                     }
                     


                     function closeApp() {
                         if(!this.closable) { return; }

                         if(this.view.hasOwnProperty("onDestroy")) {
                             this.view.onDestroy.call(this.view)
                         }

                         this.view.destroy()
                         this.destroy()
                         for (var i = 0; i < mainSplit.views.length; i++) {
                             var view = mainSplit.views[i];
                             if (view.menuItem === this) {
                                 mainSplit.views.splice(i, 1);
                                 break;
                             }
                         }
                         gui.removePlugin(this.path)
                         activeView(mainSplit.views[0].view, mainSplit.views[0].menuItem);
                     }
                 }
             }

             function createMenuItem(view, options) {
                 if(options === undefined) {
                     options = {};
                 }

                 var section;
                 switch(options.section) {
                     case "begin":
                     section = menuBegin
                     break;
                     case "ethereum":
                     section = menuDefault;
                     break;
                     case "legacy":
                     section = menuLegacy;
                     break;
                     default:
                     section = menuApps;
                     break;
                 }

                 var comp = menuItemTemplate.createObject(section)
                 comp.view = view
                 comp.title = view.title

                 if(view.hasOwnProperty("iconSource")) {
                     comp.icon = view.iconSource;
                 }
                 comp.closable = options.close;

                 if (options.section === "begin") {
                    comp.setAsBigButton(true)
                 }

                 return comp
             }

             ColumnLayout {
                 id: menuColumn
                 y: 10
                 width: parent.width
                 anchors.left: parent.left
                 anchors.right: parent.right
                 spacing: 3
                


                ColumnLayout {
                     id: menuBegin
                     spacing: 3
                     anchors {
                         left: parent.left
                         right: parent.right
                     }
                 }

                 Rectangle {
                     height: 55
                     color: "transparent"
                     Text {
                         text: "ETHEREUM"
                         font.family: sourceSansPro.name 
                         font.weight: Font.DemiBold
                         anchors {
                             left: parent.left
                             top: parent.verticalCenter
                             leftMargin: 16
                         }
                         color: "#AAA0A0"
                     }
                 }


                 ColumnLayout {
                     id: menuDefault
                     spacing: 3
                     anchors {
                         left: parent.left
                         right: parent.right
                     }
                 }

                 Rectangle {
                     height: 55
                     color: "transparent"
                     Text {
                         text: "APPS"
                         font.family: sourceSansPro.name 
                         font.weight: Font.DemiBold
                         anchors {
                             left: parent.left
                             top: parent.verticalCenter
                             leftMargin: 16
                         }
                         color: "#AAA0A0"
                     }
                 }

                 ColumnLayout {
                     id: menuApps
                     spacing: 3
                     anchors {
                         left: parent.left
                         right: parent.right
                     }
                 }

                 Rectangle {
                     height: 55
                     color: "transparent"
                     Text {
                         text: "DEBUG"
                         font.family: sourceSansPro.name 
                         font.weight: Font.DemiBold
                         anchors {
                             left: parent.left
                             top: parent.verticalCenter
                             leftMargin: 16
                         }
                         color: "#AAA0A0"
                     }
                 }


                 ColumnLayout {
                     id: menuLegacy
                     spacing: 3
                     anchors {
                         left: parent.left
                         right: parent.right
                     }
                 }
             }
         }

         /*********************
          * Main view
          ********************/
          Rectangle {
              id: rootView
              anchors.right: parent.right
              anchors.left: menu.right
              anchors.bottom: parent.bottom
              anchors.top: parent.top
              color: "#00000000"             

              /*Rectangle {
                  id: urlPane
                  height: 40
                  color: "#00000000"
                  anchors {
                      left: parent.left
                      right: parent.right
                      leftMargin: 5
                      rightMargin: 5
                      top: parent.top
                      topMargin: 5
                  }
                  TextField {
                      id: url
                      objectName: "url"
                      placeholderText: "DApp URL"
                      anchors {
                          left: parent.left
                          right: parent.right
                          top: parent.top
                          topMargin: 5
                          rightMargin: 5
                          leftMargin: 5
                      }

                      Keys.onReturnPressed: {
                          if(/^https?/.test(this.text)) {
                              newBrowserTab(this.text);
                          } else {
                              addPlugin(this.text, {close: true, section: "apps"})
                          }
                      }
                  }

              }

              // Border
              Rectangle {
                  id: divider
                  anchors {
                      left: parent.left
                      right: parent.right
                      top: urlPane.bottom
                  }
                  z: -1
                  height: 1
                  color: "#CCCCCC"
              }*/

              Rectangle {
                  id: mainView
                  color: "#00000000"
                  anchors.right: parent.right
                  anchors.left: parent.left
                  anchors.bottom: parent.bottom
                  anchors.top: parent.top

                  function createView(component) {
                      var view = component.createObject(mainView)

                      return view;
                  }
              }
          }
      }


      /******************
       * Dialogs
       *****************/
       FileDialog {
           id: generalFileDialog
           property var callback;
           onAccepted: {
               var path = this.fileUrl.toString();
               callback.call(this, path);
           }

           function show(selectExisting, callback) {
               generalFileDialog.callback = callback;
               generalFileDialog.selectExisting = selectExisting;

               this.open();
           }
       }


       /******************
        * Wallet functions
        *****************/
        function importApp(path) {
            var ext = path.split('.').pop()
            if(ext == "html" || ext == "htm") {
                eth.openHtml(path)
            }else if(ext == "qml"){
                addPlugin(path, {close: true, section: "apps"})
            }
        }

        function setWalletValue(value) {
            //walletValueLabel.text = value
        }

        function loadPlugin(name) {
            console.log("Loading plugin" + name)
            var view = mainView.addPlugin(name)
        }

        function clearPeers() { peerModel.clear() }
        function addPeer(peer) { peerModel.append(peer) }

        function setPeerCounters(text) {
            //peerCounterLabel.text = text
        }

        function timeAgo(unixTs){
            var lapsed = (Date.now() - new Date(unixTs*1000)) / 1000
            return  (lapsed + " seconds ago")
        }

        function convertToPretty(unixTs){
            var a = new Date(unixTs*1000);
            var months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
            var year = a.getFullYear();
            var month = months[a.getMonth()];
            var date = a.getDate();
            var hour = a.getHours();
            var min = a.getMinutes();
            var sec = a.getSeconds();
            var time = date+' '+month+' '+year+' '+hour+':'+min+':'+sec ;
            return time;
        }

        /**********************
         * Windows
         *********************/
         Window {
             id: peerWindow
             //flags: Qt.CustomizeWindowHint | Qt.Tool | Qt.WindowCloseButtonHint
             height: 200
             width: 700
             Rectangle {
                 anchors.fill: parent
                 property var peerModel: ListModel {
                     id: peerModel
                 }
                 TableView {
                     anchors.fill: parent
                     id: peerTable
                     model: peerModel
                     TableViewColumn{width: 180; role: "addr" ; title: "Remote Address" }
                     TableViewColumn{width: 280; role: "nodeID" ; title: "Node ID" }
                     TableViewColumn{width: 180; role: "caps" ; title: "Capabilities" }
                 }
             }
         }

         Window {
             id: aboutWin
             visible: false
             title: "About"
             minimumWidth: 350
             maximumWidth: 350
             maximumHeight: 280
             minimumHeight: 280

             Image {
                 id: aboutIcon
                 height: 150
                 width: 150
                 fillMode: Image.PreserveAspectFit
                 smooth: true
                 source: "../facet.png"
                 x: 10
                 y: 30
             }

             Text {
                 anchors.left: aboutIcon.right
                 anchors.leftMargin: 10
                 anchors.top: parent.top
                 anchors.topMargin: 30
                 font.pointSize: 12
                 text: "<h2>Mist (0.7.10)</h2><br><h3>Development</h3>Jeffrey Wilcke<br>Viktor Trón<br>Felix Lange<br>Taylor Gerring<br>Daniel Nagy<br><h3>UX</h3>Alex van de Sande<br>"
             }
         }

         Window {
             id: txImportDialog
             minimumWidth: 270
             maximumWidth: 270
             maximumHeight: 50
             minimumHeight: 50
             TextField {
                 id: txImportField
                 width: 170
                 anchors.verticalCenter: parent.verticalCenter
                 anchors.left: parent.left
                 anchors.leftMargin: 10
                 onAccepted: {
                 }
             }
             Button {
                 anchors.left: txImportField.right
                 anchors.verticalCenter: parent.verticalCenter
                 anchors.leftMargin: 5
                 text: "Import"
                 onClicked: {
                     eth.importTx(txImportField.text)
                     txImportField.visible = false
                 }
             }
             Component.onCompleted: {
                 addrField.focus = true
             }
         }

         Window {
             id: addPeerWin
             visible: false
             minimumWidth: 400
             maximumWidth: 400
             maximumHeight: 50
             minimumHeight: 50
             title: "Connect to peer"

             TextField {
                 id: addrField
                 anchors.verticalCenter: parent.verticalCenter
                 anchors.left: parent.left
                 anchors.right: addPeerButton.left
                 anchors.leftMargin: 10
                 anchors.rightMargin: 10
		 placeholderText: "enode://<hex node id>:<IP address>:<port>"
                 onAccepted: {
	             if(addrField.text.length != 0) {
			eth.connectToPeer(addrField.text)
			addPeerWin.visible = false
		     }
                 }
             }

             Button {
                 id: addPeerButton
                 anchors.right: parent.right
                 anchors.verticalCenter: parent.verticalCenter
                 anchors.rightMargin: 10
                 text: "Connect"
                 onClicked: {
	             if(addrField.text.length != 0) {
			eth.connectToPeer(addrField.text)
			addPeerWin.visible = false
		     }
                 }
             }
             Component.onCompleted: {
                 addrField.focus = true
             }
         }
     }
