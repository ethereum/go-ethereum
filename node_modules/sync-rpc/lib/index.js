'use strict';

const path = require('path');
const spawn = require('child_process').spawn;
const spawnSync = require('child_process').spawnSync;
const JSON = require('./json-buffer');

const host = '127.0.0.1';
function nodeNetCatSrc(port, input) {
  return (
    "var c=require('net').connect(" +
    port +
    ",'127.0.0.1',()=>{c.pipe(process.stdout);c.end(" +
    JSON.stringify(input)
      .replace(/\u2028/g, '\\u2028')
      .replace(/\u2029/g, '\\u2029') +
    ')})'
  );
}

const FUNCTION_PRIORITY = [nativeNC, nodeNC];

let started = false;
const configuration = {port: null, fastestFunction: null};
function start() {
  if (!spawnSync) {
    throw new Error(
      'Sync-request requires node version 0.12 or later.  If you need to use it with an older version of node\n' +
        'you can `npm install sync-request@2.2.0`, which was the last version to support older versions of node.'
    );
  }
  const port = findPort();
  const p = spawn(process.execPath, [require.resolve('./worker'), port], {
    stdio: 'inherit',
    windowsHide: true,
  });
  p.unref();
  process.on('exit', () => {
    p.kill();
  });
  waitForAlive(port);
  const fastestFunction = getFastestFunction(port);
  configuration.port = port;
  configuration.fastestFunction = fastestFunction;
  started = true;
}

function findPort() {
  const findPortResult = spawnSync(
    process.execPath,
    [require.resolve('./find-port')],
    {
      windowsHide: true,
    }
  );
  if (findPortResult.error) {
    if (typeof findPortResult.error === 'string') {
      throw new Error(findPortResult.error);
    }
    throw findPortResult.error;
  }
  if (findPortResult.status !== 0) {
    throw new Error(
      findPortResult.stderr.toString() ||
        'find port exited with code ' + findPortResult.status
    );
  }
  const portString = findPortResult.stdout.toString('utf8').trim();
  if (!/^[0-9]+$/.test(portString)) {
    throw new Error('Invalid port number string returned: ' + portString);
  }
  return +portString;
}

function waitForAlive(port) {
  let response = null;
  let err = null;
  let timeout = Date.now() + 10000;
  while (response !== 'pong' && Date.now() < timeout) {
    const result = nodeNC(port, 'ping\r\n');
    response = result.stdout && result.stdout.toString();
    err = result.stderr && result.stderr.toString();
  }
  if (response !== 'pong') {
    throw new Error(
      'Timed out waiting for sync-rpc server to start (it should respond with "pong" when sent "ping"):\n\n' +
        err +
        '\n' +
        response
    );
  }
}

function nativeNC(port, input) {
  return spawnSync('nc', [host, port], {
    input: input,
    windowsHide: true,
    maxBuffer: Infinity,
  });
}

function nodeNC(port, input) {
  const src = nodeNetCatSrc(port, input);
  if (src.length < 1000) {
    return spawnSync(process.execPath, ['-e', src], {
      windowsHide: true,
      maxBuffer: Infinity,
    });
  } else {
    return spawnSync(process.execPath, [], {
      input: src,
      windowsHide: true,
      maxBuffer: Infinity,
    });
  }
}

function test(fn, port) {
  const result = fn(port, 'ping\r\n');
  const response = result.stdout && result.stdout.toString();
  return response === 'pong';
}

function getFastestFunction(port) {
  for (let i = 0; i < FUNCTION_PRIORITY.length; i++) {
    if (test(FUNCTION_PRIORITY[i], port)) {
      return FUNCTION_PRIORITY[i];
    }
  }
}

function sendMessage(input) {
  if (!started) start();
  const res = configuration.fastestFunction(
    configuration.port,
    JSON.stringify(input) + '\r\n'
  );
  try {
    return JSON.parse(res.stdout.toString('utf8'));
  } catch (ex) {
    if (res.error) {
      if (typeof res.error === 'string') res.error = new Error(res.error);
      throw res.error;
    }
    if (res.status !== 0) {
      throw new Error(
        configuration.fastestFunction.name +
          ' failed:\n' +
          (res.stdout && res.stdout.toString()) +
          '\n' +
          (res.stderr && res.stderr.toString())
      );
    }
    throw new Error(
      configuration.fastestFunction.name +
        ' failed:\n' +
        (res.stdout && res.stdout).toString() +
        '\n' +
        (res.stderr && res.stderr).toString()
    );
  }
}
function extractValue(msg) {
  if (!msg.s) {
    const error = new Error(msg.v.message);
    error.code = msg.v.code;
    throw error;
  }
  return msg.v;
}

function createClient(filename, args) {
  const id = extractValue(sendMessage({t: 1, f: filename, a: args}));
  return function(args) {
    return extractValue(sendMessage({t: 0, i: id, a: args}));
  };
}
createClient.FUNCTION_PRIORITY = FUNCTION_PRIORITY;
createClient.configuration = configuration;

module.exports = createClient;
