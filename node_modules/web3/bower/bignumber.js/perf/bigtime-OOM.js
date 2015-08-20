
var arg, i, max, method, methodIndex, decimalPlaces,
    reps, rounding, start, timesEqual, Xs, Ys,
    bdM, bdMT, bdOT, bdRs, bdXs, bdYs,
    bnM, bnMT, bnOT, bnRs, bnXs, bnYs,
    memoryUsage, showMemory, bnR, bdR,
    prevRss, prevHeapUsed, prevHeapTotal,
    args = process.argv.splice(2),
    BigDecimal = require('./lib/bigdecimal_GWT/bigdecimal').BigDecimal,
    BigNumber = require('../bignumber'),
    bdMs = ['add', 'subtract', 'multiply', 'divide', 'remainder', 'compareTo', 'pow'],
    bnMs1 = ['plus', 'minus', 'times', 'dividedBy', 'modulo', 'comparedTo', 'toPower'],
    bnMs2 = ['', '', '', 'div', 'mod', 'cmp', ''],
    Ms = [bdMs, bnMs1, bnMs2],
    allMs = [].concat.apply([], Ms),
    expTotal = 0,
    total = 0,

    ALWAYS_SHOW_MEMORY = false,
    DEFAULT_MAX_DIGITS = 20,
    DEFAULT_POW_MAX_DIGITS = 20,
    DEFAULT_REPS = 1e4,
    DEFAULT_POW_REPS = 1e2,
    DEFAULT_PLACES = 20,
    MAX_POWER = 50,

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

        total += n + 1;

        // 50% chance of negative
        return Math.random() > 0.5 ? r : '-' + r;
    },

    pad = function (str) {
        str += '... ';
        while (str.length < 26) str += ' ';
        return str;
    },

    getFastest = function (bn, bd) {
        var r;
        if (Math.abs(bn - bd) > 2) {
            r = 'Big' + ((bn < bd)
            ? 'Number ' + (bn ? parseFloat((bd / bn).toFixed(1)) : bd)
            : 'Decimal ' + (bd ? parseFloat((bn / bd).toFixed(1)) : bn)) +
                ' times faster';
        } else {
            timesEqual = 1;
            r = 'Times approximately equal';
        }
        return r;
    },

    showMemoryChange = function () {
        if (showMemory) {
            memoryUsage = process.memoryUsage();

            var rss = memoryUsage.rss,
                heapUsed = memoryUsage.heapUsed,
                heapTotal = memoryUsage.heapTotal;

            console.log(' Change in memory usage: ' +
                ' rss: ' +  toKB(rss - prevRss) +
                ', hU: ' + toKB(heapUsed - prevHeapUsed) +
                ', hT: ' + toKB(heapTotal - prevHeapTotal));
            prevRss = rss; prevHeapUsed = heapUsed; prevHeapTotal = heapTotal;
        }
    },

    toKB = function (m) {
        return parseFloat((m / 1024).toFixed(1)) + ' KB';
    };


// PARSE COMMAND LINE AND SHOW HELP

