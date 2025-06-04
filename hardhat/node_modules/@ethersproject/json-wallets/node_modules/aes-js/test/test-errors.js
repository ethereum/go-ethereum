'use strict';

var nodeunit = require('nodeunit');

var aes = require('../index');

function newBuffer(length) {
    var buffer = new Buffer(length);
    buffer.fill(42);
    return buffer;
}

// Invalid key sizes to try
var keySizes = [0, 1, 2, 7, 8, 9, 15, 17, 23, 25, 31, 33, 100];

module.exports = {
    "test-errors-key-size": function(test) {
        for (var i = 0; i < keySizes.length; i++) {
            test.throws(function() {
                var moo = new aes.ModeOfOperation.ecb(newBuffer(keySizes[i]));
            }, function(error) {
                return (error.message === 'invalid key size (must be 16, 24 or 32 bytes)');
            },
            'invalid key size failed to throw an error');
        }

        test.done();
    },

    "test-errors-iv-size": function(test) {
        var ivSizes = [0, 15, 17, 100];
        for (var i = 0; i < 3; i++) {
            var keySize = newBuffer(16 + i * 8);

            for (var j = 0; j < ivSizes.length; j++) {
                test.throws(function() {
                    var moo = new aes.ModeOfOperation.cbc(newBuffer(keySize), newBuffer(ivSizes[j]));
                }, function(error) {
                    return (error.message === 'invalid initialation vector size (must be 16 bytes)');
                },
                'invalid iv size for cbc failed to throw an error');

                test.throws(function() {
                    var moo = new aes.ModeOfOperation.ofb(newBuffer(keySize), newBuffer(ivSizes[j]));
                }, function(error) {
                    return (error.message === 'invalid initialation vector size (must be 16 bytes)');
                },
                'invalid iv size for ofb failed to throw an error');
            }
        }

        test.done();
    },

    "test-errors-segment-size": function(test) {
        var key = newBuffer(16);
        var iv = newBuffer(16);
        for (var i = 1; i < 17; i++) {
            for (var j = 1; j < 17; j++) {
                if ((j % i) === 0) { continue; }

                var moo = new aes.ModeOfOperation.cfb(key, iv, i);

                test.throws(function() {
                    moo.encrypt(newBuffer(j));
                }, function(error) {
                    return (error.message === 'invalid plaintext size (must be segmentSize bytes)');
                },
                'invalid plaintext (invalid segment size) failed to throw an error ' + i + ' ' + j);
            }
        }

        test.done();
    },

    "test-errors-text-size": function(test) {
        var textSizes = [1, 2, 15, 17, 31, 33];

        for (var i = 0; i < 3; i++) {
            var key = newBuffer(16 + i * 8);
            for (var j = 0; j < textSizes.length; j++) {
                for (var k = 0; k < 2; k++) {
                    var text = newBuffer(textSizes[j]);
                    if (k === 0) {
                        var moo = new aes.ModeOfOperation.ecb(key);
                    } else {
                        var moo = new aes.ModeOfOperation.cbc(key, newBuffer(16));
                    }

                    test.throws(function() {
                        moo.encrypt(text);
                    }, function(error) {
                        return (error.message === 'invalid plaintext size (must be multiple of 16 bytes)');
                    },
                    'invalid text size failed to throw an error');
                }
            }
        }

        for (var i = 0; i < 3; i++) {
            var key = newBuffer(16 + i * 8);
            for (var j = 0; j < textSizes.length; j++) {
                for (var k = 0; k < 2; k++) {
                    var text = newBuffer(textSizes[j]);
                    if (k === 0) {
                        var moo = new aes.ModeOfOperation.ecb(key);
                    } else {
                        var moo = new aes.ModeOfOperation.cbc(key, newBuffer(16));
                    }

                    test.throws(function() {
                        moo.decrypt(text);
                    }, function(error) {
                        return (error.message === 'invalid ciphertext size (must be multiple of 16 bytes)');
                    },
                    'invalid text size failed to throw an error');
                }
            }
        }

        test.done();
    },

    "test-errors-counter": function(test) {
        var textSizes = [0, 1, 2, 15, 17];
        for (var i = 0; i < textSizes.length; i++) {
            test.throws(function() {
                var counter = new aes.Counter(newBuffer(textSizes[i]));
            }, function(error) {
                return (error.message === 'invalid counter bytes size (must be 16 bytes)');
            },
            'invalid counter size (bytes.length != 16) failed to throw an error');

            var counter = new aes.Counter();
            test.throws(function() {
                counter.setBytes(newBuffer(textSizes[i]));
            }, function(error) {
                return (error.message === 'invalid counter bytes size (must be 16 bytes)');
            },
            'invalid counter setBytes (bytes.length != 16) failed to throw an error');

            var counter = new aes.Counter();
            test.throws(function() {
                counter.setValue(newBuffer(textSizes[i]));
            }, function(error) {
                return (error.message === 'invalid counter value (must be an integer)');
            },
            'invalid counter setValue (Array) failed to throw an error');
        }

        test.throws(function() {
            var counter = new aes.Counter(1.5);
        }, function(error) {
            return (error.message === 'invalid counter value (must be an integer)');
        },
        'invalid counter value (non-integer number) failed to throw an error');

        var counter = new aes.Counter();
        test.throws(function() {
            counter.setValue(1.5);
        }, function(error) {
            return (error.message === 'invalid counter value (must be an integer)');
        },
        'invalid counter setValue (non-integer number) failed to throw an error');

        var badThings = [0, 1.5, 1];
        for (var i = 0; i < badThings.length; i++) {
            var counter = new aes.Counter();
            test.throws(function() {
                counter.setBytes(badThings[i]);
            }, function(error) {
                return (error.message === 'unsupported array-like object');
            },
            'invalid counter setBytes (numbers) failed to throw an error');
        }

        var badThings = [1.5, 'foobar', {}];
        for (var i = 0; i < badThings.length; i++) {
            var counter = new aes.Counter();
            test.throws(function() {
                counter.setBytes(badThings[i]);
            }, function(error) {
                return (error.message === 'unsupported array-like object');
            },
            'invalid counter setBytes (' + badThings[i] + ') failed to throw an error');

            var counter = new aes.Counter();
            test.throws(function() {
                counter.setValue(badThings[i]);
            }, function(error) {
                return (error.message === 'invalid counter value (must be an integer)');
            },
            'invalid counter setValue (' + badThings[i] + ') failed to throw an error');
        }

        test.done();
    },
};
