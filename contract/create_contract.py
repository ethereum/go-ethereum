import sys
import json
import urllib2, urllib
from eth_rpc_client import Client


def test_connectivity(ip):

    client = Client(host=ip, port="54501")
    coinbase = client.get_coinbase()
    if coinbase: return coinbase
    return False


def get_balance(addr):
    balance = Client.get_balance(addr, block="latest")
    if balance: return balance
    return False


def get_compilers(addr, rpcport):

    postdata = '{"jsonrpc":"2.0","method":"eth_getCompilers","params":[],"id":1}'
    service_url = "http://%s:%s" % (addr, rpcport)
    req = urllib2.Request(service_url, postdata)

    try:
        handle = urllib2.urlopen(req)
        json_response = handle.read()
        res = json.loads(json_response)
    except Exception as e:
        print e
        return False

    if res and res['result'][0] == "Solidity":
        return True
    return False

    

if __name__ == "__main__":
    if len(sys.argv) > 1:
        addr = test_connectivity(sys.argv[1])
        if addr > 0:
            if get_compilers("127.0.0.1", "54501"):
                print "Compiler exists."
            else:
                print "Compiler not loaded"
                sys.exit(0)
        
    else:
        print "Specify ip-address of RPC-server as first argument."
        sys.exit()
