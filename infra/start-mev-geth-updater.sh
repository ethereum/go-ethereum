#!/bin/sh -x
# Starts the Mev-Geth updater client
# Written by Luke Youngblood, luke@blockscale.net

# netport=30303 # normally set by environment

init_node() {
    # Initialization steps can go here
    echo Initializing node...
    aws configure set default.s3.max_concurrent_requests 64
    aws configure set default.s3.max_queue_size 20000
}

start_node() {
    if [ $network = "goerli" ]
    then
        geth \
        --port $netport \
        --syncmode $syncmode \
        --cache 4096 \
        --maxpeers $connections \
        --goerli &
        if [ $? -ne 0 ]
        then
            echo "Node failed to start; exiting."
            exit 1
        fi
    else
        geth \
        --port $netport \
        --syncmode $syncmode \
        --cache 4096 \
        --maxpeers $connections &
        if [ $? -ne 0 ]
        then
            echo "Node failed to start; exiting."
            exit 1
        fi
    fi
}

s3_sync_down() {
    # Determine data directory
    if [ $network = "goerli" ]
    then
        datadir=/root/.ethereum/goerli/geth/chaindata
    else
        datadir=/root/.ethereum/geth/chaindata
    fi

    # If the current1 object exists, node1 is the key we should download
    echo "A 404 error below is expected and nothing to be concerned with."
    aws s3api head-object --bucket $chainbucket --key current1
    if [ $? -eq 0 ]
    then
        echo "current1 key exists; downloading node1"
        s3key=node1
    else
        echo "current1 key doesn't exist; downloading node2"
        s3key=node2
    fi

    aws s3 sync --region $region --only-show-errors s3://$chainbucket/$s3key $datadir
    if [ $? -ne 0 ]
    then
        echo "aws s3 sync command failed; exiting."
        exit 2
    fi
}

kill_node() {
    tries=0
    while [ ! -z `ps -ef |grep geth|grep -v geth-updater|grep -v grep|awk '{print $1}'` ]
    do
        ps -ef |grep geth|grep -v geth-updater|grep -v grep
        pid=`ps -ef |grep geth|grep -v geth-updater|grep -v grep|awk '{print $1}'`
        kill $pid
        sleep 30
        echo "Waiting for the node to shutdown cleanly... try number $tries"
        let "tries+=1"
        if [ $tries -gt 29 ]
        then
            echo "Node has not stopped cleanly after $tries, forcibly killing."
            ps -ef |grep geth|grep -v geth-updater|grep -v grep
            pid=`ps -ef |grep geth|grep -v geth-updater|grep -v grep|awk '{print $1}'`
            kill -9 $pid
        fi
        if [ $tries -gt 30 ]
        then
            echo "Node has not stopped cleanly after $tries, exiting..."
            exit 3
        fi
    done
}

s3_sync_up() {
    # Determine data directory
    if [ $network = "goerli" ]
    then
        datadir=/root/.ethereum/goerli/geth/chaindata
    else
        datadir=/root/.ethereum/geth/chaindata
    fi

    # If the current1 object exists, node1 is the folder that clients will download, so we should update node2
    aws s3api head-object --bucket $chainbucket --key current1
    if [ $? -eq 0 ]
    then
        echo "current1 key exists; updating node2"
        s3key=node2
    else
        echo "current1 key doesn't exist; updating node1"
        s3key=node1
    fi

    aws s3 sync --delete --region $region --only-show-errors --acl public-read $datadir s3://$chainbucket/$s3key
    if [ $? -ne 0 ]
    then
        echo "aws s3 sync upload command failed; exiting."
        exit 4
    fi

    if [ "$s3key" = "node2" ]
    then
        echo "Removing current1 key, as the node2 key was just updated."
        aws s3 rm --region $region s3://$chainbucket/current1
        if [ $? -ne 0 ]
        then
            echo "aws s3 rm command failed; retrying."
            sleep 5
            aws s3 rm --region $region s3://$chainbucket/current1
            if [ $? -ne 0 ]
            then
                echo "aws s3 rm command failed; exiting."
                exit 5
            fi
        fi
    else
        echo "Touching current1 key, as the node1 key was just updated."
        touch ~/current1
        aws s3 cp --region $region --acl public-read ~/current1 s3://$chainbucket/
        if [ $? -ne 0 ]
        then
            echo "aws s3 cp command failed; retrying."
            sleep 5
            aws s3 cp --region $region --acl public-read ~/current1 s3://$chainbucket/
            if [ $? -ne 0 ]
            then
                echo "aws s3 cp command failed; exiting."
                exit 6
            fi
        fi
    fi
}

continuous() {
    # This function continuously stops the node every hour
    # and syncs the chain data with S3, then restarts the node.
    while true
    do
        echo "Sleeping for 60 minutes at `date`..."
        sleep 3600
        echo "Cleanly shutting down the node so we can update S3 with the latest chaindata at `date`..."
        kill_node
        echo "Syncing chain data to S3 at `date`..."
        s3_sync_up
        echo "Restarting the node after syncing to S3 at `date`..."
        start_node
    done
}

# main

echo "Initializing the node at `date`..."
init_node
echo "Syncing initial chain data with stored chain data in S3 at `date`..."
s3_sync_down
echo "Starting the node at `date`..."
start_node
echo "Starting the continuous loop at `date`..."
continuous
