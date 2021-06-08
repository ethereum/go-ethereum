from web3 import Web3
import sys
#import socket
#import random
#import json
#import rlp
#import time
#import numpy as np
import os, binascii
from datetime import datetime
from multiprocessing import Pool

# Settings
FULL_PORT = "8081"
PASSWORD = "1234"

# Account number
ACCOUNT_NUM = int(sys.argv[1])
TX_PER_BLOCK = 200

# multiprocessing
THREAD_COUNT = 8

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

    # main loop for send txs
    print("start sending transactions")
    txNums = [TX_PER_BLOCK/THREAD_COUNT]*THREAD_COUNT
    txNums[0] += TX_PER_BLOCK%THREAD_COUNT
    for i in range(int(ACCOUNT_NUM / TX_PER_BLOCK)):

        # send transactions
        sendPool.map(sendTransactions, txNums)
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



def sendTransactions(num):
    for i in range(int(num)):
        while True:
            try:
                to = makeRandHex()
                fullnode.eth.sendTransaction(
                    {'to': to, 'from': fullnode.eth.coinbase, 'value': '1', 'gas': '21000', 'data': ""})
                break
            except:
                continue



def makeRandHex():
	randHex = binascii.b2a_hex(os.urandom(20))
	return Web3.toChecksumAddress("0x" + randHex.decode('utf-8'))



if __name__ == "__main__":

    totalStartTime = datetime.now()
    sendPool = Pool(THREAD_COUNT) # -> important: this should be in this "__main__" function
    main()
    totalEndTime = datetime.now() - totalStartTime
    print("total elapsed:", totalEndTime.seconds, "seconds")
    print("DONE")
