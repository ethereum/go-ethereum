'use strict';

const getPort = require('get-port');

getPort()
  .then(port => process.stdout.write('' + port))
  .catch(err =>
    setTimeout(() => {
      throw err;
    }, 0)
  );
