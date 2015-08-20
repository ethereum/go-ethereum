This directory contains two command-line applications *bigtime.js*  and *bigtime-OOM.js*, and for the browser *bignumber-vs-bigdecimal.html*, which enable some of the methods of bignumber.js to be tested against the JavaScript translations of the two versions of BigDecimal in the *lib* directory.

* GWT: java.math.BigDecimal
<https://github.com/iriscouch/bigdecimal.js>
* ICU4J: com.ibm.icu.math.BigDecimal
<https://github.com/dtrebbien/BigDecimal.js>

The BigDecimal in Node's npm registry is the GWT version. It has some bugs: see the Node script *perf/lib/bigdecimal_GWT/bugs.js* for examples of flaws in its *remainder*, *divide* and *compareTo* methods.

An example of using *bigtime.js* to compare the time taken by the bignumber.js `plus` method and the GWT BigDecimal `add` method:  

    $ node bigtime plus 10000 40

This will time 10000 calls to each, using operands of up to 40 random digits and will check that the results match.

For help:

    $ node bigtime -h

*bigtime-OOM.js* works in the same way, but includes separate timings for object creation and method calls.

In general, *bigtime.js* is recommended over *bigtime-OOM.js*, which may run out of memory.

The usage of *bignumber-vs-bigdecimal.html* should be more or less self-explanatory.

---

###### Further notes:

###### bigtime.js

  * Creates random numbers and BigNumber and BigDecimal objects in batches.
  * Unlikely to run out of memory.
  * Doesn't show separate times for object creation and method calls.
  * Tests methods with one or two operands (i.e. includes abs and negate).
  * Doesn't indicate random number creation completion.
  * Doesn't calculate average number of digits of operands.
  * Creates random numbers in exponential notation.

###### bigtime-OOM.js

  * Creates random numbers and BigNumber and BigDecimal objects all in one go.
  * May run out of memory, e.g. if iterations > 500000 and random digits > 40.
  * Shows separate times for object creation and method calls.
  * Only tests methods with two operands (i.e. no abs or negate).
  * Indicates random number creation completion.
  * Calculates average number of digits of operands.
  * Creates random numbers in normal notation.
