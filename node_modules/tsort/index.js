var util = require('util');

module.exports = function tsort(initial) {
  var graph = new Graph();

  if (initial) {
    initial.forEach(function(entry) {
      Graph.prototype.add.apply(graph, entry);
    });
  }

  return graph;
}

function Graph() {
  this.nodes = {};
}

// Add sorted items to the graph
Graph.prototype.add = function() {
  var self = this;
  var items = [].slice.call(arguments);

  if (items.length == 1 && util.isArray(items[0]))
    items = items[0];

  items.forEach(function(item) {
    if (!self.nodes[item])
      self.nodes[item] = [];
  });

  for (var i = 1; i < items.length; i++) {
    var from = items[i];
    var to = items[i - 1];

    self.nodes[from].push(to);
  }

  return self;
};

// Depth first search
// As given in http://en.wikipedia.org/wiki/Topological_sorting
Graph.prototype.sort = function() {
  var self = this;
  var nodes = Object.keys(this.nodes);

  var sorted = [];
  var marks = {};

  for (var i = 0; i < nodes.length; i++) {
    var node = nodes[i];

    if (!marks[node])
      visit(node);
  }

  return sorted;

  function visit(node) {
    if (marks[node] === 'temp')
      throw new Error("There is a cycle in the graph. It is not possible to derive a topological sort.");
    else if (marks[node])
      return;

    marks[node] = 'temp';
    self.nodes[node].forEach(visit);
    marks[node] = 'perm';

    sorted.push(node);
  }
};
