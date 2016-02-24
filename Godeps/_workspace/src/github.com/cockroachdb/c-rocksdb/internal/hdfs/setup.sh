export USE_HDFS=1
export LD_LIBRARY_PATH=$JAVA_HOME/jre/lib/amd64/server:$JAVA_HOME/jre/lib/amd64:/usr/lib/hadoop/lib/native

export CLASSPATH=
for f in `find /usr/lib/hadoop-hdfs | grep jar`; do export CLASSPATH=$CLASSPATH:$f; done
for f in `find /usr/lib/hadoop | grep jar`; do export CLASSPATH=$CLASSPATH:$f; done
for f in `find /usr/lib/hadoop/client | grep jar`; do export CLASSPATH=$CLASSPATH:$f; done
