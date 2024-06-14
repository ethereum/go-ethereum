'use strict';
var express = require('express'),
  bodyParser = require('body-parser'),
  morgan = require('morgan'),
  PORT = 3030;

var app = express();

// parse application/x-www-form-urlencoded
app.use(bodyParser.urlencoded({extended: false}));

// parse application/json
app.use(bodyParser.json());

// configure log
app.use(morgan('dev'));

var started = false;
exports.isStarted = function() {
  return started;
};

var server;
process.on('message', function(m) {
  if (m === 'start') {
    server = app.listen(PORT, function() {
      started = true;
      return process.send('started');
    });
  } else {
    server.close(function() {
      started = false;
      return process.send('closed') && process.exit(0);
    });
  }
});

['get', 'post', 'put', 'delete'].forEach(function(method) {
  app.route('/internal-test')[method](function(req, res) {
    res.send('ok');
  });
});
