const fs = require('fs');
var lastResponse;
var nodes = {};

module.exports = {

startNode: function startNode (nodeExec, dataDir, genesisPath, listeningPort, finished) 
{
  var utils = require('./utils.js');
  var spawn = require('child_process').spawn
  var options = [
    '--private', 'privatechain',
    '-d', dataDir,
    '--config', genesisPath,
    '--ipcpath', dataDir + '/geth.ipc',
    '--ipc',
    '--listen', listeningPort,
    '--test',
    '-a', '0x1122334455667788991011121314151617181920'
  ]
  utils.cLog('starting node')
  utils.cLog(nodeExec + ' ' + options.join(' '))
  var node = spawn(nodeExec, options)
  node.stdout.on('data', (data) => {
    utils.cLog(`stdout: ${data}`)
  })
  node.stderr.on('data', (data) => {
    utils.cLog(`stderr: ${data}`)
  })
  node.on('close', (code) => {
    utils.cLog(`child process exited with code ${code}`)
  })

  nodes[dataDir] = node;
  utils.sleep(14000).then(() => {
	utils.cLog("Node Started");
	finished();
  });
},

stopNode: function stopNode(dataDir, finished)
{
  nodes[dataDir].kill();
  var utils = require('./utils.js');
  utils.sleep(1000).then(() => {
	finished();
  });
},


runScriptOnNode: function runScriptOnNode(dataDir, jsScript, args, finished)
{
	var utils = require('./utils.js');
	var ipcPath = dataDir + '/geth.ipc';

	var Web3 = require('web3');
	var web3admin = require('./web3Admin.js');
	var net = require('net');

	utils.cLog("Connecting to node at " + ipcPath);
	var web3 = new Web3(new Web3.providers.IpcProvider(ipcPath, net));
	web3admin.extend(web3);
	global.web3 = web3;

	var onScriptCallback = function (err, data)
	{
		utils.cLog(data);
		lastResponse = data;
		finished();
	}
	global.callback = onScriptCallback;
	global.args = args;

	var vm = require('vm');
	utils.cLog("Executing " + jsScript + " ...");
	fs.readFile(jsScript, 'utf8', function (err, data)
	{
		if (err)
		{
			utils.cLog(err);
			finished();
		}
		else
		{
			var script = new vm.Script(data);
			script.runInThisContext();
		}
	});
},


getLastResponse: function getLastResponse() 
{
	return lastResponse;
}

}//exports

