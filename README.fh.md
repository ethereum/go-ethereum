## Firehose Instructions

This readme contains instructions about how to manage Firehose branches and other important
instructions needed to develop and maintain the Firehose Tracer.

### Regenerate Protobufs

```bash
buf generate buf.build/streamingfast/firehose-ethereum --exclude-path sf/ethereum/substreams,sf/ethereum/trxstream,sf/ethereum/transform
```

> [!NOTE]
> You can generate from a local path `buf generate ../firehose-ethereum/proto ...` when developing new features locally.

