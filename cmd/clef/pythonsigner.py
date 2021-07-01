import os,sys, subprocess
from tinyrpc.transports import ServerTransport
from tinyrpc.protocols.jsonrpc import JSONRPCProtocol
from tinyrpc.dispatch import public,RPCDispatcher
from tinyrpc.server import RPCServer

""" This is a POC example of how to write a custom UI for Clef. The UI starts the
clef process with the '--stdio-ui' option, and communicates with clef using standard input / output.

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
    def ApproveTx(self,req):
        """
        Example request:
        {
            "jsonrpc": "2.0",
            "method": "ApproveTx",
            "params": [{
                "transaction": {
                    "to": "0xae967917c465db8578ca9024c205720b1a3651A9",
                    "gas": "0x333",
                    "gasPrice": "0x123",
                    "value": "0x10",
                    "data": "0xd7a5865800000000000000000000000000000000000000000000000000000000000000ff",
                    "nonce": "0x0"
                },
                "from": "0xAe967917c465db8578ca9024c205720b1a3651A9",
                "call_info": "Warning! Could not validate ABI-data against calldata\nSupplied ABI spec does not contain method signature in data: 0xd7a58658",
                "meta": {
                    "remote": "127.0.0.1:34572",
                    "local": "localhost:8550",
                    "scheme": "HTTP/1.1"
                }
            }],
            "id": 1
        }

        :param transaction: transaction info
        :param call_info: info abou the call, e.g. if ABI info could not be
        :param meta: metadata about the request, e.g. where the call comes from
        :return:
        """
        transaction = req.get('transaction')
        _from       = req.get('from')
        call_info   = req.get('call_info')
        meta        = req.get('meta')

        return {
            "approved" : False,
            #"transaction" : transaction,
  #          "from" : _from,
#            "password" : None,
        }

    @public
    def ApproveSignData(self, req):
        """ Example request

        """
        return {"approved": False, "password" : None}

    @public
    def ApproveExport(self, req):
        """ Example request

        """
        return {"approved" : False}

    @public
    def ApproveImport(self, req):
        """ Example request

        """
        return { "approved" : False, "old_password": "", "new_password": ""}

    @public
    def ApproveListing(self, req):
        """ Example request

        """
        return {'accounts': []}

    @public
    def ApproveNewAccount(self, req):
        """
        Example request

        :return:
        """
        return {"approved": False,
                #"password": ""
                }

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
    cmd = ["clef", "--stdio-ui"]
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
