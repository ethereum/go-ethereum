function init(connection) {
  return function(message) {
    if (message === 'big') {
      return Promise.resolve(Buffer.alloc(30 * 1024 * 1024, 42));
    }
    return Promise.resolve('sent ' + message + ' to ' + connection);
  };
}
module.exports = init;
