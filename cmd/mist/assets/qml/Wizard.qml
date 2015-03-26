// Button.qml
import QtQuick 2.0
import QtGraphicalEffects 1.0


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
        height: 280
        
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#E7E0F4" }
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
        y: 120
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
        y: 190
        visible: false
     }

     FastBlur {
        id: icebergBlur
        anchors.fill: iceberg
        source: iceberg
        radius: 32
        transparentBorder: true
    }     

    Image {
        id: icebergFront
        source: "../wizard/iceberg-front.png"
        width: 344
        height: 376
        x: 25
        y: 190
        visible: false
     }

    FastBlur {
        id: icebergFrontBlur
        width: 344
        height: 376
        x: 25
        y: 190
        source: icebergFront
        radius: 32
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
            
        },
        State {
            name: "State-0"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
                width: 344
                height: 376
                x: 25
                y: 190
            }
            PropertyChanges {
                target: icebergFrontBlur
                opacity: 1.0
                radius: 0
                width: 344
                height: 376
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
        },
        State {
            name: "State-Presale-01"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
                width: 344
                height: 376
                x: 25
                y: 510
            }
            PropertyChanges {
                target: icebergFrontBlur
                opacity: 1.0
                radius: 0
                width: 344
                height: 376
                x: 25
                y: 510
            }
            PropertyChanges {
                target: icebergBlur
                opacity: 1.0
                radius: 0
            }
            PropertyChanges {
                target: mistTitle
                y: 610
                opacity: 0.2
            }
            PropertyChanges {
                target: step0
                opacity: 0.0
                x: 500
            } 
            PropertyChanges {
                target: skyGradient
                height: 600
            }
            PropertyChanges { 
                target: stepPresale1
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
          // from: "State-0"; to: "State-Presale-01"

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
           
       }
    ]

    //
    //   STEPS
    //

     Rectangle {
         id: step0
         x: 500
         y: 0
         width: 475
         height: 670
         color: "transparent"

         
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
                    //wizardWindow.state = "State-1"
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

     Rectangle {
         id: stepPresale1
         anchors.fill: parent
         color: "transparent"
         opacity: 0
         visible: false


         Behavior on opacity { PropertyAnimation { duration: 2000 } }

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
                text: "When you bought ether, you were given a wallet file with a name similar to  wallet-SOMEADDRESS.json. 

Find it and drag it to this box."
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
         }
     }
 }
