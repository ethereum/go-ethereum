module.exports = {
  basicTests: require('require-all')(__dirname + '/BasicTests/'),
  trieTests: require('require-all')(__dirname + '/TrieTests/'),
  stateTests: require('require-all')(__dirname + '/StateTests/'),
  vmTests: require('require-all')(__dirname + '/VMTests')
};
