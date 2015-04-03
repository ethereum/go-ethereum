// Button.qml
import QtQuick 2.0
import QtGraphicalEffects 1.0
import QtQuick.Controls 1.3
import QtQuick.Controls.Styles 1.3
import QtQuick.Layouts 1.1
import QtQuick.Dialogs 1.1


Rectangle {              
    id: wizardWindow
     anchors.fill: parent
     color: "blue"
     state: "State-Initial"
    
    Timer {
        id: startTimerAnimation
        interval: 1000
        running: true
        onTriggered: wizardWindow.state = "State-0"
    }

     Rectangle {
        id: skyGradient
        anchors.left: parent.left
        anchors.top: parent.top
        anchors.right: parent.right
        height: 600
        
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#F4F0FC" }
            GradientStop { position: 1.0; color: "#FFFFFF" }
        }
     }

     Rectangle {
        id: oceanGradient
        anchors.left: parent.left
        anchors.top: skyGradient.bottom
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#385399" }
            GradientStop { position: 1.0; color: "#11264D" }
        }
     }

    Image {
        id: mistTitle
        source: "../wizard/Mist-title.png"
        width: 144
        height: 56
        x: 135
        y: 610
        opacity: 0.2
     }

     Rectangle {
        height: 124
        anchors.bottom: skyGradient.bottom
        anchors.left: parent.left
        anchors.right: parent.right


        gradient: Gradient {
            GradientStop { position: 0.0; color: "transparent" }
            GradientStop { position: 1.0; color: "#FFFFFF" }
        }
     }

     Image {
        id: iceberg
        source: "../wizard/iceberg.png"
        width: 344
        height: 376
        x: 25
        y: 510
        visible: false
        opacity: 0
     }

     FastBlur {
        id: icebergBlur
        anchors.fill: iceberg
        source: iceberg
        radius: 1
        transparentBorder: true
    }     

    Image {
        id: icebergFront
        source: "../wizard/iceberg-front.png"
        width: 344
        height: 376
        x: 25
        y: 510
        visible: false
     }

    FastBlur {
        id: icebergFrontBlur
        width: 344
        height: 376
        x: 25
        y: 510
        source: icebergFront
        radius: 0
        opacity: 1.0
        transparentBorder: true
        visible: true
    }

    Rectangle {
        anchors.left: parent.left
        anchors.top: skyGradient.bottom
        anchors.right: parent.right
        height: oceanGradient.height / 10
            
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#385399" }
            GradientStop { position: 1.0; color: "transparent" }
        }
     }    


     /********************/
     /*      STATES      */
     /********************/


     states: [
        State {
            name: "State-Initial"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
                width: 1854
                height: 2256
                x: -147
                y: -260
            }
            PropertyChanges {
                target: icebergBlur
                opacity: 1.0
                radius: 64
            }
            PropertyChanges {
                target: icebergFrontBlur
                opacity: 1.0
                radius: 64
                width: 2472
                height: 3008
                x: -256
                y: -440
            }
            PropertyChanges {
                target: mistTitle
                y: 280
                opacity: 0.1
            }
            PropertyChanges {
                target: step0
                opacity: 0.0
                x: 600
            }
            PropertyChanges {
                target: skyGradient
                height: 280
            }
            
        },
        State {
            name: "State-0"
            PropertyChanges {
                target: iceberg
                x: 25
                y: 190
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: 25
                y: 190
            }
            PropertyChanges {
                target: icebergBlur
                opacity: 1.0
                radius: 0
            }
            PropertyChanges {
                target: mistTitle
                y: 120
                opacity: 1.0
            }
            PropertyChanges {
                target: step0
                opacity: 1.0
                x: 500
            }
            PropertyChanges {
                target: skyGradient
                height: 280
            }
        },
        State {
            name: "State-Presale-01"
            PropertyChanges {
                target: iceberg
                x: 19
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: 17
            }
            PropertyChanges {
                target: mistTitle
                y: 610
                opacity: 0.2
            }
            PropertyChanges {
                target: step0
                opacity: 0.0
                x: 1000
            } 
            PropertyChanges { 
                target: stepPresale1
                visible: true
                opacity: 1
            }
        },
        State {
            name: "State-Invitation-01"
            PropertyChanges {
                target: iceberg
                x: 19
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: 17
            }
            PropertyChanges {
                target: mistTitle
                y: 610
                opacity: 0.2
            }
            PropertyChanges {
                target: step0
                opacity: 0.0
                x: 1000
            } 
            PropertyChanges { 
                target: stepInvitation1
                visible: true
                opacity: 1
            }
        },        
        State {
            name: "State-Presale-02"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
                x: 13
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: 9
            }
            PropertyChanges { 
                target: stepPresale1
                visible: true
                opacity: 0
            }
            PropertyChanges { 
                target: stepPresale2
                visible: true
                opacity: 1
            }
        },
        State {
            name: "State-Presale-03"
            PropertyChanges {
                target: iceberg
                x: 7
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: 1
            }
            PropertyChanges { 
                target: stepPresale2
                visible: true
                opacity: 0
            }
            PropertyChanges { 
                target: stepPresale3
                visible: true
                opacity: 1
            }
        },
        State {
            name: "State-Presale-04"
            PropertyChanges {
                target: iceberg
                x: 1
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: -7
            }
            PropertyChanges { 
                target: stepPresale3
                visible: true
                opacity: 0
            }
            PropertyChanges { 
                target: stepPresale4
                visible: true
                opacity: 1
            }
        },
        State {
            name: "State-Presale-05"
            PropertyChanges {
                target: iceberg
                x: -5
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: -15
            }
            PropertyChanges { 
                target: stepPresale4
                visible: true
                opacity: 0
            }
            PropertyChanges { 
                target: stepPresale5
                visible: true
                opacity: 1
            }
        },
        State {
            name: "State-Presale-06"
            PropertyChanges {
                target: iceberg
                x: -11
            }
            PropertyChanges {
                target: icebergFrontBlur
                x: -23
            }
            PropertyChanges { 
                target: stepPresale5
                visible: true
                opacity: 0
            }
            PropertyChanges { 
                target: stepPresale6
                visible: true
                opacity: 1
            }
        }
     ]

    transitions: [
       Transition {
           from: "State-Initial"; to: "State-0"
           
           SequentialAnimation {

            ParallelAnimation {
               PropertyAnimation { 
                    target: iceberg
                    properties: "opacity, width, height, x, y"
                    duration: 2000 
                    easing.type: Easing.OutExpo
               }
               PropertyAnimation { 
                    target: icebergBlur 
                    properties: "opacity, radius"
                    duration: 2000 
                    easing.type: Easing.OutExpo
               }
               PropertyAnimation { 
                    target: icebergFrontBlur 
                    properties: "opacity, radius, width, height, x, y"
                    duration: 2000 
                    easing.type: Easing.OutExpo
               }
                }
            
            ParallelAnimation {
               PropertyAnimation { 
                    target: mistTitle 
                    properties: "opacity, y"
                    duration: 1000 
                    easing.type: Easing.OutExpo
               }
               PropertyAnimation { 
                    target: step0 
                    properties: "opacity, x"
                    duration: 2000 
                    easing.type: Easing.OutElastic
               }
              }
           }
       }, 
       Transition {
          from: "State-0"; to: "State-Presale-01"

           PropertyAnimation { 
                target: skyGradient 
                properties: "height"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: step0 
                properties: "opacity, x"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: iceberg 
                properties: "x, y"
                duration: 1000 
                easing.type: Easing.InOutQuart
           } 
           PropertyAnimation { 
                target: icebergFrontBlur 
                properties: "x, y"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: mistTitle
                properties: "opacity, y"
                duration: 1500 
                easing.type: Easing.InOutQuart                
           }
           
       },
       Transition {
          from: "State-0"; to: "State-Invitation-01"

           PropertyAnimation { 
                target: skyGradient 
                properties: "height"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: step0 
                properties: "opacity, x"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: iceberg 
                properties: "x, y"
                duration: 1000 
                easing.type: Easing.InOutQuart
           } 
           PropertyAnimation { 
                target: icebergFrontBlur 
                properties: "x, y"
                duration: 1000 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: mistTitle
                properties: "opacity, y"
                duration: 1500 
                easing.type: Easing.InOutQuart                
           }
           
       },
       Transition {
          from: "State-Presale-01"; to: "State-0"

           PropertyAnimation { 
                target: skyGradient 
                properties: "height"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: step0 
                properties: "opacity, x"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: iceberg 
                properties: "x, y"
                duration: 500 
                easing.type: Easing.InOutQuart
           } 
           PropertyAnimation { 
                target: icebergFrontBlur 
                properties: "x, y"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: mistTitle
                properties: "opacity, y"
                duration: 750 
                easing.type: Easing.InOutQuart                
           }
           
       },
       Transition {
          from: "State-Invitation-01"; to: "State-0"

           PropertyAnimation { 
                target: skyGradient 
                properties: "height"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation { 
                target: step0 
                properties: "opacity, x"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: iceberg 
                properties: "x, y"
                duration: 500 
                easing.type: Easing.InOutQuart
           } 
           PropertyAnimation { 
                target: icebergFrontBlur 
                properties: "x, y"
                duration: 500 
                easing.type: Easing.InOutQuart
           }
           PropertyAnimation {
                target: mistTitle
                properties: "opacity, y"
                duration: 750 
                easing.type: Easing.InOutQuart                
           }
           
       }, 
       Transition {
          // all others

           PropertyAnimation { 
                target: iceberg 
                properties: "x, y"
                duration: 2000 
                easing.type: Easing.InOutCubic
           } 
           PropertyAnimation { 
                target: icebergFrontBlur 
                properties: "x, y"
                duration: 2000 
                easing.type: Easing.InOutCubic
           }
           
           
       }
    ]

    //
    //   STEPS
    //

     Rectangle {
         id: step0
         x: 1000
         y: 0
         width: 475
         height: 670
         color: "transparent"
         opacity: 0

         Text {
            text: "Mist is a Navigator for Ethereum, a decentralized"
            font.family: sourceSansPro.name 
            font.weight: Font.Light
            font.pixelSize: 22
            color: "#2B519E"
            x: 20
            y: 70
         }  
         
         Text {
            text: "app platform. It enables anyone to run programs that can transfer assets or information safely and privately between consenting parties to the contracts."
            font.family: sourceSansPro.name 
            font.weight: Font.Light
            font.pixelSize: 18
            wrapMode: Text.WordWrap
            width: 420
            color: "#2B519E"
            x: 20
            y: 100
         } 

         Text {
            text: "In order to use mist, you need to have an invitation or have previously acquired Ethers or Bitcoins."
            font.family: sourceSansPro.name 
            font.weight: Font.Bold
            font.pixelSize: 18
            wrapMode: Text.WordWrap
            width: 420
            color: "#2B519E"
            x: 20
            y: 190
         } 

         Rectangle {
            x: 10
            y: 320
            color: "transparent"
            width: 430
            height: 70

            Image {
                source: "../wizard/start-invitation.png"
                width: 65
                height: 65
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                text: "I have an invitation"
                font.family: sourceSansPro.name 
                font.weight: Font.SemiBold
                font.pixelSize: 24
                color: "#FFFFFF"
                anchors.verticalCenter: parent.verticalCenter
                x: 80
             } 

             MouseArea {
                anchors.fill: parent
                onClicked: {
                    wizardWindow.state = "State-Invitation-01"
                }
             }
         }
         
         Rectangle {
            x: 10
            y: 420
            width: 430
            height: 50
            color: "transparent"            

            Image {
                source: "../wizard/start-presale.png"
                width: 44
                height: 44
                x: 10
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                text: "Redeem presale ether"
                font.family: sourceSansPro.name 
                font.weight: Font.SemiBold
                font.pixelSize: 18
                color: "#FFFFFF"
                anchors.verticalCenter: parent.verticalCenter
                x: 80
             }

             MouseArea {
                anchors.fill: parent
                onClicked: {
                    wizardWindow.state = "State-Presale-01"
                }
             }
         }

         Rectangle {
            x: 10
            y: 480
            width: 430
            height: 50
            color: "transparent"

            Image {
                source: "../wizard/start-recover.png"
                width: 44
                height: 44
                x: 10
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                text: "Restore old wallet"
                font.family: sourceSansPro.name 
                font.weight: Font.SemiBold
                font.pixelSize: 18
                color: "#FFFFFF"
                anchors.verticalCenter: parent.verticalCenter
                x: 80
             }

             MouseArea {
                anchors.fill: parent
             }
         }

         Rectangle {
            x: 10
            y: 540
            width: 430
            height: 50
            color: "transparent"

            Image {
                source: "../wizard/start-bitcoin.png"
                width: 44
                height: 44
                x: 10
                anchors.verticalCenter: parent.verticalCenter
            }

            Text {
                text: "Use Bitcoins instead"
                font.family: sourceSansPro.name 
                font.weight: Font.SemiBold
                font.pixelSize: 18
                color: "#FFFFFF"
                anchors.verticalCenter: parent.verticalCenter
                x: 80
             }

             MouseArea {
                anchors.fill: parent
             }
         }         

         Rectangle {
            x: 10
            y: 600
            width: 430
            height: 50
            color: "transparent"

            Text {
                text: "I don't have any of those, just let me in anyway.."
                font.family: sourceSansPro.name 
                font.weight: Font.SemiBold
                font.italic: true
                font.pixelSize: 18
                color: "#FFFFFF"
                anchors.verticalCenter: parent.verticalCenter
                x: 10
             }

             MouseArea {
                anchors.fill: parent
             }
         }         

         
    }


     /********************/
     /*    PRESALE 01    */
     /********************/



     Rectangle {
         id: stepPresale1
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }

         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"

            Text {
                text: "Import your Presale Wallet"
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 30
                color: "#57637B"
                x: 0
                y: 90
             } 
             Text {
                text: "When you bought ether, you were given a wallet file with a name similar to  wallet-SOMEADDRESS.json. \n \nFind it and drag it to this box."
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 18
                wrapMode: Text.WordWrap
                width: 320
                color: "#57637B"
                x: 260
                y: 200
             } 

             Rectangle {
                width: 220
                height: 150
                color: "#FFFFFF"
                border.width: 4
                border.color: "#E1D9D9"
                radius: 4
                y: 200

                Rectangle {
                    id: dropIcon
                    anchors {
                        top: parent.top
                        left: parent.left
                        right: parent.right
                    }
                    height: 100
                    color: "transparent"
                    
                    Image {
                        anchors.centerIn: parent
                        source: "../wizard/drop-icon.png"
                    }

                }

                Text {
                   anchors {
                        top: dropIcon.bottom
                        left: parent.left
                        right: parent.right
                        bottom: parent.bottom
                    }
                    text: "DROP OR OPEN YOUR WALLET.JSON HERE"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Bold
                    font.pixelSize: 14
                    font.italic: true
                    wrapMode: Text.WordWrap
                    color: "#4A90E2"
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignTop

                }
             }
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-0";
                    console.log( "state-0 bak")
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-02";
                 }

            }
         }
     }




     /********************/
     /*    PRESALE 02    */
     /********************/





     Rectangle {
         id: stepPresale2
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }

         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"

            Text {
                text: "Choose a new Passphrase"
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 30
                color: "#57637B"
                x: 0
                y: 90
             } 
             Text {
                text: "We will now create a new wallet to store securely your funds. Create a new passphrase to lock this wallet. You’ll need to type it everytime you want to make a new transfer."
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 18
                wrapMode: Text.WordWrap
                width: 450
                color: "#57637B"
                x: 0
                y: 150
             } 

             

            TextField {
               y: 240
               height: 90
               width: 400
               placeholderText: "Type a Passphrase"
               font.family: sourceSansPro.name 
               font.weight: Font.Light
               font.pixelSize: 30
               font.italic: true
               horizontalAlignment: Text.AlignLeft
               verticalAlignment: Text.AlignVCenter
               echoMode: TextInput.Password

               style: TextFieldStyle {
                    background: Rectangle {
                        radius: 4
                        border.color: "#E1D9D9"
                        border.width: 4
                    }
                }

            }

            TextField {
               y: 360
               height: 90
               width: 400
               placeholderText: "Type it again"
               font.family: sourceSansPro.name 
               font.weight: Font.Light
               font.pixelSize: 30
               font.italic: true
               horizontalAlignment: Text.AlignLeft
               verticalAlignment: Text.AlignVCenter
               echoMode: TextInput.Password

               style: TextFieldStyle {
                    background: Rectangle {
                        radius: 4
                        border.color: "#E1D9D9"
                        border.width: 4
                    }
                }
            }         
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-01";
                    console.log( "state-0 bak")
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-03";
                 }

            }
         }
     }




     /********************/
     /*    PRESALE 03    */
     /********************/



    Rectangle {
         id: stepPresale3
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }
     


         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"
            
            ColumnLayout {
                Text {
                    text: "Write these magic words down"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    verticalAlignment: Text.AlignBottom
                    font.pixelSize: 30
                    color: "#57637B"
                    Layout.minimumHeight: 120
                 } 

                 Rectangle {
                    color: "transparent"
                    width: 450
                    height: 80

                    Text {
                        anchors.fill: parent
                        text: "It might look like a weird little poem works but it a security key that allows you to instant recover all your funds."
                        font.family: sourceSansPro.name 
                        font.weight: Font.Light
                        font.pixelSize: 18
                        wrapMode: Text.WordWrap
                        verticalAlignment: Text.AlignVCenter
                        color: "#57637B"
                     }

                 } 

                 
                Rectangle {
                    radius: 4
                    border.color: "#E1D9D9"
                    border.width: 4
                    color: "#FFFFFF"
                    y: 220
                   height: 300
                   width: 540
                
                    TextEdit {
                        id: magicWordsText
                        x: 20
                        y: 20
                        width: 400
                        height: 260
                        text: "proof adjust second surge \nshrimp arrive tunnel spare \ncoral man stand column \n \nfox sheriff ketchup arrest \nnews abstract pioneer inner \ninside come benefit feed"
                        font.family: sourceSansPro.name 
                        font.weight: Font.SemiBold
                        font.pixelSize: 22
                        wrapMode: Text.WordWrap
                        horizontalAlignment: Text.AlignLeft
                        verticalAlignment: Text.AlignTop
                        focus: false
                        color: "#AAA0A0"
                    }

                    Rectangle {
                        width: 90
                        height: 30
                        x: 430
                        y: 40
                        color: "transparent"

                        Image {
                            source: "../wizard/print icon.png"
                            width: 40
                            height: 40
                            anchors.verticalCenter: parent.top
                            anchors.horizontalCenter: parent.horizontalCenter
                        }

                        Text {
                            text: "PRINT WORDS"
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            font.pixelSize: 14
                            color: "#4A90E2"                            
                            anchors.verticalCenter: parent.bottom
                            anchors.horizontalCenter: parent.horizontalCenter
                        }
                    }
                    
                }

                
                RowLayout {
                    MouseArea {
                        height: 30
                        width: 90

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                         }

                         Text {
                            id: englishTab
                            text: "English"
                            anchors.fill: parent
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            color: "#555F72"
                            font.pixelSize: 18
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter    
                         }

                         onClicked: {
                            magicWordsText.text = "proof adjust second surge \nshrimp arrive tunnel spare \ncoral man stand column \n \nfox sheriff ketchup arrest \nnews abstract pioneer inner \ninside come benefit feed";
                            
                            englishTab.color = "#555F72";
                            shortAlienTab.color = "#4A90E2";
                            simpleChineseTab.color = "#4A90E2";
                            japaneseTab.color = "#4A90E2";
                            spanishTab.color = "#4A90E2";
                         }
                    }
                    MouseArea {
                        height: 30
                        Layout.minimumWidth: 90

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                         }

                         Text {
                            id: shortAlienTab
                            text: "Short Alien"
                            anchors.fill: parent
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            color: "#4A90E2"
                            font.pixelSize: 18
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter    
                         }

                         onClicked: {
                            magicWordsText.text = "alajog wor uw uwuriw giwnin \nkowbu enlan ojimow puwvoj \njolgag wilrad zijhuw zad";
                            
                            englishTab.color = "#4A90E2";
                            shortAlienTab.color = "#555F72";
                            simpleChineseTab.color = "#4A90E2";
                            japaneseTab.color = "#4A90E2";
                            spanishTab.color = "#4A90E2";
                         }
                    }
                    MouseArea {
                        height: 30
                        Layout.minimumWidth: 160

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                         }

                         Text {
                            id: simpleChineseTab
                            text: "Simplified Chinese"
                            anchors.fill: parent
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            color: "#4A90E2"
                            font.pixelSize: 18
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter    
                         }

                         onClicked: {
                            magicWordsText.text = "是在不了\n有和人这\n中大为上\n个国我以\n要他时来\n用们生到";
                            
                            englishTab.color = "#4A90E2";
                            shortAlienTab.color = "#4A90E2";
                            simpleChineseTab.color = "#555F72";
                            japaneseTab.color = "#4A90E2";
                            spanishTab.color = "#4A90E2";
                         }

                    }

                    MouseArea {
                        height: 30
                        Layout.minimumWidth: 90

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                         }

                         Text {
                            id: japaneseTab
                            text: "Japanese"
                            anchors.fill: parent
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            color: "#4A90E2"
                            font.pixelSize: 18
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter    
                         }

                         onClicked: {
                            magicWordsText.text = " あいこくしん あいさつ あいだ あおぞら \nあかちゃん あきる あけがた あける \nあこがれる あさい あさひ あしあと \n\nあじわう あずかる あずき あそぶ \nあたえる あたためる あたりまえ あたる \nあつい あつかう あっしゅく あつまり";
                            
                            englishTab.color = "#4A90E2";
                            shortAlienTab.color = "#4A90E2";
                            simpleChineseTab.color = "#4A90E2";
                            japaneseTab.color = "#555F72";
                            spanishTab.color = "#4A90E2";
                         }

                    }
                    MouseArea {
                        height: 30
                        Layout.minimumWidth: 90

                        Rectangle {
                            anchors.fill: parent
                            color: "transparent"
                         }

                         Text {
                            id: spanishTab
                            text: "Spanish"
                            anchors.fill: parent
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            color: "#4A90E2"
                            font.pixelSize: 18
                            horizontalAlignment: Text.AlignHCenter
                            verticalAlignment: Text.AlignVCenter    
                         }

                         onClicked: {
                            magicWordsText.text = "parcela rico bebida pauta \npasta torpedo yoga médula \ninfiel manso dureza tronco \n\nrancho mente útil retorno \npasta médula pauta yoga  \nbebida manso tronco dureza";
                            
                            englishTab.color = "#4A90E2";
                            shortAlienTab.color = "#4A90E2";
                            simpleChineseTab.color = "#4A90E2";
                            japaneseTab.color = "#4A90E2";
                            spanishTab.color = "#555F72";
                         }

                    }
                }
            }   
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-02";
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                   wizardWindow.state = "State-Presale-04";
                 }

            }
         }
     }



     /********************/
     /*    PRESALE 04    */
     /********************/



    Rectangle {
         id: stepPresale4
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }
     


         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"
            
            ColumnLayout {

                spacing: 20 

                Text {
                    text: "Type the magic words again"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    verticalAlignment: Text.AlignBottom
                    font.pixelSize: 30
                    color: "#57637B"
                    Layout.minimumHeight: 120
                 } 

                 Rectangle {
                    color: "transparent"
                    width: 450
                    height: 80

                    Text {
                        anchors.fill: parent
                        text: "We do this to ensure you have the exact words. As soon as you finish this, keep that paper in a secure place, it’s the only way to securely recover your funds if something happens to this computer."
                        font.family: sourceSansPro.name 
                        font.weight: Font.Light
                        font.pixelSize: 18
                        wrapMode: Text.WordWrap
                        verticalAlignment: Text.AlignVCenter
                        color: "#57637B"
                     }

                 } 

                 
                Rectangle {
                    radius: 4
                    border.color: "#E1D9D9"
                    border.width: 4
                    color: "#FFFFFF"
                    y: 220
                    height: 220
                    width: 540
                
                    TextEdit {
                        id: typeMagicWords
                        x: 20
                        y: 20
                        width: 400
                        height: 180
                        text: ""
                        font.family: sourceSansPro.name 
                        font.weight: Font.SemiBold
                        font.pixelSize: 22
                        wrapMode: Text.WordWrap
                        horizontalAlignment: Text.AlignLeft
                        verticalAlignment: Text.AlignTop
                        focus: false
                        color: "#AAA0A0"

                        onTextChanged: {
                            if (typeMagicWords.text == "proof adjust second surge") {
                                iconPositiveFeedback.visible = true;
                                iconNegativeFeedback.visible = false;
                                nextButton03.visible = true;
                            } else if (typeMagicWords.text.length > 24) {
                                iconPositiveFeedback.visible = false;
                                iconNegativeFeedback.visible = true;                                
                                nextButton03.visible = false;                                
                            } else {
                                iconPositiveFeedback.visible = false;
                                iconNegativeFeedback.visible = false;                                
                                nextButton03.visible = false;                            
                            }

                        }
                    }
                }

                RowLayout {
                    id: iconPositiveFeedback
                    spacing: 20
                    visible: false

                    Image {
                        source: "../wizard/icon-correct.png"
                        width: 34
                        height: 34
                    }
                
                    Text {
                        text: "Yep, you've got them right"
                        font.family: sourceSansPro.name 
                        font.weight: Font.SemiBold
                        verticalAlignment: Text.AlignBottom
                        font.pixelSize: 16
                        color: "#679137"
                     } 
                }

                RowLayout {
                    id: iconNegativeFeedback
                    spacing: 20
                    visible: false

                    Image {
                        source: "../wizard/icon-wrong.png"
                        width: 34
                        height: 34
                    }
                
                    Text {
                        text: "Wrong words. Keep trying."
                        font.family: sourceSansPro.name 
                        font.weight: Font.SemiBold
                        verticalAlignment: Text.AlignBottom
                        font.pixelSize: 16
                        color: "#D0021B"
                     } 
                }
            }   
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-03";
                 }

            }

            MouseArea {
                id: nextButton03
               // visible: false
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-05";
                 }

            } // MouseArea
         } // Bottom Buttons
     }// End State



     /********************/
     /*    PRESALE 05    */
     /********************/



    Rectangle {
         id: stepPresale5
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }
     


         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"
            
            Column {

                spacing: 20 

                Text {
                    text: "Signup for  secondary confirmation service"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    verticalAlignment: Text.AlignBottom
                    font.pixelSize: 30
                    color: "#57637B"
                    height: 120
                 } 

                 Rectangle {
                    color: "transparent"
                    width: 450
                    height: 80

                    Text {
                        anchors.fill: parent
                        text: "For security measures, any transaction that goes over a daily limit of 1000 ethers will require a secondary authentication. You will be able to create new wallets with custom limits as soon as you start mist."
                        font.family: sourceSansPro.name 
                        font.weight: Font.Light
                        font.pixelSize: 18
                        wrapMode: Text.WordWrap
                        verticalAlignment: Text.AlignVCenter
                        color: "#57637B"
                    }
                }


                ScrollView {
                    width: parent.width
                    height: 300

                    Column {
                        spacing: 20

                        ExclusiveGroup { id: secondFactorAuth }
                        
                        Rectangle {
                            height: 20
                            width: 100
                            color: "transparent" 
                            // QML is a definite improvement over table layout and transparent gifs..
                        }
                        Rectangle {
                            height: radioSecAuthOption01.checked ? 220 : 30
                            width: 500
                            color: "transparent"
                            anchors.topMargin: 100

                            RadioButton {
                                id: radioSecAuthOption01
                                exclusiveGroup: secondFactorAuth

                                style: RadioButtonStyle {
                                    indicator: Rectangle {
                                            implicitWidth: 16
                                            implicitHeight: 16
                                            radius: 9
                                            //border.color: control.activeFocus ? "darkblue" : "gray"
                                            border.color: "#E9E0E0"
                                            color: "#F5EEEE"
                                            border.width: 4

                                            Rectangle {
                                                anchors.fill: parent
                                                visible: control.checked
                                                color: "#4A90E2"
                                                radius: 9
                                                anchors.margins: 4
                                            }
                                    }
                                    label: Text {
                                            text: "FIRST PROVIDER"
                                            font.pixelSize: 14
                                            font.family: sourceSansPro.name 
                                            font.weight: Font.SemiBold
                                            color: "#4A90E2"
                                            x: 10
                                            y: -8
                                        }
                                }
                            }

                            Text {
                                x: 30
                                y: 8
                                text: "Some site that we are partnering with"
                                font.family: sourceSansPro.name 
                                font.weight: Font.SemiBold
                                font.pixelSize: 14
                                color: "#57637B"

                            }

                            Rectangle {
                                visible: radioSecAuthOption01.checked
                                width: 400
                                height: 190
                                y: 40
                                x: 20
                                color: "transparent"
                                
                                Text {
                                    id: link_Text

                                    text: '<html><p> 1) Create an account in <a href="http://example.com" style="color: #5AA3E8;">example.com</a> </p> <p>2) Go to Settings > 2FA > Ethereum </p> <p>3) Copy this code on the API KEY field: </p> </html>'
                                    onLinkActivated: Qt.openUrlExternally(link)
                                    font.family: sourceSansPro.name 
                                    font.weight: Font.SemiBold
                                    font.pixelSize: 14
                                    wrapMode: Text.WordWrap
                                    verticalAlignment: Text.AlignVCenter
                                    color: "#57637B"
                                }

                                Rectangle {
                                    radius: 4
                                    border.color: "#E1D9D9"
                                    border.width: 4
                                    color: "#FFFFFF"
                                    y: 100
                                    width: 300
                                    height: 50
                                    
                                    TextEdit {
                                        anchors.fill: parent
                                        text: "a8f5f167f44f4964e6c998dee827110c56831c0fa…"
                                        font.family: sourceSansPro.name 
                                        font.weight: Font.SemiBold
                                        font.pixelSize: 14
                                        wrapMode: Text.WordWrap
                                        verticalAlignment: Text.AlignVCenter
                                        horizontalAlignment: Text.AlignHCenter
                                        focus: true
                                        color: "#5AA3E8"
                                    }
                                }
                            }
                        }


                        Rectangle {
                            height: radioSecAuthOption02.checked ? 220 : 30
                            width: 500
                            color: "transparent"
                            anchors.topMargin: 100

                            RadioButton {
                                id: radioSecAuthOption02
                                exclusiveGroup: secondFactorAuth

                                style: RadioButtonStyle {
                                    indicator: Rectangle {
                                            implicitWidth: 16
                                            implicitHeight: 16
                                            radius: 9
                                            //border.color: control.activeFocus ? "darkblue" : "gray"
                                            border.color: "#E9E0E0"
                                            color: "#F5EEEE"
                                            border.width: 4

                                            Rectangle {
                                                anchors.fill: parent
                                                visible: control.checked
                                                color: "#4A90E2"
                                                radius: 9
                                                anchors.margins: 4
                                            }
                                    }
                                    label: Text {
                                            text: "SECOND PROVIDER"
                                            font.pixelSize: 14
                                            font.family: sourceSansPro.name 
                                            font.weight: Font.SemiBold
                                            color: "#4A90E2"
                                            x: 10
                                            y: -8
                                        }
                                }
                            }

                            Text {
                                x: 30
                                y: 8
                                text: "Another nice we recommend"
                                font.family: sourceSansPro.name 
                                font.weight: Font.SemiBold
                                font.pixelSize: 14
                                color: "#57637B"
                            }

                            Rectangle {
                                visible: radioSecAuthOption02.checked
                                width: 400
                                height: 190
                                y: 40
                                x: 20
                                color: "transparent"
                                
                                Text {
                                    text: '<html><p> 1) Create an account in <a href="http://example.com" style="color: #5AA3E8;">example.com</a> </p> <p>2) Go to Settings > 2FA > Ethereum </p> <p>3) Copy this code on the API KEY field: </p> </html>'
                                    onLinkActivated: Qt.openUrlExternally(link)
                                    font.family: sourceSansPro.name 
                                    font.weight: Font.SemiBold
                                    font.pixelSize: 14
                                    wrapMode: Text.WordWrap
                                    verticalAlignment: Text.AlignVCenter
                                    color: "#57637B"
                                }

                                Rectangle {
                                    radius: 4
                                    border.color: "#E1D9D9"
                                    border.width: 4
                                    color: "#FFFFFF"
                                    y: 100
                                    width: 300
                                    height: 50
                                    
                                    TextEdit {
                                        anchors.fill: parent
                                        text: "a8f5f167f44f4964e6c998dee827110c56831c0fa…"
                                        font.family: sourceSansPro.name 
                                        font.weight: Font.SemiBold
                                        font.pixelSize: 14
                                        wrapMode: Text.WordWrap
                                        verticalAlignment: Text.AlignVCenter
                                        horizontalAlignment: Text.AlignHCenter
                                        focus: true
                                        color: "#5AA3E8"
                                    }
                                }
                            }
                        }


                        Rectangle {
                            height: radioSecAuthOption03.checked ? 220 : 30
                            width: 500
                            color: "transparent"
                            anchors.topMargin: 100

                            RadioButton {
                                id: radioSecAuthOption03
                                exclusiveGroup: secondFactorAuth

                                style: RadioButtonStyle {
                                    indicator: Rectangle {
                                            implicitWidth: 16
                                            implicitHeight: 16
                                            radius: 9
                                            border.color: "#E9E0E0"
                                            color: "#F5EEEE"
                                            border.width: 4

                                            Rectangle {
                                                anchors.fill: parent
                                                visible: control.checked
                                                color: "#4A90E2"
                                                radius: 9
                                                anchors.margins: 4
                                            }
                                    }
                                    label: Text {
                                            text: "USE ANOTHER COMPUTER"
                                            font.pixelSize: 14
                                            font.family: sourceSansPro.name 
                                            font.weight: Font.SemiBold
                                            color: "#4A90E2"
                                            x: 10
                                            y: -8
                                        }
                                }
                            }

                            Text {
                                x: 30
                                y: 8
                                text: "I have access to another device I can install Mist"
                                font.family: sourceSansPro.name 
                                font.weight: Font.SemiBold
                                font.pixelSize: 14
                                color: "#57637B"
                            }

                            Rectangle {
                                visible: radioSecAuthOption03.checked
                                width: 400
                                height: 190
                                y: 40
                                x: 20
                                color: "transparent"
                                
                                Text {
                                    text: '<html><p> 1) Install Mist on your second computer </p> <p>2) On the install process choose "import wallet" </p> <p>3) Put this code on the import field: </p> </html>'
                                    onLinkActivated: Qt.openUrlExternally(link)
                                    font.family: sourceSansPro.name 
                                    font.weight: Font.SemiBold
                                    font.pixelSize: 14
                                    wrapMode: Text.WordWrap
                                    verticalAlignment: Text.AlignVCenter
                                    color: "#57637B"
                                }

                                Rectangle {
                                    radius: 4
                                    border.color: "#E1D9D9"
                                    border.width: 4
                                    color: "#FFFFFF"
                                    y: 100
                                    width: 300
                                    height: 50
                                    
                                    TextEdit {
                                        anchors.fill: parent
                                        text: "correct horse battery staple"
                                        font.family: sourceSansPro.name 
                                        font.weight: Font.SemiBold
                                        font.pixelSize: 14
                                        wrapMode: Text.WordWrap
                                        verticalAlignment: Text.AlignVCenter
                                        horizontalAlignment: Text.AlignHCenter
                                        focus: true
                                        color: "#5AA3E8"
                                    }
                                }
                            }
                        }


                        Rectangle {
                            height: radioSecAuthOption04.checked ? 220 : 30
                            width: 500
                            color: "transparent"
                            anchors.topMargin: 100

                            RadioButton {
                                id: radioSecAuthOption04
                                exclusiveGroup: secondFactorAuth

                                style: RadioButtonStyle {
                                    indicator: Rectangle {
                                            implicitWidth: 16
                                            implicitHeight: 16
                                            radius: 9
                                            border.color: "#E9E0E0"
                                            color: "#F5EEEE"
                                            border.width: 4

                                            Rectangle {
                                                anchors.fill: parent
                                                visible: control.checked
                                                color: "#4A90E2"
                                                radius: 9
                                                anchors.margins: 4
                                            }
                                    }
                                    label: Text {
                                            text: "DON'T USE A SECURE WALLET"
                                            font.pixelSize: 14
                                            font.family: sourceSansPro.name 
                                            font.weight: Font.SemiBold
                                            color: "#4A90E2"
                                            x: 10
                                            y: -8
                                        }
                                }
                            }

                            Text {
                                x: 30
                                y: 8
                                text: "Only for advanced users who know what they are doing"
                                font.family: sourceSansPro.name 
                                font.weight: Font.SemiBold
                                font.pixelSize: 14
                                color: "#57637B"
                            }
                        }
                    }
                }
            }   
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-04";
                 }
            }

            MouseArea {
                id: nextButton04
                visible: true
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-06";
                 }

            } // MouseArea
         } // Bottom Buttons
     }// End State 




     /********************/
     /*    PRESALE 06    */
     /********************/

    Rectangle {
         id: stepPresale6
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }
     


         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"
            
            ColumnLayout {
                Text {
                    text: "You just wrote your first contract!"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    verticalAlignment: Text.AlignBottom
                    font.pixelSize: 30
                    color: "#57637B"
                    Layout.minimumHeight: 120
                 } 

                 Rectangle {
                    color: "transparent"
                    width: 450
                    height: 80

                    Text {
                        anchors.fill: parent
                        text: "Your wallet is a contract. Read it carefully and agree to continue."
                        font.family: sourceSansPro.name 
                        font.weight: Font.Light
                        font.pixelSize: 18
                        wrapMode: Text.WordWrap
                        verticalAlignment: Text.AlignVCenter
                        color: "#57637B"
                     }

                 } 

                 
                Rectangle {
                    radius: 4
                    border.color: "#E1D9D9"
                    border.width: 4
                    color: "#FFFFFF"
                    y: 220
                    width: 540
                    height: 300
                    ScrollView {
                        anchors.centerIn: parent
                        width: 530
                        height: 290

                        Text {
                            x: 20
                            y: 20
                            width: 400
                            height: 400
                            text: "<html>This wallet contract will be initialized with 1000 ETH. The main holder of the contract is <strong> 27f8c208bd409eebd1cfaf1d68ffdb07 </strong> and has the right to spend transactions up to a limit of 100 eth per day. Any transaction above that limit will require a secondary authorization from  <strong> fe6f41cd6a446a81ad30bf38d3e72fa2 </strong> or  <strong> 300c04621d00ea9bad58f08c8220f459 </strong>. This account also has an emergency key, <strong> a272a3c46bf23afc105ada54d72c1cec </strong>, which is able to immediatly move all funds to another account. The main holder can also execute other contracts under this accounts name, these transactions are not subject to any limitations. </html>"
                            font.family: sourceSansPro.name 
                            font.weight: Font.SemiBold
                            font.pixelSize: 16
                            wrapMode: Text.WordWrap
                            horizontalAlignment: Text.AlignLeft
                            verticalAlignment: Text.AlignTop
                            focus: false
                            color: "#AAA0A0"
                        }
                    } 
                }

                
                
            }   
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-05";
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                   wizardWindow.state = "State-Presale-07";
                   //messageDialog.visible = true
                 }
            }
         }
        // MessageDialog {
        //     id: messageDialog
        //     title: "You have to accept this"
        //     text: "Do you agree with this contract?"
        //     standardButtons: StandardButton.Yes | StandardButton.No
        //     onAccepted: {
        //         wizardWindow.state = "State-Presale-04"
        //     }
        //     //Component.onCompleted: visible = true
        // }
     }





     /********************/
     /*    PRESALE 07    */
     /********************/



    Rectangle {
         id: stepPresale7
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false

         Behavior on opacity { PropertyAnimation { duration: 500 } }
     
         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"
            
            ColumnLayout {
                Text {
                    text: "Deployed!"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    verticalAlignment: Text.AlignBottom
                    font.pixelSize: 30
                    color: "#57637B"
                    Layout.minimumHeight: 120
                 } 

                 Rectangle {
                    color: "transparent"
                    width: 450
                    height: 80

                    Text {
                        anchors.fill: parent
                        text: "Everything is ready!"
                        font.family: sourceSansPro.name 
                        font.weight: Font.Light
                        font.pixelSize: 18
                        wrapMode: Text.WordWrap
                        verticalAlignment: Text.AlignVCenter
                        color: "#57637B"
                     }

                 }                                 
            }   
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-06";
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "FINISH"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    //onboardingWizard.visible = false;
                    wizardWindow.state = "State-Initial";
                    //startTimerAnimation.running = true;
                    //startTimerAnimation.start();
                 }
            }
         }
     }




     /********************/
     /*   INVITATION 01  */
     /********************/



     Rectangle {
         id: stepInvitation1
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 500 } }

         Rectangle {
            // Content
             x: 350
             y: 0
             color: "transparent"

            Text {
                text: "Drag your invitation"
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 30
                color: "#57637B"
                x: 0
                y: 90
             } 
             Text {
                text: "An invitation is an image that looks like the icon you see here. Find the one you were given and drag it to this space."
                font.family: sourceSansPro.name 
                font.weight: Font.Light
                font.pixelSize: 18
                wrapMode: Text.WordWrap
                width: 320
                color: "#57637B"
                x: 260
                y: 200
             } 

             Rectangle {
                width: 220
                height: 150
                color: "#FFFFFF"
                border.width: 4
                border.color: "#E1D9D9"
                radius: 4
                y: 200

                Rectangle {
                    id: invitationIcon
                    anchors {
                        top: parent.top
                        left: parent.left
                        right: parent.right
                    }
                    height: 100
                    color: "transparent"
                    
                    Image {
                        anchors.centerIn: parent
                        source: "../wizard/drop-icon.png"
                    }

                }

                Text {
                   anchors {
                        top: invitationIcon.bottom
                        left: parent.left
                        right: parent.right
                        bottom: parent.bottom
                    }
                    text: "<html>DROP OR OPEN YOUR <strong>INVITE</strong> HERE</html>"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Bold
                    font.pixelSize: 14
                    font.italic: true
                    wrapMode: Text.WordWrap
                    color: "#4A90E2"
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignTop

                }
             }
         }

         Rectangle {
            //bottom buttons
            anchors.left: parent.left
            anchors.bottom: parent.bottom
            anchors.right: parent.right
            height: 80
            color: "transparent"

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                x: 60 
                height: 60
                width: 120

                Text {
                    text: "BACK"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignLeft
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-0";
                    console.log( "state-0 bak")
                 }

            }

            MouseArea {
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: 60
                height: 60
                width: 120

                Text {
                    text: "NEXT"
                    font.family: sourceSansPro.name 
                    font.weight: Font.Light
                    font.pixelSize: 24
                    color: "#FFFFFF"
                    anchors.fill: parent
                    horizontalAlignment: Text.AlignRight
                    verticalAlignment: Text.AlignVCenter
                 }

                 onClicked: {
                    wizardWindow.state = "State-Presale-02";
                 }

            }
         }
     }



 }
