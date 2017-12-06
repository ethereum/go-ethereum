import os,sys, subprocess
from tinyrpc.transports import ServerTransport
from tinyrpc.protocols.jsonrpc import JSONRPCProtocol
from tinyrpc.dispatch import RPCDispatcher
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
        print("IN  ->\n{}".format( data))
        return None, urlparse.unquote(data)

    def send_reply(self, context, reply):
        print("OUT <-\n{}".format( reply))
        self.output.write(reply)
        self.output.write("\n")

dispatcher = RPCDispatcher()

@dispatcher.public
def ApproveTx(Transaction = None, From = None, Callinfo = None, Meta = None):
    return {
        "Approved" : True,
        "Transaction" : Transaction,
        "From" : From,
        "Password" : None,
    }

@dispatcher.public
def ApproveSignData():
    return {"Approved": False,
            "Password" : None}

@dispatcher.public
def ApproveExport():
    return {"Approved" : False}

@dispatcher.public
def ApproveImport():
    return {"Approved" : False, "OldPassword": "", "NewPassword": ""}

@dispatcher.public
def ApproveListing():
    return []

@dispatcher.public
def ApproveNewAccount():
    return {"Approved": False, "Password": ""}

@dispatcher.public
def ShowError(text = ""):
    sys.err.println("Error: %s", text)
    return

@dispatcher.public
def ShowInfo(text = ""):
    sys.err.println("Info: %s", text)
    return


def main():
    # line buffered
    p = subprocess.Popen(["./signer", "--stdio-ui"], bufsize=1, universal_newlines=True, stdin=subprocess.PIPE, stdout=subprocess.PIPE)
    transport = PipeTransport(p.stdout, p.stdin)
    rpc_server = RPCServer(
        transport,
        JSONRPCProtocol(),
        dispatcher
    )
    rpc_server.serve_forever()

if __name__ == '__main__':
    main()