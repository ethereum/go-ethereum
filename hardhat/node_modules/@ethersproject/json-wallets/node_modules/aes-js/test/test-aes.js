var nodeunit = require('nodeunit');

var aes = require('../index');

function bufferEquals(a, b) {
    if (a.length != b.length) { return false; }
    for (var i = 0; i < a.length; i++) {
        if (a[i] != b[i]) { return false; }
    }
    return true;
}

function makeTest(options) {

    var modeOfOperation = options.modeOfOperation;
    var mo = aes.ModeOfOperation[modeOfOperation];

    var plaintext = [];
    for (var i = 0; i < options.plaintext.length; i++) {
        plaintext.push(new Buffer(options.plaintext[i]));
    }

    var key = new Buffer(options.key);

    var iv = null;
    if (options.iv) { iv = new Buffer(options.iv); }

    var segmentSize = 0;
    if (options.segmentSize) { segmentSize = options.segmentSize; }

    var ciphertext = [];
    for (var i = 0; i < options.encrypted.length; i++) {
        ciphertext.push(new Buffer(options.encrypted[i]));
    }


    return function (test) {
        var func;
        switch (modeOfOperation) {
            case 'ecb':
                func = function() { return new mo(key); }
                break;
            case 'cfb':
                func = function() { return new mo(key, iv, segmentSize); }
                break;
            case 'ofb':
            case 'cbc':
                func = function() { return new mo(key, iv); }
                break;
            case 'ctr':
                func = function() { return new mo(key, new aes.Counter(0)); }
                break;
            default:
                throw new Error('unknwon mode of operation')
        }

        var encrypter = func(), decrypter = func();
        var totalDiffers = 0;
        for (var i = 0; i < plaintext.length; i++) {
            var ciphertext2 = encrypter.encrypt(plaintext[i]);
            test.ok(bufferEquals(ciphertext2, ciphertext[i]), "encrypt failed to match test vector");

            var plaintext2 = decrypter.decrypt(ciphertext2);
            test.ok(bufferEquals(plaintext2, plaintext[i]), "decrypt failed to match original text");
        }

        test.done();
    };
};


var testVectors = require('./test-vectors.json');

var Tests = {};
var counts = {}
for (var i = 0; i < testVectors.length; i++) {
    var test = testVectors[i];
    name = test.modeOfOperation + '-' + test.key.length;
    counts[name] = (counts[name] || 0) + 1;
    Tests['test-' + name + '-' + counts[name]] = makeTest(test);
}

module.exports = Tests;
