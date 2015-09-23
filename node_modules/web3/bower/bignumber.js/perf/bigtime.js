
var arg, i, j, max, method, methodIndex, decimalPlaces, rounding, reps, start,
     timesEqual, xs, ys, prevRss, prevHeapUsed, prevHeapTotal, showMemory,
    bdM, bdT, bdR, bdRs,
    bnM, bnT, bnR, bnRs,
    args = process.argv.splice(2),
    BigDecimal = require('./lib/bigdecimal_GWT/bigdecimal').BigDecimal,
    BigNumber = require('../bignumber'),
    bdMs = ['add', 'subtract', 'multiply', 'divide', 'remainder',
            'compareTo', 'pow', 'negate', 'abs'],
    bnMs1 = ['plus', 'minus', 'times', 'dividedBy', 'modulo',
             'comparedTo', 'toPower', 'negated', 'abs'],
    bnMs2 = ['', '', '', 'div', 'mod', 'cmp', '', 'neg', ''],
    Ms = [bdMs, bnMs1, bnMs2],
    allMs = [].concat.apply([], Ms),
    bdTotal = 0,
    bnTotal = 0,
    BD = {},
    BN = {},

    ALWAYS_SHOW_MEMORY = false,
    DEFAULT_MAX_DIGITS = 20,
    DEFAULT_POW_MAX_DIGITS = 20,
    DEFAULT_REPS = 1e4,
    DEFAULT_POW_REPS = 1e2,
    DEFAULT_PLACES = 20,
    MAX_POWER = 50,
    MAX_RANDOM_EXPONENT = 100,

    getRandom = function (maxDigits) {
        var i = 0, z,
            // number of digits - 1
            n = Math.random() * ( maxDigits || 1 ) | 0,
            r = ( Math.random() * 10 | 0 ) + '';

        if ( n ) {
            if ( z = r === '0' ) {
                r += '.';
            }

            for ( ; i++ < n; r += Math.random() * 10 | 0 ){}

            // 20% chance of integer
            if ( !z && Math.random() > 0.2 )
                r = r.slice( 0, i = ( Math.random() * n | 0 ) + 1 ) + '.' + r.slice(i);
        }

        // Avoid 'division by zero' error with division and modulo.
        if ((bdM == 'divide' || bdM == 'remainder') && parseFloat(r) === 0)
            r = ( ( Math.random() * 9 | 0 ) + 1 ) + '';

        // 50% chance of negative
        return Math.random() > 0.5 ? r : '-' + r;
    },

    // Returns exponential notation.
    //getRandom = function (maxDigits) {
    //    var i = 0,
    //        // n is the number of significant digits - 1
    //        n = Math.random() * (maxDigits || 1) | 0,
    //        r = ( ( Math.random() * 9 | 0 ) + 1 ) + ( n ? '.' : '' );
    //
    //    for (; i++ < n; r += Math.random() * 10 | 0 ){}
    //
    //    // Add exponent.
    //    r += 'e' + ( Math.random() > 0.5 ? '+' : '-' ) +
    //               ( Math.random() * MAX_RANDOM_EXPONENT | 0 );
    //
    //    // 50% chance of being negative.
    //    return Math.random() > 0.5 ? r : '-' + r
    //},

    getFastest = function (bn, bd) {
        var r;
        if (Math.abs(bn - bd) > 2) {
            r = 'Big' + ((bn < bd)
                ? 'Number was ' + (bn ? parseFloat((bd / bn).toFixed(1)) : bd)
                : 'Decimal was ' + (bd ? parseFloat((bn / bd).toFixed(1)) : bn)) +
                    ' times faster';
        } else {
            timesEqual = 1;
            r = 'Times approximately equal';
        }
        return r;
    },

    getMemory = function (obj) {
        if (showMemory) {
            var mem = process.memoryUsage(),
                rss = mem.rss,
                heapUsed = mem.heapUsed,
                heapTotal = mem.heapTotal;

            if (obj) {
                obj.rss += (rss - prevRss);
                obj.hU += (heapUsed - prevHeapUsed);
                obj.hT += (heapTotal - prevHeapTotal);
            }
            prevRss = rss;
            prevHeapUsed = heapUsed;
            prevHeapTotal = heapTotal;
        }
    },

    getMemoryTotals = function (obj) {
        function toKB(m) {return parseFloat((m / 1024).toFixed(1))}
        return '\trss: ' + toKB(obj.rss) +
               '\thU: ' + toKB(obj.hU) +
               '\thT: ' + toKB(obj.hT);
    };


