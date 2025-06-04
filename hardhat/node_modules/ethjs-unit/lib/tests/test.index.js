'use strict';

var units = require('../index.js'); // eslint-disable-line
var BigNumber = require('bn.js'); // eslint-disable-line
var ActualBigNumber = require('bignumber.js');
var Web3 = require('web3'); // eslint-disable-line
var web3 = new Web3(); // eslint-disable-line
var assert = require('chai').assert; // eslint-disable-line
var totalTypes = Object.keys(units.unitMap).length;

function testRandomValueAgainstWeb3ToWei(negative) {
  var stringTestValue = '' + (negative ? '-' : '') + String(Math.floor(Math.random() * 100000000000000000 + 1));
  var randomunitsType = Object.keys(units.unitMap)[Math.floor(Math.random() * (totalTypes - 1) + 1)];
  var unitsValue = units.toWei(stringTestValue, randomunitsType);
  var web3Value = new BigNumber(web3.toWei(stringTestValue, randomunitsType));

  // it(`toWei should work like web3 val ${unitsValue.toString(10)} should equal ${web3Value.toString(10)}`, () => {
  assert.deepEqual(unitsValue, web3Value);
  // });
}

function testRandomValueAgainstWeb3FromWei(negative) {
  var stringTestValue = '' + (negative ? '-' : '') + String(Math.floor(Math.random() * 100000000000000000 + 1));
  var randomunitsType = Object.keys(units.unitMap)[Math.floor(Math.random() * (totalTypes - 1) + 1)];
  var unitsValue = units.fromWei(stringTestValue, randomunitsType);
  var web3Value = web3.fromWei(stringTestValue, randomunitsType);

  // it(`fromWei should work like web3 rounded val ${unitsValue.substr(0, web3Value.length - 1)} should equal ${web3Value.substr(0, web3Value.length - 1)} for unit type ${randomunitsType}`, () => {
  assert.deepEqual(unitsValue.substr(0, web3Value.length - 1), web3Value.substr(0, web3Value.length - 1));
  // });
}

describe('getValueOfUnit', function () {
  it('should throw when undefined or not string', function () {
    function invalidFromWei() {
      units.fromWei(1000000000000000000, 'something');
    }
    assert.throws(invalidFromWei, Error);
  });
});

describe('toWei', function () {
  it('should handle edge cases', function () {
    assert.equal(units.toWei(0, 'wei').toString(10), '0');
    assert.equal(units.toWei('0.0', 'wei').toString(10), '0');
    assert.equal(units.toWei('.3', 'ether').toString(10), '300000000000000000');
    assert.throws(function () {
      return units.toWei('.', 'wei');
    }, Error);
    assert.throws(function () {
      return units.toWei('1.243842387924387924897423897423', 'ether');
    }, Error);
    assert.throws(function () {
      return units.toWei('8723.98234.98234', 'ether');
    }, Error);
  });

  it('should return the correct value', function () {
    assert.equal(units.toWei(1, 'wei').toString(10), '1');
    assert.equal(units.toWei(1, 'kwei').toString(10), '1000');
    assert.equal(units.toWei(1, 'Kwei').toString(10), '1000');
    assert.equal(units.toWei(1, 'babbage').toString(10), '1000');
    assert.equal(units.toWei(1, 'mwei').toString(10), '1000000');
    assert.equal(units.toWei(1, 'Mwei').toString(10), '1000000');
    assert.equal(units.toWei(1, 'lovelace').toString(10), '1000000');
    assert.equal(units.toWei(1, 'gwei').toString(10), '1000000000');
    assert.equal(units.toWei(1, 'Gwei').toString(10), '1000000000');
    assert.equal(units.toWei(1, 'shannon').toString(10), '1000000000');
    assert.equal(units.toWei(1, 'szabo').toString(10), '1000000000000');
    assert.equal(units.toWei(1, 'finney').toString(10), '1000000000000000');
    assert.equal(units.toWei(1, 'ether').toString(10), '1000000000000000000');
    assert.equal(units.toWei(1, 'kether').toString(10), '1000000000000000000000');
    assert.equal(units.toWei(1, 'grand').toString(10), '1000000000000000000000');
    assert.equal(units.toWei(1, 'mether').toString(10), '1000000000000000000000000');
    assert.equal(units.toWei(1, 'gether').toString(10), '1000000000000000000000000000');
    assert.equal(units.toWei(1, 'tether').toString(10), '1000000000000000000000000000000');

    assert.equal(units.toWei(1, 'kwei').toString(10), units.toWei(1, 'femtoether').toString(10));
    assert.equal(units.toWei(1, 'szabo').toString(10), units.toWei(1, 'microether').toString(10));
    assert.equal(units.toWei(1, 'finney').toString(10), units.toWei(1, 'milliether').toString(10));
    assert.equal(units.toWei(1, 'milli').toString(10), units.toWei(1, 'milliether').toString(10));
    assert.equal(units.toWei(1, 'milli').toString(10), units.toWei(1000, 'micro').toString(10));

    assert.throws(function () {
      units.toWei(1, 'wei1');
    }, Error);
  });
});

