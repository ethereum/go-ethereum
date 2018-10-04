process.stdout.write("TEST_newAccount ");
var onResult = {};
web3.personal.newAccount("123", function(err,res){ onResult(err,res); })

onResult = function (err,res) 
{
   if (res.length == 42)
	console.log("OK");
   else 
	console.log("FAILED");
   callback(err, res);
}
