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
        reply = str(reply, "utf-8")
        print("<< {}".format( reply))
        self.output.write("{}\n".format(reply))

def sanitize(txt, limit=100):
    return txt[:limit].encode("unicode_escape").decode("utf-8")

def metaString(meta):
    """
    "meta":{"remote":"clef binary","local":"main","scheme":"in-proc","User-Agent":"","Origin":""}
    """
    return """
    Request context:
	    {} -> {} -> {}
    Additional HTTP header data, provided by the external caller:
	    User-Agent: {}
	    Origin: {}
""".format( meta.get('remote', "<missing>"), meta.get('scheme','<missing>'), meta.get('local', '<missing>'),
	 sanitize(meta.get("User-Agent"), 200), sanitize(meta.get("Origin"),100))

class StdIOHandler():
    def __init__(self):
        pass

    @public
    def approveTx(self,req):
        """
        Example request:

        {"jsonrpc":"2.0","id":20,"method":"ui_approveTx","params":[{"transaction":{"from":"0xDEADbEeF000000000000000000000000DeaDbeEf","to":"0xDEADbEeF000000000000000000000000DeaDbeEf","gas":"0x3e8","gasPrice":"0x5","maxFeePerGas":null,"maxPriorityFeePerGas":null,"value":"0x6","nonce":"0x1","data":"0x"},"call_info":null,"meta":{"remote":"clef binary","local":"main","scheme":"in-proc","User-Agent":"","Origin":""}}]}

        :param transaction: transaction info
        :param call_info: info abou the call, e.g. if ABI info could not be
        :param meta: metadata about the request, e.g. where the call comes from
        :return:
        """
        transaction = req.get('transaction')
        _from       = transaction.get('from', "<missing>")
        to          = transaction.get('to', "<missing>")
        sys.stdout.write("""Sign transaction request:
    {}
    
    From: {}
    To: {}
    
    Auto-rejecting request
""".format(metaString(req.get('meta',{})), _from, to ))

        return {"approved" : False}

    @public
    def approveSignData(self, req):
        """ Example request

        {"jsonrpc":"2.0","id":8,"method":"ui_approveSignData","params":[{"content_type":"application/x-clique-header","address":"0x0011223344556677889900112233445566778899","raw_data":"+QIRoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAlAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAoAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAuQEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIIFOYIFOYIFOoIFOoIFOppFeHRyYSBkYXRhIEV4dHJhIGRhdGEgRXh0cqAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIgAAAAAAAAAAA==","messages":[{"name":"Clique header","value":"clique header 1337 [0x44381ab449d77774874aca34634cb53bc21bd22aef2d3d4cf40e51176cb585ec]","type":"clique"}],"call_info":null,"hash":"0xa47ab61438a12a06c81420e308c2b7aae44e9cd837a5df70dd021421c0f58643","meta":{"remote":"clef binary","local":"main","scheme":"in-proc","User-Agent":"","Origin":""}}]}
        """
        contentType = req.get('content_type')
        address = req.get('address')
        rawData = req.get('raw_data')
        contentType = req.get('content_type')
        hash = req.get('hash')
        meta = req.get('meta', {})
        sys.stdout.write("""Sign data request:
    {}
    
    Content-type: {}
    Address: {}
    Hash: {}

    Auto-rejecting request
""".format(metaString(meta), contentType, address, hash ))

        return {"approved": False, "password" : None}

    @public
    def approveNewAccount(self, req):
        """ Example request
        {"jsonrpc":"2.0","id":25,"method":"ui_approveNewAccount","params":[{"meta":{"remote":"clef binary","local":"main","scheme":"in-proc","User-Agent":"","Origin":""}}]}
        """
        meta = req.get('meta', {})
        sys.stdout.write("""Create new account request:
    {}
    
    Auto-rejecting request
""".format(metaString(meta) ))
        return {"approved": False}

    @public
    def showError(self,req):
        """
        Example request
        {"jsonrpc":"2.0","method":"ui_showError","params":[{"text":"If you see this message, enter 'yes' to the next question"}]}

        :param message: to display
        :return:nothing
        """
        text = req.get('text')
        sys.stdout.write("Error: {}\n".format(text) )
        sys.stdout.write("Press enter to continue\n")
        input()
        return

    @public
    def showInfo(self,req):
        """
        Example request
        {"jsonrpc":"2.0","method":"ui_showInfo","params":[{"text":"If you see this message, enter 'yes' to next question"}]}

        :param message: to display
        :return:nothing
        """
        text = req.get('text')
        sys.stdout.write("Info: {}\n".format(text) )
        sys.stdout.write("Press enter to continue\n")
        input()
        return

    @public
    def onSignerStartup(self,req):
        """
        Example request
         {"jsonrpc":"2.0",
         "method":"ui_onSignerStartup",
         "params":[{"info":{"extapi_http":"n/a","extapi_ipc":"/home/user/.clef/clef.ipc","extapi_version":"6.1.0","intapi_version":"7.0.1"}}]}
        """
        info = req.get('info')
        http = info.get('extapi_http')
        ipc = info.get('extapi_ipc')
        extVer =  info.get('extapi_version')
        intVer =  info.get('intapi_version')
        sys.stdout.write("""
        Ext api url:{}
        Int api ipc: {}
        Ext api ver: {}
        Int api ver: {}
""".format(http, ipc, extVer, intVer))

    @public
    def approveListing(self,req):
        """
         {"jsonrpc":"2.0","id":23,"method":"ui_approveListing","params":[{"accounts":[{"address":...
        """
        accounts = req.get('accounts',[])
        addrs = [x.get("address") for x in accounts]

        sys.stdout.write("\n## Account listing request\n\tDo you want to allow listing the following accounts?\n\t-{}\n\n->"
        .format( "\n\t-".join(addrs)))
        sys.stdout.write("Auto-answering No\n")
        return {}

    @public
    def onInputRequired(self,req):
        """
        Example request
        {"jsonrpc":"2.0","id":1,"method":"ui_onInputRequired","params":[{"title":"Master Password","prompt":"Please enter the password to decrypt the master seed","isPassword":true}]}

        :param message: to display
        :return:nothing
        """
        title = req.get('title')
        isPassword = req.get("isPassword")
        prompt = req.get('prompt')
        sys.stdout.write("\n## {}\n\t{}\n\n> ".format( title, prompt))
        if not isPassword:
            return { "text": input()}

        return ""

def main(args):
    cmd = ["clef", "--stdio-ui"]
    if len(args) > 0 and args[0] == "test":
        cmd.extend(["--stdio-ui-test"])
    print("cmd: {}".format(" ".join(cmd)))
    dispatcher = RPCDispatcher()
    dispatcher.register_instance(StdIOHandler(), 'ui_')

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
