package storage

// The DB schema we want to use. The actual/current DB schema might differ
// until migrations are run.
const CurrentDbSchema = DbSchemaHalloween

// There was a time when we had no schema at all.
const DbSchemaNone = ""

// "purity" is the first formal schema of LevelDB we release together with Swarm 0.3.5
const DbSchemaPurity = "purity"

// "halloween" is here because we had a screw in the garbage collector index.
// Because of that we had to rebuild the GC index to get rid of erroneous
// entries and that takes a long time. This schema is used for bookkeeping,
// so rebuild index will run just once.
const DbSchemaHalloween = "halloween"