if (arg = args[0], typeof arg != 'undefined' && !isFinite(arg) &&
    allMs.indexOf(arg) == -1 && !/^-*m$/i.test(arg)) {
    console.log(
    '\n node bigtime-OOM [METHOD] [METHOD CALLS [MAX DIGITS [DECIMAL PLACES]]]\n' +
    '\n METHOD: The method to be timed and compared with the automatically' +
    '\n         chosen corresponding method from BigDecimal or BigNumber\n' +
    '\n BigDecimal: add  subtract multiply divide remainder compareTo pow' +
    '\n BigNumber:  plus minus times dividedBy modulo comparedTo toPower' +
    '\n             (div mod cmp pow)' +
    '\n\n METHOD CALLS: The number of method calls to be timed' +
    '\n\n MAX DIGITS: The maximum number of digits of the random ' +
    '\n             numbers used in the method calls' +
    '\n\n DECIMAL PLACES: The number of decimal places used in division' +
    '\n                 (The rounding mode is randomly chosen)' +
    '\n\n Default values: METHOD: randomly chosen' +
    '\n                 METHOD CALLS: ' + DEFAULT_REPS +
    '  (pow: ' + DEFAULT_POW_REPS + ')' +
    '\n                 MAX DIGITS: ' + DEFAULT_MAX_DIGITS +
    '  (pow: ' + DEFAULT_POW_MAX_DIGITS + ')' +
    '\n                 DECIMAL PLACES: ' + DEFAULT_PLACES + '\n' +
    '\n E.g.s node bigtime-OOM\n       node bigtime-OOM minus' +
    '\n       node bigtime-OOM add 100000' +
    '\n       node bigtime-OOM times 20000 100' +
    '\n       node bigtime-OOM div 100000 50 20' +
    '\n       node bigtime-OOM 9000' +
    '\n       node bigtime-OOM 1000000 20\n' +
    '\n To show memory usage include an argument m or -m' +
    '\n E.g.  node bigtime-OOM m add');
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
            if (reps == null) {
                reps = arg <= 1e10 ? arg : 0;
            } else if (max == null) {
                max = arg <= 1e6 ? arg : 0;
            } else if (decimalPlaces == null) {
                decimalPlaces = arg <= 1e6 ? arg : DEFAULT_PLACES;
            }
        } else if (/^-*m$/i.test(arg)) {
            showMemory = true;
        } else if (method == null) {
            method = arg;
        }
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

    Xs = [reps], Ys = [reps];
    bdXs = [reps], bdYs = [reps], bdRs = [reps];
    bnXs = [reps], bnYs = [reps], bnRs = [reps];
    showMemory = showMemory || ALWAYS_SHOW_MEMORY;

    console.log('\n BigNumber %s vs BigDecimal %s', bnM, bdM);
    console.log('\n Method calls: %d', reps);

    if (bdM == 'divide') {
        rounding = Math.floor(Math.random() * 7);
        console.log('\n Decimal places: %d\n Rounding mode: %d', decimalPlaces, rounding);
        BigNumber.config(decimalPlaces, rounding);
    }

    if (showMemory) {
        memoryUsage = process.memoryUsage();
        console.log(' Memory usage:            rss: ' +
            toKB(prevRss = memoryUsage.rss) + ', hU: ' +
            toKB(prevHeapUsed = memoryUsage.heapUsed) + ', hT: ' +
            toKB(prevHeapTotal = memoryUsage.heapTotal));
    }


    // CREATE RANDOM NUMBERS

    // POW: BigDecimal requires JS Number type for exponent argument
    if (bdM == 'pow') {

        process.stdout.write('\n Creating ' + reps +
            ' random numbers (max. digits: ' + max + ')... ');

        for (i = 0; i < reps; i++) {
            Xs[i] = getRandom(max);
        }
        console.log('done\n Average number of digits: %d',
            ((total / reps) | 0));

        process.stdout.write(' Creating ' + reps +
            ' random integer exponents (max. value: ' + MAX_POWER + ')... ');

        for (i = 0; i < reps; i++) {
            bdYs[i] = bnYs[i] = Math.floor(Math.random() * (MAX_POWER + 1));
            expTotal += bdYs[i];
        }
        console.log('done\n Average value: %d', ((expTotal / reps) | 0));

        showMemoryChange();


        // POW: TIME CREATION OF BIGDECIMALS

        process.stdout.write('\n Creating BigDecimals...  ');

        start = +new Date();
            for (i = 0; i < reps; i++) {
                bdXs[i] = new BigDecimal(Xs[i]);
            }
        bdOT = +new Date() - start;

        console.log('done. Time taken: %s ms', bdOT || '<1');

        showMemoryChange();


        // POW: TIME CREATION OF BIGNUMBERS

        process.stdout.write(' Creating BigNumbers...   ');

        start = +new Date();
            for (i = 0; i < reps; i++) {
                bnXs[i] = new BigNumber(Xs[i]);
            }
        bnOT = +new Date() - start;

        console.log('done. Time taken: %s ms', bnOT || '<1');


    // NOT POW
    } else {

        process.stdout.write('\n Creating ' + (reps * 2) +
            ' random numbers (max. digits: ' + max + ')... ');


        for (i = 0; i < reps; i++) {
            Xs[i] = getRandom(max);
            Ys[i] = getRandom(max);
        }
        console.log('done\n Average number of digits: %d',
            ( total / (reps * 2) ) | 0);

        showMemoryChange();


        // TIME CREATION OF BIGDECIMALS

        process.stdout.write('\n Creating BigDecimals...  ');

        start = +new Date();
            for (i = 0; i < reps; i++) {
                bdXs[i] = new BigDecimal(Xs[i]);
                bdYs[i] = new BigDecimal(Ys[i]);
            }
        bdOT = +new Date() - start;

        console.log('done. Time taken: %s ms', bdOT || '<1');

        showMemoryChange();


        // TIME CREATION OF BIGNUMBERS

        process.stdout.write(' Creating BigNumbers...   ');

        start = +new Date();
            for (i = 0; i < reps; i++) {
                bnXs[i] = new BigNumber(Xs[i]);
                bnYs[i] = new BigNumber(Ys[i]);
            }
        bnOT = +new Date() - start;

        console.log('done. Time taken: %s ms', bnOT || '<1');
    }

    showMemoryChange();

    console.log('\n Object creation: %s\n', getFastest(bnOT, bdOT));


    // TIME BIGDECIMAL METHOD CALLS

    process.stdout.write(pad(' BigDecimal ' + bdM));

    if (bdM == 'divide') {
        start = +new Date();
            while (i--) bdRs[i] = bdXs[i][bdM](bdYs[i], decimalPlaces, rounding);
        bdMT = +new Date() - start;
    } else {
        start = +new Date();
            while (i--) bdRs[i] = bdXs[i][bdM](bdYs[i]);
        bdMT = +new Date() - start;
    }

    console.log('done. Time taken: %s ms', bdMT || '<1');


    // TIME BIGNUMBER METHOD CALLS

    i = reps;
    process.stdout.write(pad(' BigNumber  ' + bnM));

    start = +new Date();
        while (i--) bnRs[i] = bnXs[i][bnM](bnYs[i]);
    bnMT = +new Date() - start;

    console.log('done. Time taken: %s ms', bnMT || '<1');


    // TIMINGS SUMMARY

    console.log('\n Method calls:    %s', getFastest(bnMT, bdMT));

    if (!timesEqual) {
        console.log('\n Overall:         ' +
            getFastest((bnOT || 1) + (bnMT || 1), (bdOT || 1) + (bdMT || 1)));
    }



    // CHECK FOR MISMATCHES

    process.stdout.write('\n Checking for mismatches... ');

    for (i = 0; i < reps; i++) {

        bnR = bnRs[i].toString();
        bdR = bdRs[i].toPlainString();

        // Strip any trailing zeros from non-integer BigDecimals
        if (bdR.indexOf('.') != -1) {
            bdR = bdR.replace(/\.?0+$/, '');
        }

        if (bdR !== bnR) {
            console.log('breaking on first mismatch (result number %d):' +
                '\n\n BigDecimal: %s\n BigNumber:  %s', i, bdR, bnR);
            console.log('\n x: %s\n y: %s', Xs[i], Ys[i]);

            if (bdM == 'divide') {
                console.log('\n dp: %d\n r: %d',decimalPlaces, rounding);
            }
            break;
        }
    }
    if (i == reps) {
        console.log('done. None found.\n');
    }
}



