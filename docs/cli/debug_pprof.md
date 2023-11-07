# Debug Pprof

The ```debug pprof <enode>``` command will create an archive containing bor pprof traces.

## Options

- ```address```: Address of the grpc endpoint (default: 127.0.0.1:3131)

- ```output```: Output directory

- ```seconds```: seconds to profile (default: 2)

- ```skiptrace```: Skip running the trace (default: false)