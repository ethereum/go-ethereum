rm -rf /home/vivid/.ethereum/*
echo "main node datadir cleared"
./geth --datadir /home/vivid/.ethereum init genesis.json

sleep 1s
./geth --datadir /home/vivid/.ethereum --port 30303 --networkid 9876 console

