var tape = require('tape');
const ethash = require('../ethash.js');

tape('seed hash', function(t) {

  t.test('seed should match TRUTH', function(st) {
    const seed = '290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563';
    const blockNum = 30000;

    var r = new Buffer(ethash.calcSeed(blockNum));
    st.equal(r.toString('hex'), seed);

    st.end();
  });

  t.test('seed should match TRUTH2', function(st) {
    const seed = '510e4e770828ddbf7f7b00ab00a9f6adaf81c0dc9cc85f1f8249c256942d61d9';
    const blockNum = 60000;

    var r = new Buffer(ethash.calcSeed(blockNum));
    st.equal(r.toString('hex'), seed);

    st.end();
  });

  t.test('seed should match TRUTH3', function(st) {
    const seed = '510e4e770828ddbf7f7b00ab00a9f6adaf81c0dc9cc85f1f8249c256942d61d9';
    const blockNum = 60700;

    var r = new Buffer(ethash.calcSeed(blockNum));
    st.equal(r.toString('hex'), seed);

    st.end();
  });

  t.test('randomized tests', function(st) {
    for (var i = 0; i < 100; i++) {
      var x = Math.floor(ethash.params.EPOCH_LENGTH * 2048 * Math.random());
      st.equal(ethash.calcSeed(x).toString('hex'), ethash.calcSeed(Math.floor(x / ethash.params.EPOCH_LENGTH) * ethash.params.EPOCH_LENGTH ).toString('hex'));
    }
    st.end();
  });
  // '510e4e770828ddbf7f7b00ab00a9f6adaf81c0dc9cc85f1f8249c256942d61d9'
  // [7:13:32 PM] Matthew Wampler-Doty: >>> x = randint(0,700000)
  // 
  // >>> pyethash.get_seedhash(x).encode('hex') == pyethash.get_seedhash((x // pyethash.EPOCH_LENGTH) * pyethash.EPOCH_LENGTH).encode('hex')

});
