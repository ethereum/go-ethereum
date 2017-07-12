/* Demonstrates the following signer methods

- account_new: generate new password protected account
    1. password: string

    returns account object with address and URL

- account_list: listing of accounts
    no args

    returns array with accounts

- account_signTransaction: sign transaction and get tx in RLP encoded form back
    1. from: address
    2. passwd: string
    3. transaction: object

    returns signed transaction in RLP form (can be used with eth_sendRawTransaction)

- account_sign: calculate signature
    1. from: address
    2. passwd: string
    3. data: hex string

    returns signature

- account_ecRecover: derive address from signature
    1. data: hex string
    2. signature: hex string

    returns address
 */

var spawn = require('child_process').spawn;

// by default the signer used the keystore for the mainnet, in this case it is pointed to a non-standard location.
// also it accepts the chainid, by default it uses the chainid for the mainnet.
const signer = spawn('./signer', ['-keystore', '/tmp/keystore', '-chainid', 5]);
const passwd = 'my password';
var createdAccountAddress = '0x';
var signData = '0xaabbccdd';
var signSignature = '0x';
var keystoreKeyData = '0x';

var currentRequest = -1;
function nextRequest() {
    currentRequest++;
    var req = null;

    if (currentRequest < 7) {
        req = {
            id: currentRequest,
            jsonrpc: "2.0"
        };

        switch (currentRequest) {
            case 0:
                req.method = 'account_new';
                req.params = [passwd];
                break
            case 1:
                req.method = 'account_list';
                break;
            case 2:
                req.method = 'account_signTransaction';
                req.params = [createdAccountAddress, passwd, {
                    nonce: "0x0",
                    gasPrice: "0x1234",
                    gas: "0x55555",
                    value: "0x1234",
                    input: "0xabcd",
                    to: "0x07a565b7ed7d7a678680a4c162885bedbb695fe0"
                }];
                break;
            case 3:
                req.method = 'account_sign';
                req.params = [createdAccountAddress, passwd, signData];
                break;
            case 4:
                req.method = 'account_ecRecover';
                req.params = [signData, signSignature];
                break;
            case 5:
                req.method = 'account_export';
                req.params = [createdAccountAddress]
                break;
            case 6:
                req.method = 'account_import';
                req.params = [keystoreKeyData, passwd, passwd];
                break;
        }
    }

    return req;
}
signer.stdout.on('data', (data) => {
    console.log(`${data}`);

    response = JSON.parse(`${data}`);

    switch (response.id) {
        case 0:
            createdAccountAddress = response.result.address;
            break;
        case 3:
            signSignature = response.result;
            break
        case 4:
            if (createdAccountAddress !== response.result) {
                console.error("expected address", createdAccountAddress, "got", response.result);
            } else {
                //console.log("Address recovered correct");
            }
            break;
        case 5:
            keystoreKeyData = response.result;
            break;
    }

    var req = nextRequest();
    if (req !== null) {
        req = JSON.stringify(req);
        console.log(req);
        signer.stdin.write(req);
    } else {
        signer.kill();
    }
});

signer.stderr.on('data', (data) => {
    console.log(`stderr: ${data}`);
});

signer.on('close', (code) => {
    //console.log(`signer process exited with code ${code}`);
});

// kickstart request cycle
req = JSON.stringify(nextRequest());
console.log(req);
signer.stdin.write(req);
