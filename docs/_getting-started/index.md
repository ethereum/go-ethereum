---
title: Getting Started with Geth
permalink: docs/getting-started
sort_key: A
---

To use Geth, you need to install it first. You can install Geth in a variety of ways that you can find in the “Install and Build” section. 
These include installing it via your favorite package manager, downloading a standalone pre-built binary, running it as a docker container, or building it yourself.

For this guide, we assume you have Geth installed and are ready to find out how to use it. 
The guide shows you how to create accounts, sync to a network, and then send transactions between accounts.
This guide uses Clef, which is our preferred tool for signing transactions with Geth, and will replace Geth’s account management.

### Two Important terms in geth:

- Network
- Sync modes

Networks
You can connect a Geth node to several different networks using the network name as an argument. These include the main Ethereum network, a private network you create, and three test networks that use different consensus algorithms:

-   Ropsten: Proof-of-work test network
-   Rinkeby: Proof-of-authority test network
-   Görli: Proof-of-authority test network

For this guide, you will use the Görli network. The default port is 8545, so you need to enable at least outgoing access from your node to that port.
Sync modes
You can start Geth in one of three different sync modes using the --syncmode "<mode>" argument that determines what sort of node it is in the network.
These are:

- Full: Downloads all blocks (including headers, transactions, and receipts) and generates the state of the blockchain incrementally by executing every block.
- Snap (Default): Same functionality as fast, but with a faster algorithm.
- Light: Downloads all block headers, block data, and verifies some randomly.

For this tutorial, you will use a light sync:

### Prerequisites:

- Curl experience 
- Command line
- Basic Blockchain Knowledge

