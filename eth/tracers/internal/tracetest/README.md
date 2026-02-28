# Filling test cases

To fill test cases for the built-in tracers, the `makeTest.js` script can be used. Given a transaction on a dev/test network, `makeTest.js` will fetch its prestate and then traces with the given configuration.
In the Geth console do:

```terminal
let tx = '0x...'
loadScript('makeTest.js')
makeTest(tx, { tracer: 'callTracer' })
```