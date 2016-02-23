## Cross-building

RocksDB can be built as a single self contained cross-platform JAR. The cross-platform jar can be usd on any 64-bit OSX system, 32-bit Linux system, or 64-bit Linux system.

Building a cross-platform JAR requires:

 * [Vagrant](https://www.vagrantup.com/)
 * [Virtualbox](https://www.virtualbox.org/)
 * A Mac OSX machine that can compile RocksDB.
 * Java 7 set as JAVA_HOME.

Once you have these items, run this make command from RocksDB's root source directory:

    make jclean clean rocksdbjavastaticrelease

This command will build RocksDB natively on OSX, and will then spin up two Vagrant Virtualbox Ubuntu images to build RocksDB for both 32-bit and 64-bit Linux. 

You can find all native binaries and JARs in the java/target directory upon completion:

    librocksdbjni-linux32.so
    librocksdbjni-linux64.so
    librocksdbjni-osx.jnilib
    rocksdbjni-3.5.0-javadoc.jar
    rocksdbjni-3.5.0-linux32.jar
    rocksdbjni-3.5.0-linux64.jar
    rocksdbjni-3.5.0-osx.jar
    rocksdbjni-3.5.0-sources.jar
    rocksdbjni-3.5.0.jar

## Maven publication

Set ~/.m2/settings.xml to contain:

    <settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 http://maven.apache.org/xsd/settings-1.0.0.xsd">
      <servers>
        <server>
          <id>sonatype-nexus-staging</id>
          <username>your-sonatype-jira-username</username>
          <password>your-sonatype-jira-password</password>
        </server>
      </servers>
    </settings>

From RocksDB's root directory, first build the Java static JARs:

    make jclean clean rocksdbjavastaticpublish

This command will [stage the JAR artifacts on the Sonatype staging repository](http://central.sonatype.org/pages/manual-staging-bundle-creation-and-deployment.html). To release the staged artifacts.

1. Go to [https://oss.sonatype.org/#stagingRepositories](https://oss.sonatype.org/#stagingRepositories) and search for "rocksdb" in the upper right hand search box.
2. Select the rocksdb staging repository, and inspect its contents.
3. If all is well, follow [these steps](https://oss.sonatype.org/#stagingRepositories) to close the repository and release it.

After the release has occurred, the artifacts will be synced to Maven central within 24-48 hours.
