fullDataDir="/root/dockerVolume/db_full"
rm -rf ${fullDataDir}
./geth --datadir ${fullDataDir} init genesis1.json
