(function() {
    if (typeof(Promise) === "undefined")
        window.Promise = Q.Promise;

    var eth = web3.eth;

    web3.setProvider(new web3.providers.QtProvider());
})()
