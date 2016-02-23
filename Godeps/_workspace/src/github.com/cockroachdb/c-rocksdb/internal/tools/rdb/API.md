# JavaScript API

## DBWrapper

### Constructor

    # Creates a new database wrapper object
    RDB()

### Open

    # Open a new or existing RocksDB database.
    #
    # db_name         (string)   - Location of the database (inside the
    #                              `/tmp` directory).
    # column_families (string[]) - Names of additional column families
    #                              beyond the default. If there are no other
    #                              column families, this argument can be
    #                              left off.
    #
    # Returns true if the database was opened successfully, or false otherwise
    db_obj.(db_name, column_families = [])

### Get

    # Get the value of a given key.
    #
    # key           (string) - Which key to get the value of.
    # column_family (string) - Which column family to check for the key.
    #                          This argument can be left off for the default
    #                          column family
    #
    # Returns the value (string) that is associated with the given key if
    # one exists, or null otherwise.
    db_obj.get(key, column_family = { default })

### Put

    # Associate a value with a key.
    #
    # key           (string) - Which key to associate the value with.
    # value         (string) - The value to associate with the key.
    # column_family (string) - Which column family to put the key-value pair
    #                          in. This argument can be left off for the
    #                          default column family.
    #
    # Returns true if the key-value pair was successfully stored in the
    # database, or false otherwise.
    db_obj.put(key, value, column_family = { default })

### Delete

    # Delete a value associated with a given key.
    #
    # key           (string) - Which key to delete the value of..
    # column_family (string) - Which column family to check for the key.
    #                          This argument can be left off for the default
    #                          column family
    #
    # Returns true if an error occured while trying to delete the key in
    # the database, or false otherwise. Note that this is NOT the same as
    # whether a value was deleted; in the case of a specified key not having
    # a value, this will still return true. Use the `get` method prior to
    # this method to check if a value existed before the call to `delete`.
    db_obj.delete(key, column_family = { default })

### Dump

    # Print out all the key-value pairs in a given column family of the
    # database.
    #
    # column_family (string) - Which column family to dump the pairs from.
    #                          This argument can be left off for the default
    #                          column family.
    #
    # Returns true if the keys were successfully read from the database, or
    # false otherwise.
    db_obj.dump(column_family = { default })

### WriteBatch

    # Execute an atomic batch of writes (i.e. puts and deletes) to the
    # database.
    #
    # cf_batches (BatchObject[]; see below) - Put and Delete writes grouped
    #                                         by column family to execute
    #                                         atomically.
    #
    # Returns true if the argument array was well-formed and was
    # successfully written to the database, or false otherwise.
    db_obj.writeBatch(cf_batches)

### CreateColumnFamily

    # Create a new column familiy for the database.
    #
    # column_family_name (string) - Name of the new column family.
    #
    # Returns true if the new column family was successfully created, or
    # false otherwise.
    db_obj.createColumnFamily(column_family_name)

### CompactRange

    # Compact the underlying storage for a given range.
    #
    # In addition to the endpoints of the range, the method is overloaded to
    # accept a non-default column family, a set of options, or both.
    #
    # begin (string)         - First key in the range to compact.
    # end   (string)         - Last key in the range to compact.
    # options (object)       - Contains a subset of the following key-value
    #                          pairs:
    #                            * 'target_level'   => int
    #                            * 'target_path_id' => int
    # column_family (string) - Which column family to compact the range in.
    db_obj.compactRange(begin, end)
    db_obj.compactRange(begin, end, options)
    db_obj.compactRange(begin, end, column_family)
    db_obj.compactRange(begin, end, options, column_family)



### Close

    # Close an a database and free the memory associated with it.
    #
    # Return null.
    # db_obj.close()


## BatchObject

### Structure

A BatchObject must have at least one of the following key-value pairs:

* 'put' => Array of ['string1', 'string1'] pairs, each of which signifies that
the key 'string1' should be associated with the value 'string2'
* 'delete' => Array of strings, each of which is a key whose value should be
deleted.

The following key-value pair is optional:

* 'column_family' => The name (string) of the column family to apply the
changes to.

### Examples

    # Writes the key-value pairs 'firstname' => 'Saghm' and
    # 'lastname' => 'Rossi' atomically to the database.
    db_obj.writeBatch([
        {
            put: [ ['firstname', 'Saghm'], ['lastname', 'Rossi'] ]
        }
    ]);


    # Deletes the values associated with 'firstname' and 'lastname' in
    # the default column family and adds the key 'number_of_people' with
    # with the value '2'. Additionally, adds the key-value pair
    # 'name' => 'Saghm Rossi' to the column family 'user1' and the pair
    # 'name' => 'Matt Blaze' to the column family 'user2'. All writes
    # are done atomically.
    db_obj.writeBatch([
        {
            put: [ ['number_of_people', '2'] ],
            delete: ['firstname', 'lastname']
        },
        {
            put: [ ['name', 'Saghm Rossi'] ],
            column_family: 'user1'
        },
        {
            put: [ ['name', Matt Blaze'] ],
            column_family: 'user2'
        }
    ]);
