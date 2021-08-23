from web3 import Web3
import sys
#import socket
import random
#import json
#import rlp
import time
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
THREAD_COUNT = 1

# tx arguments option
INCREMENTAL_RECEIVER_ADDRESS = True # set tx receiver: incremental vs random
INCREMENTAL_SEND_AMOUNT = True      # set send amount: incremental vs same (1 wei)
MAX_ADDRESS = 0                     # set max address to set the receiver address upper bound (0 means there is no bound)

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

    # stop mining
    fullnode.geth.miner.stop()

    # get current block
    currentBlock = fullnode.eth.blockNumber

    # main loop for send txs
    print("start sending transactions")
    offset = 1
    txNums = [int(TX_PER_BLOCK/THREAD_COUNT)]*THREAD_COUNT
    txNums[0] += TX_PER_BLOCK%THREAD_COUNT
    for i in range(int(ACCOUNT_NUM / TX_PER_BLOCK)):
        # set arguments for multithreading function
        arguments = []
        for j in range(THREAD_COUNT):
            arguments.append((txNums[j], offset))
            offset += txNums[j]

        # send transactions
        sendPool.starmap(sendTransactions, arguments)
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



def sendTransactions(num, offset):
    for i in range(int(num)):
        # set receiver
        if INCREMENTAL_RECEIVER_ADDRESS:
            to = intToAddr(int(offset+i))
        else:
            to = makeRandHex()

        # if the upper bound is set, select receiver within the bound
        if MAX_ADDRESS != 0:
            to = intToAddr(random.randint(1, MAX_ADDRESS))
        
        # set send amount
        if INCREMENTAL_SEND_AMOUNT:
            amount = int(offset+i)
        else:
            amount = int(1)

        # print("to: ", to, "/ from: ", fullnode.eth.coinbase, "/ amount:", amount)

        while True:
            try:
                fullnode.eth.sendTransaction(
                    {'to': to, 'from': fullnode.eth.coinbase, 'value': hex(amount), 'gas': '21000', 'data': ""})
                break
            except:
                time.sleep(1)
                continue



def makeRandHex():
	randHex = binascii.b2a_hex(os.urandom(20))
	return Web3.toChecksumAddress("0x" + randHex.decode('utf-8'))



def intToAddr(num):
    intToHex = f'{num:0>40x}'
    return Web3.toChecksumAddress("0x" + intToHex)



if __name__ == "__main__":

    totalStartTime = datetime.now()
    sendPool = Pool(THREAD_COUNT) # -> important: this should be in this "__main__" function
    main()
    totalEndTime = datetime.now() - totalStartTime
    print("total elapsed:", totalEndTime.seconds, "seconds")
    print("DONE")
