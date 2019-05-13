## Initializing the signer

First, initialize the master seed.

```text
#./signer init

WARNING!

The signer is alpha software, and not yet publically released. This software has _not_ been audited, and there
are no guarantees about the workings of this software. It may contain severe flaws. You should not use this software
unless you agree to take full responsibility for doing so, and know what you are doing.

TLDR; THIS IS NOT PRODUCTION-READY SOFTWARE!


Enter 'ok' to proceed:
>ok
A master seed has been generated into /home/martin/.signer/secrets.dat

This is required to be able to store credentials, such as :
* Passwords for keystores (used by rule engine)
* Storage for javascript rules
* Hash of rule-file

You should treat that file with utmost secrecy, and make a backup of it.
NOTE: This file does not contain your accounts. Those need to be backed up separately!
```

(for readability purposes, we'll remove the WARNING printout in the rest of this document)

## Creating rules

Now, you can create a rule-file. Note that it is not mandatory to use predefined rules, but it's really handy.

```javascript
function ApproveListing(){
    return "Approve"
}
```

Get the `sha256` hash. If you have openssl, you can do `openssl sha256 rules.js`...
```text
#sha256sum rules.js
6c21d1737429d6d4f2e55146da0797782f3c0a0355227f19d702df377c165d72  rules.js
```
...now `attest` the file...
```text
#./signer attest 6c21d1737429d6d4f2e55146da0797782f3c0a0355227f19d702df377c165d72

INFO [02-21|12:14:38] Ruleset attestation updated              sha256=6c21d1737429d6d4f2e55146da0797782f3c0a0355227f19d702df377c165d72
```

...and (this is required only for non-production versions) load a mock-up `4byte.json` by copying the file from the source to your current working directory:
```text
#cp $GOPATH/src/github.com/ethereum/go-ethereum/cmd/clef/4byte.json $PWD
```

At this point, we can start the signer with the rule-file:
```text
#./signer --rules rules.js --rpc

INFO [09-25|20:28:11.866] Using CLI as UI-channel 
INFO [09-25|20:28:11.876] Loaded 4byte db                          signatures=5509 file=./4byte.json
INFO [09-25|20:28:11.877] Rule engine configured                   file=./rules.js
DEBUG[09-25|20:28:11.877] FS scan times                            list=100.781µs set=13.253µs diff=5.761µs
DEBUG[09-25|20:28:11.884] Ledger support enabled 
DEBUG[09-25|20:28:11.888] Trezor support enabled 
INFO [09-25|20:28:11.888] Audit logs configured                    file=audit.log
DEBUG[09-25|20:28:11.888] HTTP registered                          namespace=account
INFO [09-25|20:28:11.890] HTTP endpoint opened                     url=http://localhost:8550
DEBUG[09-25|20:28:11.890] IPC registered                           namespace=account
INFO [09-25|20:28:11.890] IPC endpoint opened                      url=<nil>
------- Signer info -------
* extapi_version : 2.0.0
* intapi_version : 2.0.0
* extapi_http : http://localhost:8550
* extapi_ipc : <nil>
```

Any list-requests will now be auto-approved by our rule-file.

## Under the hood

While doing the operations above, these files have been created:

```text
#ls -laR ~/.signer/
/home/martin/.signer/:
total 16
drwx------  3 martin martin 4096 feb 21 12:14 .
drwxr-xr-x 71 martin martin 4096 feb 21 12:12 ..
drwx------  2 martin martin 4096 feb 21 12:14 43f73718397aa54d1b22
-rwx------  1 martin martin  256 feb 21 12:12 secrets.dat

/home/martin/.signer/43f73718397aa54d1b22:
total 12
drwx------ 2 martin martin 4096 feb 21 12:14 .
drwx------ 3 martin martin 4096 feb 21 12:14 ..
-rw------- 1 martin martin  159 feb 21 12:14 config.json

#cat /home/martin/.signer/43f73718397aa54d1b22/config.json
{"ruleset_sha256":{"iv":"6v4W4tfJxj3zZFbl","c":"6dt5RTDiTq93yh1qDEjpsat/tsKG7cb+vr3sza26IPL2fvsQ6ZoqFx++CPUa8yy6fD9Bbq41L01ehkKHTG3pOAeqTW6zc/+t0wv3AB6xPmU="}}

```

In `~/.signer`, the `secrets.dat` file was created, containing the `master_seed`.
The `master_seed` was then used to derive a few other things:

- `vault_location` : in this case `43f73718397aa54d1b22` .
   - Thus, if you use a different `master_seed`, another `vault_location` will be used that does not conflict with each other.
   - Example: `signer --signersecret /path/to/afile ...`
- `config.json` which is the encrypted key/value storage for configuration data, containing the key `ruleset_sha256`.


## Adding credentials

In order to make more useful rules like signing transactions, the signer needs access to the passwords needed to unlock keystores.

```text
#./signer addpw "0x694267f14675d7e1b9494fd8d72fefe1755710fa" "test_password"

INFO [02-21|13:43:21] Credential store updated                 key=0x694267f14675d7e1b9494fd8d72fefe1755710fa
```
## More advanced rules

Now let's update the rules to make use of credentials:

```javascript
function ApproveListing(){
    return "Approve"
}
function ApproveSignData(r){
    if( r.address.toLowerCase() == "0x694267f14675d7e1b9494fd8d72fefe1755710fa")
    {
        if(r.message.indexOf("bazonk") >= 0){
            return "Approve"
        }
        return "Reject"
    }
    // Otherwise goes to manual processing
}

```
In this example:
* Any requests to sign data with the account `0x694...` will be
    * auto-approved if the message contains with `bazonk`
    * auto-rejected if it does not.
* Any other signing-requests will be passed along for manual approve/reject.

_Note: make sure that `0x694...` is an account you have access to. You can create it either via the clef or the traditional account cli tool. If the latter was chosen, make sure both clef and geth use the same keystore by specifing `--keystore path/to/your/keystore` when running clef._

Attest the new file...
```text
#sha256sum rules.js
2a0cb661dacfc804b6e95d935d813fd17c0997a7170e4092ffbc34ca976acd9f  rules.js

#./signer attest 2a0cb661dacfc804b6e95d935d813fd17c0997a7170e4092ffbc34ca976acd9f

INFO [02-21|14:36:30] Ruleset attestation updated              sha256=2a0cb661dacfc804b6e95d935d813fd17c0997a7170e4092ffbc34ca976acd9f
```

And start the signer:

```
#./signer --rules rules.js --rpc

INFO [09-25|21:02:16.450] Using CLI as UI-channel 
INFO [09-25|21:02:16.466] Loaded 4byte db                          signatures=5509 file=./4byte.json
INFO [09-25|21:02:16.467] Rule engine configured                   file=./rules.js
DEBUG[09-25|21:02:16.468] FS scan times                            list=1.45262ms set=21.926µs diff=6.944µs
DEBUG[09-25|21:02:16.473] Ledger support enabled 
DEBUG[09-25|21:02:16.475] Trezor support enabled 
INFO [09-25|21:02:16.476] Audit logs configured                    file=audit.log
DEBUG[09-25|21:02:16.476] HTTP registered                          namespace=account
INFO [09-25|21:02:16.478] HTTP endpoint opened                     url=http://localhost:8550
DEBUG[09-25|21:02:16.478] IPC registered                           namespace=account
INFO [09-25|21:02:16.478] IPC endpoint opened                      url=<nil>
------- Signer info -------
* extapi_version : 2.0.0
* intapi_version : 2.0.0
* extapi_http : http://localhost:8550
* extapi_ipc : <nil>
```

And then test signing, once with `bazonk` and once without:

```
#curl -H "Content-Type: application/json" -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"account_sign\",\"params\":[\"0x694267f14675d7e1b9494fd8d72fefe1755710fa\",\"0x$(xxd -pu <<< '  bazonk baz gaz')\"],\"id\":67}" http://localhost:8550/
{"jsonrpc":"2.0","id":67,"result":"0x93e6161840c3ae1efc26dc68dedab6e8fc233bb3fefa1b4645dbf6609b93dace160572ea4ab33240256bb6d3dadb60dcd9c515d6374d3cf614ee897408d41d541c"}

#curl -H "Content-Type: application/json" -X POST --data "{\"jsonrpc\":\"2.0\",\"method\":\"account_sign\",\"params\":[\"0x694267f14675d7e1b9494fd8d72fefe1755710fa\",\"0x$(xxd -pu <<< '  bonk baz gaz')\"],\"id\":67}" http://localhost:8550/
{"jsonrpc":"2.0","id":67,"error":{"code":-32000,"message":"Request denied"}}

```

Meanwhile, in the signer output:
```text
INFO [02-21|14:42:41] Op approved
INFO [02-21|14:42:56] Op rejected
```

The signer also stores all traffic over the external API in a log file. The last 4 lines shows the two requests and their responses:

```text
#tail -n 4 audit.log
t=2018-02-21T14:42:41+0100 lvl=info msg=Sign       api=signer type=request  metadata="{\"remote\":\"127.0.0.1:49706\",\"local\":\"localhost:8550\",\"scheme\":\"HTTP/1.1\"}" addr="0x694267f14675d7e1b9494fd8d72fefe1755710fa [chksum INVALID]" data=202062617a6f6e6b2062617a2067617a0a
t=2018-02-21T14:42:42+0100 lvl=info msg=Sign       api=signer type=response data=93e6161840c3ae1efc26dc68dedab6e8fc233bb3fefa1b4645dbf6609b93dace160572ea4ab33240256bb6d3dadb60dcd9c515d6374d3cf614ee897408d41d541c error=nil
t=2018-02-21T14:42:56+0100 lvl=info msg=Sign       api=signer type=request  metadata="{\"remote\":\"127.0.0.1:49708\",\"local\":\"localhost:8550\",\"scheme\":\"HTTP/1.1\"}" addr="0x694267f14675d7e1b9494fd8d72fefe1755710fa [chksum INVALID]" data=2020626f6e6b2062617a2067617a0a
t=2018-02-21T14:42:56+0100 lvl=info msg=Sign       api=signer type=response data=                                                                                                                                   error="Request denied"
```