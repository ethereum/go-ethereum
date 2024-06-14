'use strict';

const spawn = require('child_process').spawn;
const spawnSync = require('child_process').spawnSync;
const thenRequest = require('then-request');
const syncRequest = require('../');

const server = spawn(process.execPath, [require.resolve('./benchmark-server.js')]);

setTimeout(() => {
  let asyncDuration, syncDuration;
  let ready = Promise.resolve(null);
  const startAsync = Date.now();
  for (let i = 0; i < 1000; i++) {
    ready = ready.then(function () {
      return thenRequest('get', 'http://localhost:3045');
    });
  }
  ready.then(function () {
    const endAsync = Date.now();
    asyncDuration = endAsync - startAsync;
    console.log('1000 async requests in: ' + asyncDuration);
    const startSync = Date.now();
    for (let i = 0; i < 500; i++) {
      syncRequest('get', 'http://localhost:3045');
    }
    const endSync = Date.now();
    syncDuration = endSync - startSync;
    console.log('1000 sync requests in: ' + syncDuration);
  }).then(() => {
    server.kill();
    if (syncDuration > (asyncDuration * 10)) {
      console.error('This is more than 10 times slower than using async requests, that is not good enough.');
      process.exit(1);
    }
    process.exit(0);
  }, function (err) {
    console.error(err.stack);
    process.exit(1);
  });
  ready = null;
}, 1000);
