
# Config to run Retesteth using docker

- git submodule add --depth 1 https://github.com/ethereum/tests.git tests/testdata  
- download docker image from http://retesteth.ethdevops.io/  
- load the docker image with `sudo docker load -i dretest*.tar` 
- Download the `dretesteth.sh` script with
```
  wget https://raw.githubusercontent.com/ethereum/retesteth/master/dretesteth.sh
  chmod +x dretesteth.sh
```
- Run a test with the following command. It will create the retesteth configuration directories in `~/tests/config`, and run a sanity check  
```
sudo ./dretesteth.sh -t GeneralStateTests/stExample -- \
  --testpath ~/tests --datadir /tests/config
```
- The output should be similar to
```
Running 1 test case...
Running tests using path: /tests
Active client configurations: 't8ntool '
Running tests for config 'Ethereum GO on StateTool'
Test Case "stExample":
100%
*** No errors detected
*** Total Tests Run: 1
```

# Testing against bor

- Change configs by replacing geth with bor inside the docker container  
```
mkdir ~/retestethBuild
cd ~/retestethBuild
wget https://raw.githubusercontent.com/ethereum/retesteth/develop/dretesteth.sh
chmod +x dretesteth.sh
wget https://raw.githubusercontent.com/ethereum/retesteth/develop/Dockerfile
```
Modify the RUN git clone line in the Dockerfile for repo “retesteth” to change branch -b from master to develop. Do not modify repo branches for “winsvega/solidity” [LLLC opcode support] and “go-ethereum”.
Modify the Dockerfile so that the eth client points to bor  
e.g. : `https://github.com/ethereum/retesteth/blob/master/Dockerfile#L41`
from `RUN git clone --depth 1 -b master https://github.com/ethereum/go-ethereum.git /geth`
to: `RUN git clone --depth 1 -b master https://github.com/maticnetwork/bor.git /geth`

- build docker image
`sudo ./dretesteth.sh build`

- clone repo
``` 
git clone --branch develop https://github.com/ethereum/tests.git
```
this step will be eventually replaced by adding the git submodule directly into bor repo with   
``` 
git submodule add --depth 1 https://github.com/ethereum/tests.git tests/testdata
```
- Let's move to the restestethBuild folder
```
cd /home/ubuntu/retestethBuild
```
Now we have the tests repo here  
```
ls
> Dockerfile  dretesteth.sh  tests
```
- Run test example    
```
./dretesteth.sh -t GeneralStateTests/stExample --  --testpath /home/ubuntu/retestethBuild/tests --datadir /tests/config
```
This will create the config files for the different clients in ~/tests/config
Eventually. these config needs to be adapted according to the following doc  
https://ethereum-tests.readthedocs.io/en/latest/retesteth-tutorial.html
Specifically:  
``` 
f you look inside ~/tests/config, you’ll see a directory for each configured client. Typically this directory has these files:

config, which contains the configuration for the client:
The communication protocol to use with the client (typically TCP)
The address(es) to use with that protocol
The forks the client supports
The exceptions the client can throw, and how retesteth should interpret them. This is particularly important when testing the client’s behavior when given invalid blocks.
start.sh, which starts the client inside the docker image
stop.sh, which stops the client instance(s)
genesis, a directory which includes the genesis blocks for various forks the client supports. If this directory does not exist for a client, it uses the genesis blocks for the default client.
```

We replaced geth inside docker by using https://ethereum-tests.readthedocs.io/en/latest/retesteth-tutorial.html#replace-geth-inside-the-docker  
Theoretically, we would not need any additional config change  

- Run test suites    
``` 
./dretesteth.sh -t <TestSuiteName> --  --testpath /home/ubuntu/retestethBuild/tests --datadir /tests/config
```
Where `TestSuiteName` is one of the maintained test suites, reported here https://github.com/ethereum/tests  
```
BasicTests
BlockchainTests
GeneralStateTests
TransactionTests
RLPTest
src
```