if (arg = args[0], typeof arg != 'undefined' && !isFinite(arg) &&
    allMs.indexOf(arg) == -1 && !/^-*m$/i.test(arg)) {
    console.log(
        '\n node bigtime [METHOD] [METHOD CALLS [MAX DIGITS [DECIMAL PLACES]]]\n' +
            '\n METHOD: The method to be timed and compared with the' +
            '\n \t corresponding method from BigDecimal or BigNumber\n' +
            '\n   BigDecimal: add subtract multiply divide remainder' +
            ' compareTo pow\n\t\tnegate abs\n\n   BigNumber: plus minus times' +
            ' dividedBy modulo comparedTo toPower\n\t\tnegated abs' +
            ' (div mod cmp pow neg)' +
            '\n\n METHOD CALLS: The number of method calls to be timed' +
            '\n\n MAX DIGITS: The maximum number of digits of the random ' +
            '\n\t\tnumbers used in the method calls\n\n ' +
            'DECIMAL PLACES: The number of decimal places used in division' +
            '\n\t\t(The rounding mode is randomly chosen)' +
            '\n\n Default values: METHOD: randomly chosen' +
            '\n\t\t METHOD CALLS: ' + DEFAULT_REPS +
            '  (pow: ' + DEFAULT_POW_REPS + ')' +
            '\n\t\t MAX DIGITS: ' + DEFAULT_MAX_DIGITS +
            '  (pow: ' + DEFAULT_POW_MAX_DIGITS + ')' +
            '\n\t\t DECIMAL PLACES: ' + DEFAULT_PLACES + '\n' +
            '\n E.g.   node bigtime\n\tnode bigtime minus\n\tnode bigtime add 100000' +
            '\n\tnode bigtime times 20000 100\n\tnode bigtime div 100000 50 20' +
            '\n\tnode bigtime 9000\n\tnode bigtime 1000000 20\n' +
            '\n To show memory usage, include an argument m or -m' +
            '\n E.g.   node bigtime m add');
} else {

    BigNumber.config({
        EXPONENTIAL_AT: 1E9,
        RANGE: 1E9,
        ERRORS: false,
        MODULO_MODE: 1,
        POW_PRECISION: 10000
    });

    Number.prototype.toPlainString = Number.prototype.toString;

    for (i = 0; i < args.length; i++) {
        arg = args[i];
        if (isFinite(arg)) {
            arg = Math.abs(parseInt(arg));
            if (reps == null)
                reps = arg <= 1e10 ? arg : 0;
            else if (max == null)
                max = arg <= 1e6 ? arg : 0;
            else if (decimalPlaces == null)
                decimalPlaces = arg <= 1e6 ? arg : DEFAULT_PLACES;
        } else if (/^-*m$/i.test(arg))
            showMemory = true;
        else if (method == null)
            method = arg;
    }

    for (i = 0;
         i < Ms.length && (methodIndex = Ms[i].indexOf(method)) == -1;
         i++) {}

    bnM = methodIndex == -1
        ? bnMs1[methodIndex = Math.floor(Math.random() * bdMs.length)]
        : (Ms[i][0] == 'add' ? bnMs1 : Ms[i])[methodIndex];

    bdM = bdMs[methodIndex];

    if (!reps)
        reps = bdM == 'pow' ? DEFAULT_POW_REPS : DEFAULT_REPS;
    if (!max)
        max = bdM == 'pow' ? DEFAULT_POW_MAX_DIGITS : DEFAULT_MAX_DIGITS;
    if (decimalPlaces == null)
        decimalPlaces = DEFAULT_PLACES;

    xs = [reps], ys = [reps], bdRs = [reps], bnRs = [reps];
    BD.rss = BD.hU = BD.hT = BN.rss = BN.hU = BN.hT = 0;
    showMemory = showMemory || ALWAYS_SHOW_MEMORY;

    console.log('\n BigNumber %s vs BigDecimal %s\n' +
        '\n Method calls: %d\n\n Random operands: %d', bnM, bdM, reps,
        bdM == 'abs' || bdM == 'negate' || bdM == 'abs' ? reps : reps * 2);

    console.log(' Max. digits of operands: %d', max);

    if (bdM == 'divide') {
        rounding = Math.floor(Math.random() * 7);
        console.log('\n Decimal places: %d\n Rounding mode: %d', decimalPlaces, rounding);
        BigNumber.config(decimalPlaces, rounding);
    }

    process.stdout.write('\n Testing started');

    outer:
    for (; reps > 0; reps -= 1e4) {

        j = Math.min(reps, 1e4);


        // GENERATE RANDOM OPERANDS

        for (i = 0; i < j; i++) {
            xs[i] = getRandom(max);
        }

        if (bdM == 'pow') {
            for (i = 0; i < j; i++) {
                ys[i] = Math.floor(Math.random() * (MAX_POWER + 1));
            }
        } else if (bdM != 'abs' && bdM != 'negate') {
            for (i = 0; i < j; i++) {
                ys[i] = getRandom(max);
            }
        }

        getMemory();


        // BIGDECIMAL

        if (bdM == 'divide') {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bdRs[i] = new BigDecimal(xs[i])[bdM](new BigDecimal(ys[i]),
                    decimalPlaces, rounding);
            }
            bdT = +new Date() - start;

        } else if (bdM == 'pow') {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bdRs[i] = new BigDecimal(xs[i])[bdM](ys[i]);
            }
            bdT = +new Date() - start;

        } else if (bdM == 'abs' || bdM == 'negate') {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bdRs[i] = new BigDecimal(xs[i])[bdM]();
            }
            bdT = +new Date() - start;

        } else {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bdRs[i] = new BigDecimal(xs[i])[bdM](new BigDecimal(ys[i]));
            }
            bdT = +new Date() - start;

        }

        getMemory(BD);


        // BIGNUMBER

        if (bdM == 'pow') {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bnRs[i] = new BigNumber(xs[i])[bnM](ys[i]);
            }
            bnT = +new Date() - start;

        } else if (bdM == 'abs' || bdM == 'negate') {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bnRs[i] = new BigNumber(xs[i])[bnM]();
            }
            bnT = +new Date() - start;

        } else {

            start = +new Date();
            for (i = 0; i < j; i++) {
                bnRs[i] = new BigNumber(xs[i])[bnM](new BigNumber(ys[i]));
            }
            bnT = +new Date() - start;

        }

        getMemory(BN);


        // CHECK FOR MISMATCHES

        for (i = 0; i < j; i++) {
            bnR = bnRs[i].toString();
            bdR = bdRs[i].toPlainString();

            // Strip any trailing zeros from non-integer BigDecimals
            if (bdR.indexOf('.') != -1) {
                bdR = bdR.replace(/\.?0+$/, '');
            }

            if (bdR !== bnR) {
                console.log('\n breaking on first mismatch (result number %d):' +
                '\n\n BigDecimal: %s\n BigNumber:  %s', i, bdR, bnR);
                console.log('\n x: %s\n y: %s', xs[i], ys[i]);

                if (bdM == 'divide')
                    console.log('\n dp: %d\n r: %d',decimalPlaces, rounding);
                break outer;
            }
        }

        bdTotal += bdT;
        bnTotal += bnT;

        process.stdout.write(' .');
    }


    // TIMINGS SUMMARY


    if (i == j) {
        console.log(' done\n\n No mismatches.');
        if (showMemory) {
            console.log('\n Change in memory usage (KB):' +
                        '\n\tBigDecimal' + getMemoryTotals(BD) +
                        '\n\tBigNumber ' + getMemoryTotals(BN));
        }
        console.log('\n Time taken:' +
            '\n\tBigDecimal  ' + (bdTotal || '<1') + ' ms' +
            '\n\tBigNumber   ' + (bnTotal || '<1') + ' ms\n\n ' +
            getFastest(bnTotal, bdTotal) + '\n');
    }
}