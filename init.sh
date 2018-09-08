make geth && rm -rf /tmp/foo2 &&
 build/bin/geth --datadir /tmp/foo2 init aura.genesis &&
 build/bin/geth --datadir /tmp/foo2 --nodiscover --maxpeers 0 console

