function log(text) {
  console.log("[JS TEST SCRIPT] " + text);
}

function sleep(seconds) {
    var now = new Date().getTime();
    while(new Date().getTime() < now + seconds){}
}

