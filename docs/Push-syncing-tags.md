Push syncing tags
=====

motivation
------
push syncing tags are to provide the ability to measure how long it is going to take for a file uploaded to swarm to finish syncing, this in turn allows a node (presumably a light node) to know when it can go offline once all uploads are synced to the swarm.

definitions
---

* tag - an upload tag which is transactional and maps an entire upload transaction to a unique identifier. this ID is randomly generated on the fly. (i.e. `swarm up --recursive mydir`, `mydir` being the tag)


* tag index - an index that creates a unique ID for each upload, allowing for less full iterations on indexes when querying all results for pending tags. defined as `UploadID|TARFilename->TotalChunkCount|SyncedChunkCount`
or `UploadID->TotalChunkCount|SyncedChunkCount|TARFilename`

* push index - `localstore` push syncing index

tags spec
----

* whats an upload tag?
    * an ID that is transactional for a complete upload (i.e. `swarm up <dirname>`)

* tag index operations:
    * create - once an upload txn is started (saves too)
    * persist - save a tag to disk
    * delete - once upload txn is synced to the swarm (maybe we don't want to delete immediately? let user see upload history? including uploaded hashes?)
    * get one/all - get status of upload(s)


* which operations should tags facilitate?
    * get count of distinct chunks for a file
    * get count of chunks pending to sync for a file
    * get existing tags (files with pending syncronisation)


example sync status
```
swarm status sync
file1.tar.gz, 5% complete, ETA 05:52
[=========>    ]
file2.tar.gz, 99% complete, ETA 00:02
[=============>]
```

## Issues

### other states

as part of this effort we want to support progress bars/metrics for

* progress of chunking (splitting to chunks)
* progress of storage
* progress of sending out to push sync

For this we need to introduce counts for 5 states

* SPLIT - count chunk instances
* STORED - count chunk instances
* SEEN - count of chunks previously stored (duplicates)
* SENT - count distinct chunks
* SYNCED - count distinct chunks

progress on a state is characterised by 2 integers `c, n` standing for "completed `c` chunks out of known `n`". This is the main interface that progress bar UX can call and also which enables ETA calculation .


If we want progress on localstore storage, the STORED count should increment every time localstore `Put` is called.

### known file sizes

If we know a files's size we can use it to calculate the total number of chunks (note that it depends on encryption), so that progress of chunking (SPLIT) and storage (STORED) can be meaningful.

If the size and total number of chunks split is not yet known, progress of SPLIT is undefined. After the chunker finished splitting, one can set `total` to the SPLIT count.

If we relied on SPLIT count only, we would lose the very common use case of uploading one file.
Note that if upload also includes a manifest, the total count will serve only as an estimation until `total` is set SPLIT count. This estimation converges to the correct value as the  size of the file grows.


### duplicate chunks

Duplicate chunks are chunks that occur multiple times within an upload or across uploads. In order to have a locally verifiable definition, we define a chunk as a duplicate (or seen) if and only if it is already found in the localstore.
When chunks enter the localstore via upload they are push synced, therefore seen chunks need not push sync again.

In other words only newly stored chunks need counting when assessing the synced ETA of an upload.

If we want progress on SENT/SYNCED counts, we need to give a status, where `n ` represents the total count of *distinct* chunks. Therefore SENT/SYNCED need comparison to `STORED`.
