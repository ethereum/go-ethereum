# Filling test cases

To fill test cases for the built-in tracers, the `makeTest.js` script can be used. Given a transaction on a dev/test network, `makeTest.js` will fetch its prestate and then traces with the given configuration.
In the Geth console do:

```terminal
let tx = '0x...'
loadScript('makeTest.js')
makeTest(tx, { tracer: 'callTracer' })
```

## Updating the existing call tracer test cases
In case a change is introduced to the output/format of the call tracer, you may use the following invocation to update the expected output with the current output:
```bash
UPDATE_TESTS=1 go test ./eth/tracers/internal/tracetest
```

This will blindly "accept" the current trace response as the correct one, so manual inspection of the updated outputs is advised to ensure they are as expected.
