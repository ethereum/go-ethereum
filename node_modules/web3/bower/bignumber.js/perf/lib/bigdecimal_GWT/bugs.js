// node bugs
// Compare with BigDecTest.java

var i, x, y, r,
    BigDecimal = require('./bigdecimal').BigDecimal;

// remainder

x = new BigDecimal("9.785496E-2");
y = new BigDecimal("-5.9219189762E-2");
r = x.remainder(y);
console.log( r.toString() );
//           0.09785496
// Should be 0.038635770238

x = new BigDecimal("1.23693014661017964112E-5");
y = new BigDecimal("-6.9318042E-7");
r = x.remainder(y);
console.log( r.toPlainString() );
//           0.0000123693014661017964112
// Should be 0.0000005852343261017964112

 // divide

x = new BigDecimal("6.9609119610E-78");
y = new BigDecimal("4E-48");
r = x.divide(y, 40, 6);               // ROUND_HALF_EVEN
console.log( r.toString() );
//           1.7402279903E-30
// Should be 1.7402279902E-30

x = new BigDecimal("5.383458817E-83");
y = new BigDecimal("8E-54");
r = x.divide(y, 40, 6);
console.log( r.toString() );
//           6.7293235213E-30
// Should be 6.7293235212E-30

 // compareTo

x = new BigDecimal("0.04");
y = new BigDecimal("0.079393068");
i = x.compareTo(y);
console.log(i);
//            1
// Should be -1

x = new BigDecimal("7.88749578569876987785987658649E-10");
y = new BigDecimal("4.2545098709E-6");
i = x.compareTo(y);
console.log(i);
//            1
// Should be -1