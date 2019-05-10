## WebAuthETH

*Say it fast five times. It's fun*

**Elevator Pitch**: Buy an NFT with your fingerprint on a stock Android phone on Chrome.

**What:** Enable Webauthn keys to control ETH SC wallets. 

**Why:** 1) Browser blockchain UX is cumbersome to use 2) Extension installs are bad for product conversion rates 3) Mobile blockchain is impossible without a dapp browser

**Philosophy**: We don't care too much about gas right now, it should work on as many devices as possible, done and working is better than perfect

Requirments: (email only used for multi device login and recovery)

## First Use Flow

User arrives at a Dapper auth enabled app called Foobar through a Webauthn enabled browser for the first time. User has not used this app or Dapper auth or the Foobar app before.

1. User loads application page

2. Script tag in page loads `dapper.js` which exposes a `web3`

3. Script injects a login overlay over the web page with a dapper auth iframe

4. User clicks *Log In with Dapper* button. Application calls `dapper.authenticate(appId)` which returns a `Promise`

5. This triggers an windowed  iframe at location `https://auth.dapperauth.com/appId=XYZ`

6. `dapper.js` looks in `localStorage` for `dapper-auth-credential-id-XYZ`

   1. will not find this key since user is new and a credential for XYZ does not exist
   2. Will trigger start API lookup for a valid app credential for this user

7. `dapper.js` looks in `localStorage` for  `dapper-auth-user` for the user's email id 

   1. If not found user will be prompted to enter an email ID
   2. Fetch `https://auth.dapperauth.com/credentials/appId=XYZ&user=email`
      1. Response will return null if no credentials are currently associated with this appId
      2. will trigger the wallet create flow

8. Dialog opens, presenting the user with the app's info and requested permissions

   1. User taps accept in the dialog
   2. If the user declines, throw `UserDeniedPermission`

9. Call `navigator.credentials.create` with a preset create tx / challenge below. User will be prompted to create a new Webauthn credential

   ```js
   let tx = 'CREATE'
   let challenge = kekkack(payload)
   let webAuthnResponse = await navigator.credentials.create({
     payload, challenge, ...params
   })
   let { pubKey, sig } = getPubKeySignature(webAuthnResponse)
   ```

   We use the returned `response` to set the compressed Ed25519 `pubKey` and `sig` which is a signature of `pubKey` over challenge.

10. Trigger a recaptcha flow to get a `captcha` token from Google

11. Establish Websocket to `https://auth.dapper.com/register?payload=2f3a42...` Transmit this payload base64 encoded to establish connection

    ```js
    let payload = { appId, pubKey, sig, tx, captcha }
    paylod = window.btoa(JSON.stringify(payload))
    ```

    - Server will recieve socket connection requests at `auth.dapper.com/register``. It parses the message frame as JSON and does the following:
      - Ensure `tx === "CREATE"`. If failed, reject the connection
      - Ensure `ecrecoverEd25519(sig) === pubKey` If failed, reject
      - Verify the captcha with the Google. If it failed, reject
      - Accept the socket connection and kick off the wallet creation 
        - Instantiate wallet smart contract with constructor arguments `(pubKey, appId)`, which will allow the new contract wallet to be used with `pubKey` for contracts related to `appId` 
        - Wait for this contract creation transaction to be mined and publish the newly deployed wallet address event on a queue with `id=$pubKey`
        - Listen for this queue event from your socket event loop and once you recieve it, write the wallet address to the client socket and terminate the connection after recieving an ack

12. Dapper auth `iframe`  recieves this wallet address, acks the message and returns it to parent frame and minimizes its frame frome view.

13. `dapper.js` in parent frame constructs a wrapped instance of `web3` with `.accounts[0]` set to the newly created wallet address.

14. Resolve the `Promise` for `dapper.authenticate(appId)`



### Todo

- ~~Scaffold a new vm precompile function for ed25519~~
-  Decide on a public key compression.
- Create a failing test with output from a call to `navigator.credentials.create` against ecrecover25519
- Write implementation for `ecrecoverEd25519` contract
- Write a simple docker file, run new geth image with kube on GCP with a new testnet with `id=1337`
- Deploy a test contract for  `ecrecoverEd25519`
- Write a passing test in JS with `truffle` with provider connected to new `geth` testnet
- Implement `web3.js` wrapper with pubkey gen, compression methods that connects to our `geth`
- Scaffold web3 instance that uses smart contract wallet implementation
- Deploy test contract and write a passing JS test with `navigator.credentials.create` against new smart contract

