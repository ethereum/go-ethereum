function sleep(milliseconds) {
  var start = new Date().getTime();
  for (var i = 0; i < 1e7; i++) {
    if ((new Date().getTime() - start) > milliseconds){
      break;
    }
  }
}

function interaction() {
        console.log("peer count: " + net.peerCount)
	admin.addPeer("enode://a321db2296eb0078fab331d27460456f9c66c478049ee34441595d9c4fc4391af31f24b81fcdf21f28a7e391d07ab4fc550b6e235021abad310992f32d2fc876@127.0.0.1:30303")
	sleep(5000)	
        console.log("peer count: " + net.peerCount)


}

interaction()