## Step 1: Open Terminal
You will need your system terminal to run the commands for this tutorial.
![Clef init command](https://paper.dropbox.com/ep/redirect/image?url=https%3A%2F%2Fpaper-attachments.dropbox.com%2Fs_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644261248020_Screenshot%2B2022-02-07%2Bat%2B20.10.06.png&hmac=u78AuCP9io2DlhLyay%2BRQioFXVcp9%2BOGq4OcX8OqacM%3D){:width="70%"}

### Steps to open your system terminal:
**Mac book:**
Press the Command + Space button on your keyboard to open spotlight search, type terminal, and hit return.

**Windows:**
Type cmd in the search box, press Enter to open the Command Prompt shortcut highlighted, and then hit the enter button. 

**Linux:**
To quickly open a Terminal window at any time, press Ctrl+Alt+T.

## Step 2: Create accounts

use the command below to create an account 
Note: you will need to create two accounts for this guide

```shell
clef newaccount --keystore geth-tutorial/keystore
```

it will give you the result below:

![Create new account command](https://paper.dropbox.com/ep/redirect/image?url=https%3A%2F%2Fpaper-attachments.dropbox.com%2Fs_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644255368476_Screenshot%2B2022-02-07%2Bat%2B18.12.41.png&hmac=Gv1oXwU4YBtCatEb2XL5Pu%2F%2Bp1%2F025nqncAgEizTF5U%3D){:width="70%"}

Enter “ok” and hit the enter key. Next, the system will request the below action.
Please enter a password for the new account to be created (attempt 0 of 3)
Enter your desired password and hit the enter key to get the result below:

![Create new account command](https://paper.dropbox.com/ep/redirect/image?url=https%3A%2F%2Fpaper-attachments.dropbox.com%2Fs_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644255433564_Screenshot%2B2022-02-07%2Bat%2B18.26.37.png&hmac=eeLGC9zJ7L%2B9q%2B1%2B8JZtV%2BIqN2%2FZ9pwfWWCgkGq5AsI%3D&width=1490){:width="70%"}

Copy and save your password and generated account somewhere safe; you will need it later in this tutorial.
**The Generated account:**
```shell
0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC
```
## Step 3:  Start Clef

To start clef, open a new terminal and run the command below. It would be best if you did not close this terminal, always keep it running while working.

```shell
clef --keystore geth-tutorial/keystore --configdir geth-tutorial/clef --chainid 5
```

Note:  geth-tutorial folder is the directory holding your keystore

after running the command above, the system will request you to type “ok” to proceed

A successful call will give you the result below:

![Create new account command](https://paper.dropbox.com/ep/redirect/image?url=https%3A%2F%2Fpaper-attachments.dropbox.com%2Fs_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644272658382_Screenshot%2B2022-02-07%2Bat%2B23.22.20.png&hmac=Qzk34Kp9ClJQf2hAEdKNlO8FWd2EaK3J18rPb66bGe8%3D&width=1490){:width="70%"}

Note: keep this terminal open.
## Step 4:  Start Geth
To start geth, open a new terminal and run the command below. It would be best if you did not close this terminal, always keep it running while working.

```shell
geth --datadir geth-tutorial --signer=geth-tutorial/clef/clef.ipc --goerli --syncmode "light" --http
```


A successful call will give you the result below:

![Create new account command](https://paper.dropbox.com/ep/redirect/image?url=https%3A%2F%2Fpaper-attachments.dropbox.com%2Fs_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644272805150_Screenshot%2B2022-02-07%2Bat%2B23.26.25.png&hmac=FJHQSqMQzJih7TP3jC7qDwdaRtzE0o4dYQkNmwpH2yU%3D&width=1490){:width="70%"}

Note: keep this terminal open.



## Step 5:  Get Free Goerli Faucets

A crypto faucet is a web application that rewards you with small amounts of cryptocurrencies for completing easy tasks. The reward is small, that's why it is called faucets. The primary purpose of the faucet is to fund your testnet account to pay for gas fees for testing your project. 
The following sites gives free goerli faucets:

- https://faucets.chain.link/goerli
- https://fauceth.komputing.org/?chain=5

## Step 6: Interact with Geth via IPC or RPC

You can interact with Geth in two ways: Directly with the node using the JavaScript console over IPC or connecting to the node remotely over HTTP using RPC.

- IPC (Inter-Process Communication):
    allows you to do more, especially when creating and interacting with accounts, but you need direct access to the node.
- RPC (Remote Procedure Call):
     allows remote applications to access your node but has limitations and security considerations, and by default, only allows access to methods in the eth and shh namespaces. Find out how to override this setting in the RPC docs.

## Step 7: Using IPC

**→ Connect to console**
Connect to the IPC console on a node from another terminal window, this will open the Geth javascript console
run the command below

```shell
geth attach http://127.0.0.1:8545
```

Result after running the above command: 

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644274703969_Screenshot+2022-02-07+at+23.57.11.png){:width="70%"}


**→ Check account balance**

Note: the value comes in wei
**Syntax:**

```shell
web3.fromWei(eth.getBalance("<ADDRESS_1>"),"ether")
````

run the command below to check your account balance

```
web3.fromWei(eth.getBalance("0xca57F3b40B42FCce3c37B8D18aDBca5260ca72EC"),"ether")
```

**Result:**

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644276258069_Screenshot+2022-02-08+at+00.22.55.png){:width="70%"}


**→ Check list of accounts**
**step 1:**
Run the command below to get the list of accounts in your keystore
 eth.accounts
**step 2:**
Accept request in your Clef terminal 
the command in step 1 will need approval from the terminal running clef, before showing the list of accounts.

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644276747708_Screenshot+2022-02-08+at+00.31.12.png){:width="70%"}


approve the request by typing “y” and hit the enter key.

**Result:**

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644276986503_Screenshot+2022-02-08+at+00.31.12.png){:width="70%"}


**→ Send ETH to account**

Send 0.01 ETH from the account that you added ETH to with the Görli faucet, to the second account you created.
**Syntax:**

```shell
eth.sendTransaction({from:"<ADDRESS_1>",to:"<ADDRESS_2>", value: web3.toWei(0.01,"ether")})
```

**step 1:** 
run the command below to transfer 0.01ether to the other account you created
```javscript
eth.sendTransaction({from:"0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec",to:"0x8EB19d8DF81a8B43a178207E23E9a57ff8cA61B1", value: web3.toWei(0.01,"ether")})
```

**step 2:**
accept request in your Clef terminal 
The command in step 1 will need approval from the terminal running clef, Clef will prompt you to approve the transaction, and when you do, it will ask you for the password for the account you are sending the ETH from. If the password is correct, Geth proceeds with the transaction.


![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644278525355_Screenshot+2022-02-08+at+01.01.18.png){:width="70%"}


After approving the transaction you will see the below screen in the Clef terminal

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644278775275_Screenshot+2022-02-08+at+01.06.05.png){:width="70%"}


**Step 1** Terminal Result, it will return a response that includes the transaction hash:

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644279037329_Screenshot+2022-02-08+at+01.09.45.png){:width="70%"}


**→ Check Transaction hash**

**Syntax:**

```shell
eth.getTransaction("hash id")
```

A Transaction Hash (Tx Hash) is a record of successful transaction in a blockchain that can be accessed with unique address.

Run the command below.

```javscript
eth.getTransaction("0xa2b547d8742e345fa5f86f017d9da38c4a19cacee91e85191a57c0c7e420d187")
````

if successful, you will get the below response 

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644290184662_Screenshot+2022-02-08+at+04.15.18.png){:width="70%"}


## Step 7: Using RPC

**→ Check account balance**

**Syntax:**

```shell
    curl -X POST http://<GETH_IP_ADDRESS>:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["<ADDRESS_1>","latest"], "id":1}'
```
 
 Note: http://127.0.0.1:8545 this is the default address
 
 To check your account balance use the command below.

 ```shell
  curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_getBalance", "params":["0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","latest"], "id":5}'
   ```

A successful call will return a response below:

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644280003369_Screenshot+2022-02-08+at+01.26.32.png){:width="70%"}


so Geth returns the value without invoking Clef. Note that the value returned is in hexadecimal and WEI. To get the ETH value, convert to decimal and divide by 10^18.

**→ Check list of accounts**

Run the command below to get all the accounts.
```shell
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_accounts","params":[], "id":5}'
```

Follow the same step as the IPC Check account balance

A successful call will return a response below:

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644280927308_Screenshot+2022-02-08+at+01.28.14.png){:width="70%"}


**→ Send ETH to accounts**

**Syntax:**
```shell
    curl -X POST http://<GETH_IP_ADDRESS>:8545 \
        -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "<ADDRESS_1>","to": "<ADDRESS_2>","value": "0x9184e72a"}], "id":1}'
```

**step 1:** convert value from Eth to Wei decimal
Use this link to do the conversation: https://eth-converter.com/

**step 2:** convert decimal to Hexadecimal
Use this link to do the conversion: https://www.rapidtables.com/convert/number/decimal-to-hex.html and add 0x at the begging of the number.
Example:  from 2386F26FC10000 to 0x2386F26FC10000

**step 3:** Run the command below
```shell
curl -X POST http://127.0.0.1:8545 \
    -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0", "method":"eth_sendTransaction", "params":[{"from": "0xca57f3b40b42fcce3c37b8d18adbca5260ca72ec","to": "0x1f7a76611939fbAcf7d2dAD2F864F6184BDCD690","value": "0x2386F26FC10000"}], "id":5}'
```

A successful call will return a response below:

![](https://paper-attachments.dropbox.com/s_B9DD796393E608BD6B8358DDFCBFEB5B6F1555AA272CA011CE94B8B98D38751D_1644288977377_Screenshot+2022-02-08+at+02.53.21.png){:width="70%"}

