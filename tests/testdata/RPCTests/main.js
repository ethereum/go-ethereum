//requires npm
//requires installed node v6
//requires ethereum eth path on input

var async = require("async");
var utils = require('./modules/utils.js');
var testutils = require('./modules/testutils.js');
var ethconsole = require('./modules/ethconsole.js');

var ethpath = process.argv[2];
var testdir = __dirname + "/dynamic";
var workdir = __dirname;

var dynamic = {};

function cb(){}
function main()
{
if (!ethpath || !utils.fileExists(ethpath))
{
	utils.cLog("Executable '" + ethpath + "' not found!");
	utils.cLog("Please, set eth path. Usage: node main.js <ethpath>");
	return;
}

testutils.readTestsInFolder(workdir + "/scripts/tests");
async.series([
function(cb) {
	utils.setDebug(false);
	ethconsole.startNode(ethpath, testdir + "/ethnode1", testdir + "/genesis.json", 30305, cb);
},
function(cb) {
	prepareDynamicVars(cb);
},
function(cb) {
	ethconsole.stopNode(testdir + "/ethnode1", cb);
},
function(cb) {
	ethconsole.startNode(ethpath, testdir + "/ethnode1", testdir + "/genesis.json", 30305, cb);
	dynamic["node1_port"] = "30305";
},
function(cb) {
	ethconsole.startNode(ethpath, testdir + "/ethnode2", testdir + "/genesis.json", 30306, cb);
	dynamic["node2_port"] = "30306";
},
function(cb) {
	runAllTests(cb);	
},
function(cb) {
	ethconsole.stopNode(testdir + "/ethnode1", cb);
	ethconsole.stopNode(testdir + "/ethnode2", cb);
}
], function() { 
	utils.rmdir(testdir); }
)
}//main



function prepareDynamicVars(finished)
{
  async.series([
	function(cb) {		
		ethconsole.runScriptOnNode(testdir + "/ethnode1", workdir + "/scripts/testNewAccount.js", {}, cb);
	},
	function(cb) {
		dynamic["account"] = ethconsole.getLastResponse();
		utils.mkdir(testdir);
		testutils.generateCustomGenesis(testdir + '/genesis.json', workdir + "/scripts/genesis.json", dynamic["account"], cb);
	},
	function(cb) {
		ethconsole.runScriptOnNode(testdir + "/ethnode1", workdir + "/scripts/getNodeInfo.js", {}, cb);
	},
	function(cb) {
		dynamic["node1_ID"] = ethconsole.getLastResponse().id;
		cb();
	}
  ], function() { finished(); })
}

function runAllTests(finished)
{
	var currentTest = -1;
	var updateDynamic = function(){};

	function nextTest()
	{
	   currentTest++;
	   if (currentTest == testutils.getTestCount())
		finished();
	   else
	   {
		var testObject = testutils.getTestNumber(currentTest);
		var nodepath;
		if (testObject.node == '01')
			nodepath = testdir + "/ethnode1";
		if (testObject.node == '02')
			nodepath = testdir + "/ethnode2";

		ethconsole.runScriptOnNode(nodepath, testObject.file, dynamic, updateDynamic);
	   }
	}

	updateDynamic = function updateDynamic()
	{
		async.series([
			function(cb) {
				ethconsole.runScriptOnNode(testdir + "/ethnode1", workdir + "/scripts/getLastBlock.js", {}, cb);
			},
			function(cb) {
				dynamic["node1_lastblock"] = ethconsole.getLastResponse();
				cb();
			},
			function(cb) {
				ethconsole.runScriptOnNode(testdir + "/ethnode2", workdir +  "/scripts/getLastBlock.js", {}, cb);
			},
			function(cb) {
				dynamic["node2_lastblock"] = ethconsole.getLastResponse();
				cb();
			}
	    	], function() { nextTest(); });
	}
	nextTest();
}

main();

