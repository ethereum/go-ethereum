var test = require('tape')
var ordinal = require('../')
var fixtures = require('./fixtures.json')

test('returns ordinal numbers', function (t) {
  fixtures.forEach(function (x) {
    t.equal(ordinal(x.i), x.ordinal, x.i + ' === ' + x.ordinal)
  })

  t.end()
})

test('returns negative ordinal numbers', function (t) {
  fixtures.forEach(function (x) {
    x.i === 0
      ? t.equal(ordinal(x.i), x.ordinal, x.i + ' === ' + x.ordinal)
      : t.equal(ordinal(-x.i), '-' + x.ordinal, x.i + ' === ' + x.ordinal)
  })

  t.end()
})

test('returns non-finite numbers', function (t) {
  [Infinity, -Infinity, NaN, -NaN].forEach(function (i) {
    t.equal(ordinal(i) + '', i + '', i + ' === ' + i)
  })

  t.end()
})

test('throws for non-number types', function (t) {
  t.throws(function () {
    ordinal('foo')
  }, /Expected Number, got string foo/)
  t.end()
})
