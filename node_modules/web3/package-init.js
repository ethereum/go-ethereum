/* jshint ignore:start */


// Browser environment
if(typeof window !== 'undefined') {
    web3 = (typeof window.web3 !== 'undefined') ? window.web3 : require('web3');
    BigNumber = (typeof window.BigNumber !== 'undefined') ? window.BigNumber : require('bignumber.js');
}


// Node environment
if(typeof global !== 'undefined') {
    web3 = (typeof global.web3 !== 'undefined') ? global.web3 : require('web3');
    BigNumber = (typeof global.BigNumber !== 'undefined') ? global.BigNumber : require('bignumber.js');
}

/* jshint ignore:end */