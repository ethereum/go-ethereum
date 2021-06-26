---
title: Filtered Tracing
sort_key: B
---

In the previous section you learned how to create a complete trace. However, those traces can include the complete status of the EVM at every point 
in the execution, which is huge. Usually you are only interested in a small subset of this information. To get it, you can specify a JavaScript filter.

**Note:** The JavaScript package used by Geth is [Goja](https://github.com/dop251/goja), which is only up to the
[ECMAScript 5.1 standard](https://262.ecma-international.org/5.1/). This means we cannot use [arrow functions](https://www.w3schools.com/js/js_arrow_function.asp)
and [template literals](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals).


## Running a Simple Trace

1. Create a file, `filterTrace_1.js`, with this content:

   ```javascript
   
   tracer = function(tx) {
      return debug.traceTransaction(tx, {tracer: 
         '{' +
            'retVal: [],' +
            'step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},'$
            'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
            'result: function(ctx,db) {return this.retVal}' + 
         '}'
      }) // return debug.traceTransaction ...
   }   // tracer = function ...

   ```

   We could specify this command directly in the JavaScript console, but it would be excessively long and unwieldy.
   
2. Run the [JavaScript console](https://geth.ethereum.org/docs/interface/javascript-console). 
3. Get the hash of a recent transaction. For example, if you use the Goerli network, you can get such a value
   [here](https://goerli.etherscan.io/).
4. Run this command to run the script:

   ```javascript
   loadScript("filterTrace_1.js")
   ```

5. Run the trancer from the script:

   ```javascript
   tracer("<hash of transaction>")
   ```
   
   
### How Does It Work?

We call the same `debug.traceTransaction` function we use for [basic traces](https://geth.ethereum.org/docs/dapp/tracing), but
with a new parameter, `tracer`. This parameter is a string that is the JavaScript object we use. In the fact of the program
above, it is:

```javascript
{
   retVal: [],
   step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},
   fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},
   result: function(ctx,db) {return this.retVal}
}
```
   
## Conclusion

Link to https://geth.ethereum.org/docs/rpc/ns-debug#javascript-based-tracing
