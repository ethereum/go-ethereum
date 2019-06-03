Swarm DB migration notes
=========================
Swarm `v0.4` introduces major changes to the existing codebase. Among other things, the storage layer has been rewritten to be more modular and flexible
in a manner that will accomodate for our future needs. Since Swarm at this point does not provide any storage guarantees, we have made the decision to not impose any migrations on our public cluster nor on our users. What this essentially means is that local storage will be purged on `v0.4`. We have nevertheless, provided a procedure below for those of you running private clusters and would like to migrate the data to the new local storage format.

You are highly encouraged to report to us any bugs or problems caused by running the migration steps below.

**Note**: we highly recommend you run the commands below with `--verbosity 5` flag and open an issue with the relevant terminal output in case something goes wrong.

**Important**: since you would be creating an export of your local store, the potential disk usage might peak at `x2-x3` times the normal Swarm data folder size. Please make sure you have enough disk space, backup mediums or other form of local/network attached storage _before_ executing the following steps!

**Important**: when trying to run Swarm with an old local store format, the Swarm binary will refuse to start showing an error message.

You will need the following information for the migration procedure:
1. Your `datadir` path. This is indicated with the `--datadir` flag when running Swarm. If you do not specify this flag, the `datadir` will reside by default on `$HOME/.ethereum`.
2. Your chunk directory location. This would normally be located in your `datadir/swarm/bzz-<your bzz account>/chunks`. We will refer to this as `chunkDir` below.
3. Your `bzzAddr`. This is _not_ your `--bzzaccount`! You can find your `bzzAddr` when starting Swarm by looking for the following line:
```
INFO [03-21|17:25:04.791] Swarm network started                    bzzaddr=ca1e9f3938cc1425c6061b96ad9eb93e134dfe8734ad490164ef20af9d1cf59c
```

The migration process is done in the following manner:
1. Try to run the updated Swarm binary, it should complain about the local store format and exit. If it does - execute the following steps:
2. `$ swarm --verbosity 5 db export <chunkDir> <exportLocation>/<exportFilename>.tar <bzzAddr>`
3. Move or Remove your existing `chunkDir`
4. Run the new Swarm binary as your would start your Swarm node normally. The binary should now load normally and not complain. This step creates a new empty chunk store. Please shut down the node after it starts correctly.
5. `$ swarm --verbosity 5 db import --legacy <chunkDir> <exportLocation>/<exportFilename>.tar <bzzAddr>`
6. Wait patientally for the `Imported X chunks successfully` message.
7. Start your Swarm node as you normally would
8. Have a beer

