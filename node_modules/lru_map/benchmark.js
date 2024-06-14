// This is a benchmark suite which will run with nodejs.
// $ node --expose-gc benchmark.js
var assert = require('assert'),
    util = require('util'),
    LRUMap = require('./lru').LRUMap;

// Create a cache with N entries
var N = 10000;

// Run each measurement Iterations times
var Iterations = 1000;


var has_gc_access = typeof gc === 'function';
var gc_collect = has_gc_access ? gc : function(){};

Number.prototype.toHuman = function(divisor) {
  var N = Math.round(divisor ? this/divisor : this);
  var n = N.toString().split('.');
  n[0] = n[0].replace(/(\d)(?=(\d\d\d)+(?!\d))/g, '$1,');
  return n.join('.');
}
Number.prototype.toSignedHuman = function(divisor) {
  var n = this.toHuman(divisor);
  if (this > -1) n = '+'+n;
  return n;
}

function measure(block) {
  gc_collect();
  var elapsed = 0, start, usec, n;
  var sm = process.memoryUsage();
  if (Iterations && Iterations > 0) {
    start = new Date();
    for (n = 0; n < Iterations; n++) {
      block();
    }
    usec = ((new Date() - start) / Iterations) * 1000;
  } else {
    start = new Date();
    block();
    usec = (new Date() - start) * 1000;
  }

  var msg = '\n----------\n  ' + block.toString().replace(/\n/g, "\n  ") + '\n';

  if (has_gc_access) {
    gc_collect();
    var em = process.memoryUsage();
    var mrssd = em.rss - sm.rss;
    var mhtotd = em.heapTotal - sm.heapTotal;
    var mhusedd = em.heapUsed - sm.heapUsed;
    msg += '   rss:        ' + mrssd.toSignedHuman(1024) + ' kB -- (' +
           sm.rss.toHuman(1024) + ' kB -> ' + em.rss.toHuman(1024) + ' kB)\n';
    if (typeof sm.vsize === 'number') {
      var mvsized = em.vsize - sm.vsize;
      msg +=
           '   vsize:      ' + mvsized.toSignedHuman(1024) + ' kB -- (' +
           sm.vsize.toHuman(1024) + ' kB -> ' + em.vsize.toHuman(1024) + ' kB)\n';
    }
    msg += '   heap total: ' + mhtotd.toSignedHuman(1024) + ' kB -- (' +
           sm.heapTotal.toHuman(1024) + ' kB -> ' + em.heapTotal.toHuman(1024) + ' kB)\n' +
           '   heap used:  ' + mhusedd.toSignedHuman(1024) + ' kB -- (' +
           sm.heapUsed.toHuman(1024) + ' kB -> ' + em.heapUsed.toHuman(1024) + ' kB)\n\n';
  }

  var call_avg = usec;
  msg += '  -- ' + (call_avg / 1000) + ' ms avg per iteration --\n';

  process.stdout.write(msg);
}

var c = new LRUMap(N);

console.log('N = ' + N + ', Iterations = ' + Iterations);

// We should probably spin up the system in some way, or repeat the benchmarks a
// few times, since initial heap resizing takes considerable time.

measure(function(){
  // 1. put
  //    Simply append a new entry.
  //    There will be no reordering since we simply append to the tail.
  for (var i=N; --i;)
    c.set('key'+i, i);
});

measure(function(){
  // 2. get recent -> old
  //    Get entries starting with newest, effectively reversing the list.
  //
  // a. For each get, a find is first executed implemented as a native object with
  //    keys mapping to entries, so this should be reasonably fast as most native
  //    objects are implemented as hash maps.
  //
  // b. For each get, the aquired item will be moved to tail which includes a
  //    maximum of 7 assignment operations (minimum 3).
  for (var i=1,L=N+1; i<L; ++i)
    c.get('key'+i, i);
});


measure(function(){
  // 3. get old -> recent
  //    Get entries starting with oldest, effectively reversing the list.
  //
  //  - Same conditions apply as for test 2.
  for (var i=1,L=N+1; i<L; ++i)
    c.get('key'+i);
});

measure(function(){
  // 4. get missing
  //    Get try to get entries not in the cache.
  //  - Same conditions apply as for test 2, section a.
  for (var i=1,L=N+1; i<L; ++i)
    c.get('xkey'+i);
});

measure(function(){
  // 5. put overflow
  //    Overflow the cache with N more items than it can hold.
  // a. The complexity of put in this case should be:
  //    ( <get whith enough space> + <shift> )
  for (var i=N; --i;)
    c.set('key2_'+i, i);
});


measure(function(){
  // 6. shift head -> tail
  //    Remove all entries going from head to tail
  for (var i=1,L=N+1; i<L; ++i)
    c.shift();
});

measure(function(){
  // 7. put 
  //    Simply put N new items into an empty cache with exactly N space.
  for (var i=N; --i;)
    c.set('key'+i, i);
});

// pre-build random key array
var shuffledKeys = Array.from(c.keys());
shuffledKeys.sort(function (){return Math.random()-0.5; });

measure(function(){
  // 8. delete random
  // a. Most operations (which are not entries at head or tail) will cause closes
  //    siblings to be relinked.
  for (var i=shuffledKeys.length, key; key = shuffledKeys[--i]; ) {
    c.delete('key'+i, i);
  }
});
