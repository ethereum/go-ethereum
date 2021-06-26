---
title: Filtered Tracing
sort_key: B
---

In the previous section you learned how to create a complete trace. However, those traces can include the complete status of the EVM at every point 
in the execution, which is huge. Usually you are only interested in a small subset of this information. To get it, you can specify a JavaScript filter.

**Note:** The JavaScript package used by Geth is [Goja](https://github.com/dop251/goja), which is only up to the
[ECMAScript 5.1 standard](https://262.ecma-international.org/5.1/). This means we cannot use [arrow functions](https://www.w3schools.com/js/js_arrow_function.asp)
and [template literals](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals).


## Getting Started

```javascript
debug.traceTransaction(tx, {tracer: '{' +
   'retVal: [],' +
   'step: function(log,db) {this.retVal.push(log.getPC() + ":" + log.op.toString())},' +
   'fault: function(log,db) {this.retVal.push("FAULT: " + JSON.stringify(log))},' +
   'result: function(ctx,db) {return this.retVal}}'
})
```


## Conclusion

Link to https://geth.ethereum.org/docs/rpc/ns-debug#javascript-based-tracing
