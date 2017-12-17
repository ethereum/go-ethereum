import os,sys, subprocess
from tinyrpc.transports import ServerTransport
from tinyrpc.protocols.jsonrpc import JSONRPCProtocol
from tinyrpc.dispatch import public,RPCDispatcher
from tinyrpc.server import RPCServer

""" This is a POC example of how to write a custom UI for the signer. The UI starts the 
signer process with the '--stdio-ui' option, and communicates with the signer binary
using standard input / output.

The standard input/output is a relatively secure way to communicate, as it does not require opening any ports
or IPC files. Needless to say, it does not protect against memory inspection mechanisms where an attacker
can access process memory."""

try:
    import urllib.parse as urlparse
except ImportError:
    import urllib as urlparse

class StdIOTransport(ServerTransport):
    """ Uses std input/output for RPC """
    def receive_message(self):
        return None, urlparse.unquote(sys.stdin.readline())

    def send_reply(self, context, reply):
        print(reply)

class PipeTransport(ServerTransport):
    """ Uses std a pipe for RPC """

    def __init__(self,input, output):
        self.input = input
        self.output = output

    def receive_message(self):
        data = self.input.readline()
        print(">> {}".format( data))
        return None, urlparse.unquote(data)

    def send_reply(self, context, reply):
        print("<< {}".format( reply))
        self.output.write(reply)
        self.output.write("\n")

class StdIOHandler():

    def __init__(self):
        pass

    @public
    def ApproveTx(self,transaction = None, fromaccount = None, call_info = None, meta = None):
        """
        Example request:
        
        {"jsonrpc":"2.0","method":"ApproveTx","params":{"transaction":{"to":null,"gas":null,"gasPrice":null,"value":null,"data":"0x","nonce":null},"from":"0x0000000000000000000000000000000000000000","call_info":null,"meta":{"remote":"signer binary","local":"main","scheme":"in-proc"}},"id":2}

        :param transaction: transaction info
        :param call_info: info abou the call, e.g. if ABI info could not be
        :param meta: metadata about the request, e.g. where the call comes from
        :return: 
        """
        return {
            "approved" : False,
            "transaction" : None,
            #"fromaccount" : fromaccount,
            "password" : None,
        }

    @public
    def ApproveSignData(self,address=None, raw_data = None, message = None, hash = None, meta = None):
        """ Example request

        {"jsonrpc":"2.0","method":"ApproveSignData","params":{"address":"0x0000000000000000000000000000000000000000","raw_data":"0x01020304","message":"\u0019Ethereum Signed Message:\n4\u0001\u0002\u0003\u0004","hash":"0x7e3a4e7a9d1744bc5c675c25e1234ca8ed9162bd17f78b9085e48047c15ac310","meta":{"remote":"signer binary","local":"main","scheme":"in-proc"}},"id":3}


        """
        return {"approved": False,
                "password" : None}

    @public
    def ApproveExport(self,address = None, meta = None):
        """ Example request

        {"jsonrpc":"2.0","method":"ApproveExport","params":{"address":"0x0000000000000000000000000000000000000000","meta":{"remote":"signer binary","local":"main","scheme":"in-proc"}},"id":5}

        """
        return {"approved" : False}

    @public
    def ApproveImport(self,meta = None):
        """ Example request

        {"jsonrpc":"2.0","method":"ApproveImport","params":{"Meta":{}},"id":4}

        """
        return {"approved" : False, "old_password": "", "new_password": ""}

    @public
    def ApproveListing(self,accounts=None, meta = None):
        """ Example request

        {"jsonrpc":"2.0","method":"ApproveListing","params":{"accounts":[{"type":"Account","url":"keystore:///home/user/ethereum/keystore/file","address":"0x010101010101010010101010101abcdef0001337"}],"Meta":{}},"id":2}
        """
        return {'accounts': []}

    @public
    def ApproveNewAccount(self,meta = None):
        """
        Example request

        {"jsonrpc":"2.0","method":"ApproveNewAccount","params":{"meta":{"remote":"signer binary","local":"main","scheme":"in-proc"}},"id":5}

        :return:
        """
        return {"approved": False, "password": ""}

    @public
    def ShowError(self,message = {}):
        """
        Example request:

        {"jsonrpc":"2.0","method":"ShowInfo","params":{"message":"Testing 'ShowError'"},"id":1}

        :param message: to show
        :return: nothing
        """
        if 'text' in message.keys():
            sys.stderr.write("Error: {}\n".format( message['text']))
        return

    @public
    def ShowInfo(self,message = {}):
        """
        Example request
        {"jsonrpc":"2.0","method":"ShowInfo","params":{"message":"Testing 'ShowInfo'"},"id":0}

        :param message: to display
        :return:nothing
        """
        if 'text' in message.keys():
            sys.stdout.write("Error: {}\n".format( message['text']))
        return

def main(args):

    cmd = ["./signer", "--stdio-ui"]
    if len(args) > 0 and args[0] == "test":
        cmd.extend(["--stdio-ui-test"])
    print("cmd: {}".format(" ".join(cmd)))
    dispatcher = RPCDispatcher()
    dispatcher.register_instance(StdIOHandler(), '')
    # line buffered
    p = subprocess.Popen(cmd, bufsize=1, universal_newlines=True, stdin=subprocess.PIPE, stdout=subprocess.PIPE)

    rpc_server = RPCServer(
        PipeTransport(p.stdout, p.stdin),
        JSONRPCProtocol(),
        dispatcher
    )
    rpc_server.serve_forever()

if __name__ == '__main__':
    main(sys.argv[1:])
