rm -rf /home/vivid/peernode/*
echo "peer node datadir cleared"
./geth --datadir /home/vivid/peernode init genesis.json

sleep 1s
./geth --datadir /home/vivid/peernode --port 30304 --networkid 9876 console

