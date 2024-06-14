var assert = require('assert');
var tsort = require('../')

describe('tsort', function() {
  it('should create an empty graph', function() {
    var graph = tsort();
    assert.equal(typeof graph.sort, 'function');
    assert.equal(typeof graph.add, 'function');
  });

  it('should create a graph from initial data', function() {
    var graph = tsort([['a', 'b', 'c'], ['b', 'c']]);
    assert.equal(Object.keys(graph.nodes).length, 3);

    var graph = tsort([['a', 'b', 'c'], ['b', 'c'], ['k', 'l']]);
    assert.equal(Object.keys(graph.nodes).length, 5);
  });

  it('should add items to a graph', function() {
    var graph = tsort([['a', 'b', 'c']]);
    graph.add('b', 'd');
    graph.add('b', 'e');
    graph.add('k', 'l');
    assert.equal(Object.keys(graph.nodes).length, 7);
  });

  it('should add arrays to a graph', function() {
    var graph = tsort([['a', 'b', 'c']]);
    graph.add(['b', 'd']);
    graph.add(['b', 'e']);
    graph.add(['k', 'l']);
    assert.equal(Object.keys(graph.nodes).length, 7);
  });

  it('should sort an empty graph', function() {
    var result = tsort().sort();
    assert.equal(result.length, 0);
  });

  it('should sort a trivial graph', function() {
    var result = tsort([
      ['a', 'b'],
      ['b', 'c'],
      ['0', 'a']
    ]).sort();

    assert.deepEqual(result, ['0', 'a', 'b', 'c']);
  });

  it('should sort graph 1', function() {
    var result = tsort([
      ['a', 'b'],
      ['b', 'c'],
      ['0', 'a']
    ]).sort();

    assert.deepEqual(result, ['0', 'a', 'b', 'c']);
  });

  it('should sort graph 2', function() {
    var result = tsort([
      ['a', 'b'],
      ['b', 'c'],
      ['0', 'a'],
      ['t', 'n', 's'],
      ['k', 'n']
    ]).sort();

    assert.deepEqual(result, ['0', 'a', 'b', 'c', 't', 'k', 'n', 's']);
  });

  it('should detect a cycle', function() {
    assert.throws(function() {
      var result = tsort([
        ['a', 'b'],
        ['b', 'c'],
        ['0', 'a'],
        ['c', 'k', '0']
      ]).sort();
    });
  });
});
