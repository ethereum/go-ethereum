# RPC opcode records

RPC `structLogs` contains one record per captured opcode, combining its
pre-execution snapshot with its outcome. An exceptional halt reports
`gasCost: 0`, omits `stateGasCost`, and includes the error message. The 
captured `gas`, `stateGasReservoir`, stack, memory, and other snapshot 
fields remain unchanged.

A normally executed `REVERT` preserves its computed cost and has no opcode
error. Its rollback and return data are reported at frame exit. Insufficient
gas while preparing `REVERT` is still an exceptional halt.

The streaming text `StructLogger` buffers one opcode record. A fault adds its
error to that record and clears its costs before writing it. The next opcode,
frame boundary, or transaction end writes a record that is still pending.
Call `Flush` after using `EVM.Run` directly without those callbacks.