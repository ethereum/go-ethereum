---
title: Mobile API
---

The Ethereum blockchain along with its two extension protocols Whisper and Swarm was
originally conceptualized to become the supporting pillar of web3, providing the
consensus, messaging and storage backbone for a new generation of distributed (actually,
decentralized) applications called DApps.

The first incarnation towards this dream of web3 was a command line client providing an
RPC interface into the peer-to-peer protocols. The client was soon enough extended with a
web-browser-like graphical user interface, permitting developers to write DApps based on
the tried and proven HTML/CSS/JS technologies.

As many DApps have more complex requirements than what a browser environment can handle,
it became apparent that providing programmatic access to the web3 pillars would open the
door towards a new class of applications. As such, the second incarnation of the web
dream is to open up all our technologies for other projects as reusable components.

Starting with the 1.5 release family of `go-ethereum`, we transitioned away from providing
only a full blown Ethereum client and started shipping official Go packages that could be
embedded into third party desktop and server applications. It took only a small leap from
here to begin porting our code to mobile platforms.

## Quick overview

Similarly to our reusable Go libraries, the mobile wrappers also focus on four main usage
areas:

- Simplified client side account management
- Remote node interfacing via different transports
- Contract interactions through auto-generated bindings
- In-process Ethereum, Whisper and Swarm peer-to-peer node

You can watch a quick overview about these in Peter's (@karalabe) talk titled "Import
Geth: Ethereum from Go and beyond", presented at the Ethereum Devcon2 developer conference
in September, 2016 (Shanghai). Slides are [available
here](https://ethereum.karalabe.com/talks/2016-devcon.html).

[![Peter's Devcon2 talk](https://img.youtube.com/vi/R0Ia1U9Gxjg/0.jpg)](https://www.youtube.com/watch?v=R0Ia1U9Gxjg)

## Library bundles

The `go-ethereum` mobile library is distributed either as an Android `.aar` archive
(containing binaries for `arm-7`, `arm64`, `x86` and `x64`); or as an iOS XCode framework
(containing binaries for `arm-7`, `arm64` and `x86`). We do not provide library bundles
for Windows phone the moment.

### Android archive

The simplest way to use `go-ethereum` in your Android project is through a Maven
dependency. We provide bundles of all our stable releases (starting from v1.5.0) through
Maven Central, and also provide the latest develop bundle through the Sonatype OSS
repository.

#### Stable dependency (Maven Central)

To add an Android dependency to the **stable** library release of `go-ethereum`, you'll
need to ensure that the Maven Central repository is enabled in your Android project, and
that the `go-ethereum` code is listed as a required dependency of your application. You
can do both of these by editing the `build.gradle` script in your Android app's folder:

```gradle
repositories {
    mavenCentral()
}

dependencies {
    // All your previous dependencies
    compile 'org.ethereum:geth:1.5.2' // Change the version to the latest release
}
```

#### Develop dependency (Sonatype)

To add an Android dependency to the current version of `go-ethereum`, you'll need to
ensure that the Sonatype snapshot repository is enabled in your Android project, and that
the `go-ethereum` code is listed as a required `SNAPSHOT` dependency of your application.
You can do both of these by editing the `build.gradle` script in your Android app's
folder:

```gradle
repositories {
    maven {
        url "https://oss.sonatype.org/content/groups/public"
    }
}

dependencies {
    // All your previous dependencies
    compile 'org.ethereum:geth:1.5.3-SNAPSHOT' // Change the version to the latest release
}
```

#### Custom dependency

If you prefer not to depend on Maven Central or Sonatype; or would like to access an older
develop build not available any more as an online dependency, you can download any bundle
directly from [our website](https://geth.ethereum.org/downloads/) and insert it into your
project in Android Studio via `File -> New -> New module... -> Import .JAR/.AAR Package`.

You will also need to configure `gradle` to link the mobile library bundle to your
application. This can be done by adding a new entry to the `dependencies` section of your
`build.gradle` script, pointing it to the module you just added (named `geth` by default).

```gradle
dependencies {
    // All your previous dependencies
    compile project(':geth')
}
```

#### Manual builds

Lastly, if you would like to make modifications to the `go-ethereum` mobile code and/or
build it yourself locally instead of downloading a pre-built bundle, you can do so using a
`make` command. This will create an Android archive called `geth.aar` in the `build/bin`
folder that you can import into your Android Studio as described above.

```bash
$ make android
[...]
Done building.
Import "build/bin/geth.aar" to use the library.
```

### iOS framework

The simplest way to use `go-ethereum` in your iOS project is through a
[CocoaPods](https://cocoapods.org/) dependency. We provide bundles of all our stable
releases (starting from v1.5.3) and also latest develop versions.

#### Automatic dependency

To add an iOS dependency to the current stable or latest develop version of `go-ethereum`,
you'll need to ensure that your iOS XCode project is configured to use CocoaPods.
Detailing that is out of scope in this document, but you can find a guide in the upstream
[Using CocoaPods](https://guides.cocoapods.org/using/using-cocoapods.html) page.
Afterwards you can edit your `Podfile` to list `go-ethereum` as a dependency:

```ruby
target 'MyApp' do
    # All your previous dependencies
    pod 'Geth', '1.5.4' # Change the version to the latest release
end
```

Alternatively, if you'd like to use the latest develop version, replace the package
version `1.5.4` with `~> 1.5.5-unstable` to switch to pre-releases and to always pull in
the latest bundle from a particular release family.

#### Custom dependency

If you prefer not to depend on CocoaPods; or would like to access an older develop build
not available any more as an online dependency, you can download any bundle directly from
[our website](https://geth.ethereum.org/downloads/) and insert it into your project in
XCode via `Project Settings -> Build Phases -> Link Binary With Libraries`.

Do not forget to extract the framework from the compressed `.tar.gz` archive. You can do
that either using a GUI tool or from the command line via (replace the archive with your
downloaded file):

```
tar -zxvf geth-ios-all-1.5.3-unstable-e05d35e6.tar.gz
```

#### Manual builds

Lastly, if you would like to make modifications to the `go-ethereum` mobile code and/or
build it yourself locally instead of downloading a pre-built bundle, you can do so using a
`make` command. This will create an iOS XCode framework called `Geth.framework` in the
`build/bin` folder that you can import into XCode as described above.

```bash
$ make ios
[...]
Done building.
Import "build/bin/Geth.framework" to use the library.
```
