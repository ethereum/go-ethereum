const net = require('net');
const path = require('path');

let conn = net.connect(path.join('\\\\?\\pipe', 'ethereum-signer'))

const req = {
    id: 1234,
    jsonrpc: '2.0',
    //method: 'account_list',
    method: 'account_signTransaction',
    params: ['0xaabbccddaabbccddaabbccddaabbccddaabbccdd', {
      to: '0x0011223344556677889900112233445566778899',
      value: '0x123450000',
      data: '0xabcdef',
      gas: '0x12345',
      gasPrice: '0x67890'
    }]
};

conn.on('data', (data) => {
  console.log(data.toString());
});

conn.write(JSON.stringify(req));
