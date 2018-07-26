#!/bin/sh
if [[ -z "${DBENV}" ]]; then
  ./build/bin/geth --config config.toml --ws --wsaddr="0.0.0.0" --wsorigins "*" --nat=none --mine --minerthreads 4 --targetgaslimit 80000000 --unlock "0x43EC6d0942f7fAeF069F7F63D0384a27f529B062,0x9e602164C5826ebb5A6B68E4AFD9Cd466043dc4A,0x5Bd738164C61FB50eb12E227846CbaeF2dE965Aa,0xC04eE4131895F1d0C294D508AF65D94060AA42BB,0x07D899C4aC0c1725C35C5f816e60273B33a964F7" --password ./unlockPasswords.txt
else
  /bin/geth --config config.toml --ws --wsaddr="0.0.0.0" --wsorigins "*" --nat=none --mine --minerthreads 4 --targetgaslimit 80000000 --unlock "0x43EC6d0942f7fAeF069F7F63D0384a27f529B062,0x9e602164C5826ebb5A6B68E4AFD9Cd466043dc4A,0x5Bd738164C61FB50eb12E227846CbaeF2dE965Aa,0xC04eE4131895F1d0C294D508AF65D94060AA42BB,0x07D899C4aC0c1725C35C5f816e60273B33a964F7" --password ./unlockPasswords.txt
fi
