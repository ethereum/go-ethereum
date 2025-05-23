const flatten = require('./')
const TestRunner = require('test-runner')
const a = require('assert')

const runner = new TestRunner()

runner.test('flatten', function () {
  const numbers = [ 1, 2, [ 3, 4 ], 5 ]
  const result = numbers.reduce(flatten, [])
  a.deepStrictEqual(result, [ 1, 2, 3, 4, 5 ])
})
