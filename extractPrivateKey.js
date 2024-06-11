const fs = require('fs');
const keythereum = require('keythereum');
const path = require('path');

// Path to the keystore file
const keystoreDir = path.join(__dirname, 'node2', 'keystore');
const keystoreFile = fs.readdirSync(keystoreDir).find(file => file.startsWith('UTC'));
if (!keystoreFile) {
    console.error('Keystore file not found');
    process.exit(1);
}

const keystorePath = path.join(keystoreDir, keystoreFile);
const password = '123'; // Replace with your keystore password

// Read and parse the keystore file
try {
    const keystore = JSON.parse(fs.readFileSync(keystorePath));

    // Extract the private key using the password
    keythereum.recover(password, keystore, (privateKey) => {
        if (!privateKey) {
            console.error('Failed to recover private key');
            process.exit(1);
        }
        console.log(`Private key: 0x${privateKey.toString('hex')}`);
    });
} catch (error) {
    console.error('Error reading keystore file:', error);
    process.exit(1);
}

