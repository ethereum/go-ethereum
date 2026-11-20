# Opcode trace records

RPC `structLogs` contains one record per captured opcode, combining its
pre-execution snapshot with its outcome. An exceptional halt reports
`gasCost: 0`, omits `stateGasCost`, and includes the error message. The
captured `gas`, `stateGasReservoir`, stack, memory, and other snapshot
fields remain unchanged.

A normally executed `REVERT` preserves its computed cost and has no opcode
error. Its rollback and return data are reported at frame exit. Insufficient
gas while preparing `REVERT` is still an exceptional halt.

The streaming text, JSON, and Markdown loggers buffer at most one opcode record.
A fault adds its error to that record and clears its costs before writing it.
JSON omits the zero `stateGasCost`, and Markdown shows the error in the same
table row. The next opcode, frame boundary, or transaction end writes a record
that is still pending. For a direct `EVM.Run` without those callbacks, flush
the final record with `StructLogger.Flush` or call the logger's `OnTxEnd` hook.
