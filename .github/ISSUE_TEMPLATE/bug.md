---
name: Report a bug
about: Something with bor client is not working as expected
title: ''
labels: 'type:bug'
assignees: ''
---

Our support team has aggregated some common issues and their solutions from past which are faced while running or interacting with a bor client. In order to prevent redundant efforts, we would encourage you to have a look at the [FAQ's section](https://wiki.polygon.technology/docs/faq/technical-faqs/) of our documentation mentioning the same, before filing an issue here. In case of additional support, you can also join our [discord](https://discord.com/invite/0xPolygonDevs) server

<!--
NOTE: Please make sure to check of any addresses / private keys / rpc url's / IP's before sharing the logs or anything from the additional information section (start.sh or heimdall config).
-->

#### **System information**

Bor client version: [e.g. v0.2.16] <!--Can be found by running the command `bor version`-->

Heimdall client version: [e.g. v0.2.10] <!--Can be found by running the command `heimdalld version`-->

OS & Version: Windows / Linux / OSX

Environment: Polygon Mainnet / Polygon Mumbai / Polygon Amoy / Devnet

Type of node: Validator / Sentry / Archive

Additional Information: <!--Modifications in the client (if any)-->

#### **Overview of the problem**

Please describe the issue you experiencing.
<!--
Mention in detail about the issue. Also mention the actual and expected behaviour.
-->

#### **Reproduction Steps**

Please mention the steps required to reproduce this issue. 

<!--
E.g. 
1. Start bor using these flags. 
2. Node is unable to connect with other peers in the network and keeps disconnecting. 
-->

#### **Logs / Traces / Output / Error Messages**
 
Please post any logs/traces/output/error messages (as text and not screenshots) which you believe may have caused the issue. If the log is longer than a few dozen lines, please include the URL to the [gist](https://gist.github.com/) of the log instead of posting it in the issue.

**Additional Information**

In order to debug the issue faster, we would stongly encourage if you can provide some of the details mentioned below (whichever seems relevant to your issue)

1. Your `start.sh` file or `bor.service`, if you're facing some peering issue or unable to use some service (like `http` endpoint) as expected. Moreover, if possible mention the chain configuration printed while starting the node which looks something like `Initialised chain configuration config="{ChainID: 137, ..., Engine: bor}"`
<!--
It should be start.sh if you're using bor v0.2.x and bor.service (ideally located under `/lib/systemd/system/`) if it's bor v0.3.x. Mention this file if you're facing any issues like unable to use some flag/s according to their expected behaviour.
-->
2. The result of `eth.syncing`, `admin.peers.length`, `admin.nodeInfo`, value of the `maxpeers` flag in start.sh, and bootnodes/static nodes (if any) is you're facing some syncing issue.
<!--
You can get the above results by attaching to the IPC using the command `bor attach $BORDIR/bor.ipc` or `bor attach $DATADIR/bor.ipc` and running the mentioned commands. 
Mention this if you're facing issues where bor keeps stalling and is not importing new blocks or making any progress. Adding chain configuration mentioned in the previous step would also be really helpful here as it might also be a genesis mismatch issue.
-->
3. Your `heimdall-config.toml` parameters for checking the ETH and BOR RPC url's, incase of issue with bor heimdall communication. 
<!--
The location should be `~/.heimdalld/config/` if running heimdall v0.2.x and `/var/lib/heimdalld/config` if running heimdall v0.3.x. 
As a sub-set of syncing issues, if your node keeps printing logs like `Retrying again in 5 seconds to fetch data from Heimdall`, it might be an issue with the communication between your bor node and heimdall node. In this case, also check if all the heimdall services (heimdalld, bridge, rest-server) are running correctly.
-->
4. The CURL request (for that specific error) if you're facing any issues or identify a bug while making RPC request.  
<!--
Make sure you hide the IP of your machine if you're doing the request externally.  
-->
