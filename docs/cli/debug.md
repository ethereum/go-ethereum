
# Debug

The ```bor debug``` command takes a debug dump of the running client.

## Options

- ```seconds```: Number of seconds to trace cpu and traces.

- ```output```: Output directory for the data dump.

## Examples

By default it creates a tar.gz file with the output:

```
$ bor debug
Starting debugger...                                                                                 
                                                                                                     
Created debug archive: bor-debug-2021-10-26-073819Z.tar.gz
```

Send the output to a specific directory:

```
$ bor debug --output data
Starting debugger...

Created debug directory: data/bor-debug-2021-10-26-075437Z
```
