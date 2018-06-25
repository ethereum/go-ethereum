function sleep(milliseconds) {
  var start = new Date().getTime();
  for (var i = 0; i < 1e7; i++) {
    if ((new Date().getTime() - start) > milliseconds){
      break;
    }
  }
}

function interaction() {
    console.log("[peermode]: peer count: " + net.peerCount)
    //admin.addPeer("enode://a321db2296eb0078fab331d27460456f9c66c478049ee34441595d9c4fc4391af31f24b81fcdf21f28a7e391d07ab4fc550b6e235021abad310992f32d2fc876@127.0.0.1:30303")

    console.log("[peermode]: connect to [::]:30303")
    admin.addPeer("enode://07a0106f92170dd3fc3ffa201c5b0708c8b3b92f73ad58b6f21a43338eea924b53e0a1d942c70f582127f10fb86b569c10dabcc97fa308a90f23aa6003c5b76c@[::]:30303")
    while (net.peerCount == 0)
    {
        sleep(1000)
        console.log("[peermode]: peer count: " + net.peerCount)
    }
}

interaction()
