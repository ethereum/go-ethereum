fbson commit: 
https://github.com/facebook/mysql-5.6/commit/55ef9ff25c934659a70b4094e9b406c48e9dd43d

# TODO.
* Had to convert zero sized array to [1] sized arrays due to the fact that MS Compiler complains about it not being standard. At some point need to contribute this change back to MySql where this code was taken from.
