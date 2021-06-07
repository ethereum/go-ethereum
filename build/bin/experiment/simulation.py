from web3 import Web3
import sys
#import socket
#import random
#import json
#import rlp
#import time
#import binascii
#import numpy as np
import os,binascii

# Settings
FULL_PORT = "8081"
PASSWORD = "1234"

# Account number
ACCOUNT_NUM = int(sys.argv[1])
TX_PER_BLOCK = 10

# providers
fullnode = Web3(Web3.HTTPProvider("http://localhost:" + FULL_PORT))

# functions
def main():
    
    if ACCOUNT_NUM < TX_PER_BLOCK:
        print("too less accounts. at least", TX_PER_BLOCK, "accounts are needed")
        return

    print("Insert ", ACCOUNT_NUM, " accounts")

    # unlock coinbase
    fullnode.geth.personal.unlockAccount(fullnode.eth.coinbase, PASSWORD, 0)

    # get current block
    currentBlock = fullnode.eth.blockNumber

    print("start sending transactions")
    # main loop for send txs
    for i in range(int(ACCOUNT_NUM / TX_PER_BLOCK)):

        # send transactions
        for j in range(TX_PER_BLOCK):
            to = makeRandHex()
            sendTransaction(to)
            #print("Send Tx# {0}".format(j), end="\r")
        
        print("inserted ", (i+1)*TX_PER_BLOCK, "accounts")

        # mining
        fullnode.geth.miner.start(1)  # start mining
        while (fullnode.eth.blockNumber == currentBlock):
            pass # just wait for mining
        fullnode.geth.miner.stop()  # stop mining
        currentBlock = fullnode.eth.blockNumber




def sendTransaction(to):
    #print("start try to send tx to full node")
    #print("to: ", to, "/ from: ", fullnode.eth.coinbase)
    while True:
        try:
            fullnode.eth.sendTransaction(
                {'to': to, 'from': fullnode.eth.coinbase, 'value': '1', 'gas': '21000', 'data': ""})
            break
        except:
            continue



def makeRandHex():
	randHex = binascii.b2a_hex(os.urandom(20))
	return Web3.toChecksumAddress("0x" + randHex.decode('utf-8'))



if __name__ == "__main__":

    #print(Web3.toChecksumAddress(makeRandHex()))
    main()
    print("DONE")
