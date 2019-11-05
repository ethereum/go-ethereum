
## How the fast iterator works

Consider the following example, where we have `6` iterators, sorted from
left to right in ascending order.

Our 'primary' `A` iterator is on the left, containing the elements `[0,1,8]`
```
 A  B  C  D  E  F

 0  1  2  4  7  9
 1  2  9  -  14 13
 8  8  -     15 15
 -  -        -  16
                 -
```
When we call `Next` on the primary iterator, we get (ignoring the future keys)

```
A  B  C  D  E  F

1  1  2  4  7  9
```
We detect that we now got an equality between our element and the next element.
And we need to continue `Next`ing on the next element

```
1  2  2  4  7  9
```
And move on:
```
A  B  C  D  E  F

1  2  9  4  7  9
```
Now we broke out of the equality, but we need to re-sort the element `C`

```
A  B  D  E  F  C

1  2  4  7  9  9
```

And after shifting it rightwards, we check equality again, and find `C == F`, and thus
call `Next` on `C`

```
A  B  D  E  F  C

1  2  4  7  9  -
```
At this point, `C` was exhausted, and is removed

```
A  B  D  E  F

1  2  4  7  9
```
And we're done with this step.

