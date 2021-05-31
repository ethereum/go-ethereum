# the-index geth

## Build

1. brew install go (1.15+)

2. git clone into `$GOPATH/src/github.com/orbs-network/the-index-go-ethereum`

3. go into the repo directory 

4. run `make geth`

## Develop (on Rinkeby)

1. go into the repo directory

2. make sure `./the-index` directory for index outputs is created

3. to delete old index data run `rm -rf ./the-index/*.rlp`

4. to reset the chain run `./build/bin/geth --rinkeby removedb`

5. run `export THEINDEX_PATH=./the-index/; ./build/bin/geth --rinkeby --syncmode=full --port 0`

## Run (on AWS)

1. Create a new EFS disk for the index data (one zone, no backups, no encryption)

2. Create a i3.xlarge machine without an EBS drive with Ubuntu 20

3. Mount the NVMe disk:

```
sudo mkfs -t xfs /dev/nvme0n1
sudo mkdir /data
sudo mount /dev/nvme0n1 /data
sudo chmod a+w /data
```

4. Fix the EFS permissions:

```
sudo chmod a+w /mnt/efs/fs1
mkdir /mnt/efs/fs1/the-index
```

5. Install missing dependencies:

```
sudo snap install go --classic
export PATH=$PATH:/usr/local/go/bin
sudo apt get gcc
sudo apt get make
```

6. Clone and build:

```
cd ~
mkdir go
cd go
mkdir src
cd src
mkdir github.com
cd github.com
mkdir orbs-network
cd orbs-network
git clone https://github.com/orbs-network/the-index-go-ethereum
cd the-index-go-ethereum
make geth
```

7. Run geth with chaindata on the NVMe and index on the EFS:

```
export THEINDEX_PATH=/mnt/efs/fs1/the-index/
./build/bin/geth --datadir /data/ --cache=4096 --maxpeers=50 --syncmode=full
```