## RocksDB dump format

The version 1 RocksDB dump format is fairly simple:

1) The dump starts with the magic 8 byte identifier "ROCKDUMP"

2) The magic is followed by an 8 byte big-endian version which is 0x00000001.

3) Next are arbitrarily sized chunks of bytes prepended by 4 byte little endian number indicating how large each chunk is.

4) The first chunk is special and is a json string indicating some things about the creation of this dump.  It contains the following keys:
* database-path: The path of the database this dump was created from.
* hostname: The hostname of the machine where the dump was created.
* creation-time: Unix seconds since epoc when this dump was created.

5) Following the info dump the slices paired into are key/value pairs.
