'use strict';

if (process.env.SYNC_REQUEST_LEGACY) {
  // break PATH so running `nc` will fail.
  process.env.PATH = '';
}

var request = require('../');
var FormData = request.FormData;

const fork = require('child_process').fork;
var server = fork(__dirname + '/fake-server', {stdio: 'pipe'});

test('start server', () => {
  return new Promise(resolve => {
    server.on('message', m => {
      if (m === 'started') {
        resolve();
      }
    });
    server.send('start');
  });
});

test('GET request', () => {
  var res = request('GET', 'http://localhost:3030/internal-test', {
    timeout: 2000,
  });
  expect(res.statusCode).toBe(200);
  expect(res.getBody('utf8')).toMatchSnapshot();
});

test('POST request', () => {
  var res = request('POST', 'http://localhost:3030/internal-test', {
    timeout: 2000,
    body: '<body/>',
  });
  expect(res.statusCode).toBe(200);
  expect(res.getBody('utf8')).toMatchSnapshot();
});

test('PUT request', () => {
  var res = request('PUT', 'http://localhost:3030/internal-test', {
    timeout: 2000,
    body: '<body/>',
  });
  expect(res.statusCode).toBe(200);
  expect(res.getBody('utf8')).toMatchSnapshot();
});

test('DELETE request', () => {
  var res = request('DELETE', 'http://localhost:3030/internal-test', {
    timeout: 2000,
  });
  expect(res.statusCode).toBe(200);
  expect(res.getBody('utf8')).toMatchSnapshot();
});

test('stop server', () => {
  server.send('stop');
});
