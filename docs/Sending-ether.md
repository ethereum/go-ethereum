# Sending ether

The basic way of sending a simple transaction of ether with the console is as follows:
```js
> eth.sendTransaction({from:sender, to:receiver, value: amount})
```

Using the built-in JavaScript, you can easily set variables to hold these values. For example:

```js
> var sender = eth.accounts[0];
> var receiver = eth.accounts[1];
> var amount = web3.toWei(0.01, "ether")
```

Alternatively, you can compose a transaction in a single line with:

```js
> eth.sendTransaction({from:eth.coinbase, to:eth.accounts[1], value: web3.toWei(0.05, "ether")})
Please unlock account d1ade25ccd3d550a7eb532ac759cac7be09c2719.
Passphrase: 
Account is now unlocked for this session.
'0xeeb66b211e7d9be55232ed70c2ebb1bcc5d5fd9ed01d876fac5cff45b5bf8bf4'
```

The resulting transaction is `0xeeb66b211e7d9be55232ed70c2ebb1bcc5d5fd9ed01d876fac5cff45b5bf8bf4`

If the password was incorrect you will instead receive an error:
```js
error: could not unlock sender account
```