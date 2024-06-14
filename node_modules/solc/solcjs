#!/usr/bin/env node

// hold on to any exception handlers that existed prior to this script running, we'll be adding them back at the end
var originalUncaughtExceptionListeners = process.listeners("uncaughtException");

var fs = require('fs-extra');
var path = require('path');
var solc = require('./index.js');
var smtchecker = require('./smtchecker.js');
var smtsolver = require('./smtsolver.js');
// FIXME: remove annoying exception catcher of Emscripten
//        see https://github.com/chriseth/browser-solidity/issues/167
process.removeAllListeners('uncaughtException');
var commander = require('commander');

var program = new commander.Command();
program.name('solcjs');
program.version(solc.version());
program
  .option('--version', 'Show version and exit.')
  .option('--optimize', 'Enable bytecode optimizer.')
  .option('--bin', 'Binary of the contracts in hex.')
  .option('--abi', 'ABI of the contracts.')
  .option('--standard-json', 'Turn on Standard JSON Input / Output mode.')
  .option('--base-path <path>', 'Automatically resolve all imports inside the given path.')
  .option('-o, --output-dir <output-directory>', 'Output directory for the contracts.');
program.parse(process.argv);

var files = program.args;
var destination = program.outputDir || '.'

function abort (msg) {
  console.error(msg || 'Error occured');
  process.exit(1);
}

function readFileCallback(path) {
  path = program.basePath + '/' + path;
  if (fs.existsSync(path)) {
    try {
      return { 'contents': fs.readFileSync(path).toString('utf8') }
    } catch (e) {
      return { error: 'Error reading ' + path + ': ' + e };
    }
  } else
    return { error: 'File not found at ' + path}
}
function stripBasePath(path) {
  if (program.basePath && path.startsWith(program.basePath))
    return path.slice(program.basePath.length);
  else
    return path;
}

var callbacks = undefined
if (program.basePath)
  callbacks = {'import': readFileCallback};

if (program.standardJson) {
  var input = fs.readFileSync(process.stdin.fd).toString('utf8');
  var output = solc.compile(input, callbacks);

  try {
    var inputJSON = smtchecker.handleSMTQueries(JSON.parse(input), JSON.parse(output), smtsolver.smtSolver);
    if (inputJSON) {
      output = solc.compile(JSON.stringify(inputJSON), callbacks);
    }
  }
  catch (e) {
    var addError = {
      component: "general",
      formattedMessage: e.toString(),
      message: e.toString(),
      type: "Warning"
    };

    var outputJSON = JSON.parse(output);
    if (!outputJSON.errors) {
      outputJSON.errors = []
    }
    outputJSON.errors.push(addError);
    output = JSON.stringify(outputJSON);
  }

  console.log(output);
  process.exit(0);
} else if (files.length === 0) {
  console.error('Must provide a file');
  process.exit(1);
}

if (!(program.bin || program.abi)) {
  abort('Invalid option selected, must specify either --bin or --abi');
}

var sources = {};

for (var i = 0; i < files.length; i++) {
  try {
    sources[stripBasePath(files[i])] = { content: fs.readFileSync(files[i]).toString() };
  } catch (e) {
    abort('Error reading ' + files[i] + ': ' + e);
  }
}

var output = JSON.parse(solc.compile(JSON.stringify({
  language: 'Solidity',
  settings: {
    optimizer: {
      enabled: program.optimize
    },
    outputSelection: {
      '*': {
        '*': [ 'abi', 'evm.bytecode' ]
      }
    }
  },
  sources: sources
}), callbacks));

let hasError = false;

if (!output) {
  abort('No output from compiler');
} else if (output['errors']) {
  for (var error in output['errors']) {
    var message = output['errors'][error]
    if (message.severity === 'warning') {
      console.log(message.formattedMessage)
    } else {
      console.error(message.formattedMessage)
      hasError = true
    }
  }
}

fs.ensureDirSync (destination);

function writeFile (file, content) {
  file = path.join(destination, file);
  fs.writeFile(file, content, function (err) {
    if (err) {
      console.error('Failed to write ' + file + ': ' + err);
    }
  });
}

for (var fileName in output.contracts) {
  for (var contractName in output.contracts[fileName]) {
    var contractFileName = fileName + ':' + contractName;
    contractFileName = contractFileName.replace(/[:./\\]/g, '_');

    if (program.bin) {
      writeFile(contractFileName + '.bin', output.contracts[fileName][contractName].evm.bytecode.object);
    }

    if (program.abi) {
      writeFile(contractFileName + '.abi', JSON.stringify(output.contracts[fileName][contractName].abi));
    }
  }
}

// Put back original exception handlers.
originalUncaughtExceptionListeners.forEach(function (listener) {
  process.addListener('uncaughtException', listener);
});

if (hasError) {
  process.exit(1);
}