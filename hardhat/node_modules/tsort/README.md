# tsort - node.js topological sort utility

    npm install tsort

## usage

```js
var tsort = require('tsort');

// create an empty graph
var graph = tsort();

// add nodes
graph.add('a', 'b');
graph.add('b', 'c');
graph.add('0', 'a');

// outputs: [ '0', 'a', 'b', 'c' ]
console.dir(graph.sort());

// can add more than one node
graph.add('1', '2', '3', 'a');
// outputs: [ '0', '1', '2', '3', 'a', 'b', 'c' ]
console.dir(graph.sort());

// can add in array form
graph.add(['1', '1.5']);
graph.add(['1.5', 'a']);
// outputs: [ '0', '1', '2', '3', '1.5', 'a', 'b', 'c' ]
console.dir(graph.sort());

// detects cycles
graph.add('first', 'second');
graph.add('second', 'third', 'first');
// throws: Error: There is a cycle in the graph. It is not possible to derive a topological sort.
graph.sort();
```

#license
MIT
