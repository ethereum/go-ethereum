var testname = "TEST_addPeerOnNode2 ";
process.stdout.write(testname);

var onResult = {};
web3.admin.addPeer("enode://" + args["node1_ID"] + "@127.0.0.1:" + args["node1_port"], function(err, res){ onResult(err, res); })

function sleep(ms) {
   return new Promise(resolve => setTimeout(resolve, ms));
}

var onGetPeerCount = {};
onResult = function (err,res) 
{
   //wait for peer being added
   sleep(1000).then(() => {
	web3.net.getPeerCount(function(err, res){ onGetPeerCount(err, res); })
   });
}

onGetPeerCount = function (err, res)
{
   if (res == 1)
	console.log("OK");
   else
   {
	console.log("FAILED");
	console.error(testname + "FAILED");
   }
   callback(err, res);
}

