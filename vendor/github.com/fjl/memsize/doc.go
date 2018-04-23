/*
Package memsize computes the size of your object graph.

So you made a spiffy algorithm and it works really well, but geez it's using
way too much memory. Where did it all go? memsize to the rescue!

To get started, find a value that references all your objects and scan it.
This traverses the graph, counting sizes per type.

    sizes := memsize.Scan(myValue)
    fmt.Println(sizes.Total)

memsize can handle cycles just fine and tracks both private and public struct fields.
Unfortunately function closures cannot be inspected in any way.
*/
package memsize
