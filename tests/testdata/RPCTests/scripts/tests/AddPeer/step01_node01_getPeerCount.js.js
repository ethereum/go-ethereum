var testname = "TEST_getPeerCountOnNode1 ";
process.stdout.write(testname);

var onResult = {};
web3.net.getPeerCount(function(err, res){ onResult(err, res); })
onResult = function (err,res) 
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

