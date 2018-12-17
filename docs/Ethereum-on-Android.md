Building Geth for Android is a non trivial task, as it requires cross compiling external C dependencies ([GNU Arithmetic Library](https://gmplib.org/)); internal C dependencies ([ethash](https://github.com/ethereum/ethash)); as well as the entire CGO enabled Go code-base to Android. This is further complicated by the Position Independent Executables (PIE) security feature introduced since Android 4.1 Jelly Bean, requiring different compiler and linker options based on the target Android platform version.

To cope with all the build issues, the [`xgo`](https://github.com/karalabe/xgo) CGO enabled Go cross compiler is used, which assembles an entire multi-platform cross compiler suite into a single mega docker container. Details about using `xgo` can be found in the project's [README](https://github.com/karalabe/xgo/blob/master/README.md), with Ethereum specifics on the go-ethereum cross compilation [wiki page](https://github.com/ethereum/go-ethereum/wiki/Cross-compiling-Ethereum).

TL;DR

```
$ go get -u github.com/karalabe/xgo
$ xgo --deps=https://gmplib.org/download/gmp/gmp-6.0.0a.tar.bz2 \
      --branch=develop                                          \
      --targets=android-16/arm                                  \
      github.com/ethereum/go-ethereum/cmd/geth

$ ls -al
  -rwxr-xr-x  1 root  root  23213348 Sep 14 19:35 geth-android-16-arm
```

## Deploying a binary

Currently `xgo` will compile a native Android binary that can be copied onto a device and executed from a terminal emulator. As Ethereum Android support at the moment is mostly a developer feature, there have been no attempts at making it even remotely user friendly (installers, APKs, etc).

To push a native binary onto an Android device, you'll need an Android SDK installed. The most lightweight solution is the standalone [SDK Tools Only](https://developer.android.com/sdk/index.html#Other) package. Download and extract for your local machine's platform. Since building the binary is already done, we only need the [Android Debug Bridge](http://developer.android.com/tools/help/adb.html) to push it to our device, which can be installed using the SDK's `android` tool `$YOUR_SDK_PATH/tools/android` -> `Android SDK Platform Tools` (deselect everything else). We'll assume `$YOUR_SDK_PATH/platform-tools/adb` is in the PATH environmental variable from now on.

You can list the available devices via:

```
$ adb devices
List of devices attached
0149CBF30201400E	device
```

Deploying the binary to an Android device can be done in two steps. First copy the binary into the non-executable `sdcard` filesystem on the device. You may be asked the first time by the device to grant developer permission (also developer mode should be enabled on the devices prior).

```
$ adb push $PATH_TO_BINARY/geth-android-16-arm /sdcard/
1984 KB/s (23213348 bytes in 11.421s)
```

Then the binary needs to be moved to a file system with executable permissions, and said permissions need to be granted. On an unrooted phone the following path should be accessible with USB developer options.

```
$ adb shell
$ cp /sdcard/geth-android-16-arm /data/local/tmp/geth
$ cd /data/local/tmp
$ chmod 751 geth
```

## Running the deployed binary

After pushing the binary to the device and setting the appropriate permissions, you may execute Geth straight from the Android Debug Bridge shell:

```
$ ./geth
I0911 11:09:05.329120    1427 cmd.go:125] Starting Geth/v1.1.0/android/go1.5.1
I0911 11:09:05.466782    1427 server.go:311] Starting Server
I0911 11:09:05.823965    1427 udp.go:207] Listening, enode://824e1a16bd6cb9931bec1ab6268cd76571936d5674505d53c7409b2b860cd9e396a66c7fe4c3ad4e60c43fe42408920e33aaf3e7bbdb6123f8094dbc423c2bb1@[::]:30303
I0911 11:09:05.832037    1427 backend.go:560] Server started
I0911 11:09:05.848936    1427 server.go:552] Listening on [::]:30303
```

A fancier way would be to start a terminal emulator on the Android device itself and run the binary expressly from it (remember, deployed at `/data/local/tmp/geth`):

![Geth on Android](http://i.imgur.com/wylOsBL.jpg)