describe('numberToString', function () {
  it('should handle edge cases', function () {
    // assert.throws(() => units.numberToString(null), Error);
    assert.throws(function () {
      return units.numberToString(undefined);
    }, Error);
    // assert.throws(() => units.numberToString(NaN), Error);
    assert.throws(function () {
      return units.numberToString({});
    }, Error);
    assert.throws(function () {
      return units.numberToString([]);
    }, Error);
    assert.throws(function () {
      return units.numberToString('-1sdffsdsdf');
    }, Error);
    assert.throws(function () {
      return units.numberToString('-0..-...9');
    }, Error);
    assert.throws(function () {
      return units.numberToString('fds');
    }, Error);
    assert.throws(function () {
      return units.numberToString('');
    }, Error);
    assert.throws(function () {
      return units.numberToString('#');
    }, Error);
    assert.equal(units.numberToString(55), '55');
    assert.equal(units.numberToString(1), '1');
    assert.equal(units.numberToString(-1), '-1');
    assert.equal(units.numberToString(0), '0');
    assert.equal(units.numberToString(-0), '0');
    assert.equal(units.numberToString(new ActualBigNumber(10.1)), '10.1');
    assert.equal(units.numberToString(new ActualBigNumber(10000)), '10000');
    assert.equal(units.numberToString(new BigNumber(10000)), '10000');
    assert.equal(units.numberToString(new BigNumber('-1')), '-1');
    assert.equal(units.numberToString(new BigNumber('1')), '1');
    assert.equal(units.numberToString(new BigNumber(0)), '0');
  });
});

describe('fromWei', function () {
  it('should handle options', function () {
    assert.equal(units.fromWei(10000000, 'wei', { commify: true }), '10,000,000');
  });

  it('should return the correct value', function () {
    assert.equal(units.fromWei(1000000000000000000, 'wei'), '1000000000000000000');
    assert.equal(units.fromWei(1000000000000000000, 'kwei'), '1000000000000000');
    assert.equal(units.fromWei(1000000000000000000, 'mwei'), '1000000000000');
    assert.equal(units.fromWei(1000000000000000000, 'gwei'), '1000000000');
    assert.equal(units.fromWei(1000000000000000000, 'szabo'), '1000000');
    assert.equal(units.fromWei(1000000000000000000, 'finney'), '1000');
    assert.equal(units.fromWei(1000000000000000000, 'ether'), '1');
    assert.equal(units.fromWei(1000000000000000000, 'kether'), '0.001');
    assert.equal(units.fromWei(1000000000000000000, 'grand'), '0.001');
    assert.equal(units.fromWei(1000000000000000000, 'mether'), '0.000001');
    assert.equal(units.fromWei(1000000000000000000, 'gether'), '0.000000001');
    assert.equal(units.fromWei(1000000000000000000, 'tether'), '0.000000000001');
  });
});

describe('units', function () {
  describe('normal functionality', function () {
    it('should be the same as web3', function () {
      for (var i = 0; i < 15000; i++) {
        // eslint-disable-line
        testRandomValueAgainstWeb3ToWei(false);
        testRandomValueAgainstWeb3ToWei(true);
        testRandomValueAgainstWeb3FromWei(false);
        testRandomValueAgainstWeb3FromWei(true);
      }
    });
  });
});