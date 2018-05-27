  CookieJar - A contestant's toolbox
======================================

CookieJar is a small collection of common algorithms, data structures and library extensions that were deemed handy for computing competitions at one point or another.

This toolbox is a work in progress for the time being. It may be lacking, and it may change drastically between commits (although every effort is made not to). You're welcome to use it, but it's your head on the line :)

  Installation
----------------

To get the package, execute:

    go get gopkg.in/karalabe/cookiejar.v2

To import this package, add the following line to your code:

    import "gopkg.in/karalabe/cookiejar.v2"

For more details, see the [package documentation](http://godoc.org/gopkg.in/karalabe/cookiejar.v2).

  Contents
------------

Algorithms:
 - Graph
     - [Breadth First Search](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/graph/bfs)
     - [Depth First Search](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/graph/dfs)

Data structures:
 - [Bag](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/bag)
 - [Deque](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/deque)
 - [Graph](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/graph)
 - [Priority Queue](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/prque)
 - [Queue](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/queue)
 - [Set](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/set)
 - [Stack](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/collections/stack)
 
Extensions:
 - [fmt](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/exts/fmtext)
     - `Scan` and `Fscan` for `int`, `float64`, `string` and lines
 - [math](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/exts/mathext)
     - `Abs` for `int`
     - `Min` and `Max` for `int`, `big.Int` and `big.Rat`
     - `Sign` for `int` and `float64`
 - [os](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/exts/osext)
     - `Open` and `Create` without error codes
 - [sort](http://godoc.org/gopkg.in/karalabe/cookiejar.v2/exts/sortext)
     - `Sort` and `Search` for `big.Int` and `big.Rat`
     - `Unique` for any `sort.Interface`

Below are the performance results for the data structures and the complexity analysis for the algorithms.

  Performance
---------------

Intel(R) Core(TM) i7-2600 CPU @ 3.40GHz:
```
- bag
    - BenchmarkInsert    309     ns/op
    - BenchmarkRemove    197     ns/op
    - BenchmarkDo        28.1    ns/op
- deque
    - BenchmarkPush      25.4    ns/op
    - BenchmarkPop       6.72    ns/op
- prque
    - BenchmarkPush      171     ns/op
    - BenchmarkPop       947     ns/op
- queue
    - BenchmarkPush      23.0    ns/op
    - BenchmarkPop       5.92    ns/op
- set
    - BenchmarkInsert    259     ns/op
    - BenchmarkRemove    115     ns/op
    - BenchmarkDo        20.9    ns/op
- stack
    - BenchmarkPush      16.4    ns/op
    - BenchmarkPop       6.45    ns/op
```

  Complexity
--------------

| Algorithm | Time complexity | Space complexity |
|:---------:|:---------------:|:----------------:|
| graph/bfs | O(E)            | O(V)             |
| graph/dfs | O(E)            | O(E)             |

  Here be dragons :)
----------------------

```
     .     _///_,
   .      / ` ' '>
     )   o'  __/_'>
    (   /  _/  )_\'>
     ' "__/   /_/\_>
         ____/_/_/_/
        /,---, _/ /
       ""  /_/_/_/
          /_(_(_(_                 \
         (   \_\_\\_               )\
          \'__\_\_\_\__            ).\
          //____|___\__)           )_/
          |  _  \'___'_(           /'
           \_ (-'\'___'_\      __,'_'
           __) \  \\___(_   __/.__,'
        ,((,-,__\  '", __\_/. __,'
                     '"./_._._-'
```
