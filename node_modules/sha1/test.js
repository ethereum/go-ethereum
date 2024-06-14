var sha1 = require('./sha1.js');
var assert = require('assert');

describe('sha1', function () {
  it ('should return the expected SHA-1 hash for "message"', function () {
    assert.equal('6f9b9af3cd6e8b8a73c2cdced37fe9f59226e27d', sha1('message'));
  });

  it('should not return the same hash for random numbers twice', function () {
    var msg1 = Math.floor((Math.random() * 100000) + 1) + (new Date).getTime();
    var msg2 = Math.floor((Math.random() * 100000) + 1) + (new Date).getTime();

    if (msg1 !== msg2)
      assert.notEqual(sha1(msg1), sha1(msg2));
    else
      assert.equal(sha1(msg1), sha1(msg1));
  });

  it('should node.js Buffer', function() {

    var buffer = new Buffer('hello, sha1', 'utf8');

    assert.equal(sha1(buffer), sha1('hello, sha1'));
  })
});
