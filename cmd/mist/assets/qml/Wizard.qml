// Button.qml
import QtQuick 2.0
import QtGraphicalEffects 1.0


Rectangle {              
    id: wizardWindow
     anchors.fill: parent
     color: "blue"
     state: "State-Initial"
    
    Timer {
        interval: 1000
        running: true
        onTriggered: wizardWindow.state = "State-0"
    }

     Image {
        anchors.centerIn: parent
        source: "../wizard/illustration-wizard.png"
        height: 680
        width: 993
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
        id: iceberg
        source: "../wizard/iceberg.png"
        width: 344
        height: 376
        x: 25
        y: 190
     }

     FastBlur {
        id: icebergBlur
        anchors.fill: iceberg
        source: iceberg
        radius: 32
        transparentBorder: true
        transform: Scale { 
            origin.x: 172 
            origin.y: 188
            xScale: 0.75
            yScale: 0.75
        }

    }

     Image {
        source: "../wizard/Mist-title.png"
        width: 144
        height: 56
        x: 155
        y: 120
     }


     states: [
        State {
            name: "State-Initial"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
            }
            PropertyChanges {
                target: icebergBlur
                opacity: 0.1
                radius: 64
                //transform: Scale {  xScale: 1;   yScale: 1; }                
            }
            
        },
        State {
            name: "State-0"
            PropertyChanges {
                target: iceberg
                opacity: 0.0
            }
            PropertyChanges {
                target: icebergBlur
                opacity: 1.0
                radius: 0
                // transform: Scale { 
                //     origin.x: 150; 
                //     origin.y: 150; 
                //     xScale: 1
                //     yScale: 1
                // } 
            }
        }
     ]

    transitions: [
       Transition {
           from: "State-Initial"; to: "State-0"
           PropertyAnimation { 
            target: iceberg
            properties: "opacity"
            duration: 2000 
           }
           PropertyAnimation { 
            target: icebergBlur 
            properties: "opacity, radius"
            duration: 2000 
           }
       },
       Transition {
           from: "State-0"; to: "State-Initial"
           PropertyAnimation { 
                target: iceberg            
                properties: "opacity"; 
                duration: 1000 
            }
            PropertyAnimation { 
                target: icebergBlur 
                properties: "opacity, radius"
                duration: 1000 
           }
       }
    ]


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
                    wizardWindow.state = "State-Initial"
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
                    wizardWindow.state = "State-0"
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
 }

 // Rectangle {              
 //                 anchors.fill: parent
 //                 color: "blue"

 //                 Image {
 //                    anchors.centerIn: parent
 //                    source: "../wizard/illustration-wizard.png"
 //                    height: 680
 //                    width: 993
 //                 }

 //                 Rectangle {
 //                     x: 520
 //                     y: 0
 //                     width: 475
 //                     height: 670
                     
 //                     ColumnLayout{
 //                        anchors.fill: parent

 //                        Rectangle {
 //                            color: "red"
 //                            Layout.preferredHeight: 280
 //                            Layout.fillWidth : true
 //                        }

 //                        Rectangle {
 //                            color: "green"
 //                            Layout.preferredHeight: 70
 //                            Layout.fillWidth : true

 //                        }

 //                        Rectangle {
 //                            color: "blue"
 //                            Layout.preferredHeight: 40
 //                            Layout.fillWidth : true
 //                            Layout.fillHeight: true

 //                        }
 //                    }
 //                }
 //             }