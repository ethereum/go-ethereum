---
title: Mobile Clients
---
**This page has been obsoleted. An new guide is in the progress at [[Mobile: Introduction]]**

---

*This page is meant to be a guide on using go-ethereum from mobile platforms. Since neither the mobile libraries nor the light client protocol is finalized, the content here will be sparse, with emphasis being put on how to get your hands dirty. As the APIs stabilize this section will be expanded accordingly.*

### Changelog

* 30th September, 2016: Create initial page, upload Android light bundle.

### Background

Before reading further, please skim through the slides of a Devcon2 talk: [Import Geth: Ethereum from Go and beyond](https://ethereum.karalabe.com/talks/2016-devcon.html), which introduces the basic concepts behind using go-ethereum as a library, and also showcases a few code snippets on how you can do various client side tasks, both on classical computing nodes as well as Android devices. A recording of the talk will be linked when available.

*Please note, the Android and iOS library bundles linked in the presentation will not be updated (for obvious posterity reasons), so always grab latest bundles from this page (until everything is merged into the proper build infrastructure).*

### Mobile bundles

You can download the latest bundles at:

 * [Android (30th September, 2016)](https://bintray.com/karalabe/ethereum/download_file?file_path=geth.aar) - `SHA1: 753e334bf61fa519bec83bcb487179e36d58fc3a`
 * iOS: *light client has not yet been bundled*

### Android quickstart

We assume you are using Android Studio for your development. Please download the latest Android `.aar` bundle from above and import it into your Android Studio project via `File -> New -> New Module`. This will result in a `geth` sub-project inside your work-space. To use the library in your project, please modify your apps `build.gradle` file, adding a dependency to the Geth library:

```gradle
dependencies {
    // All your previous dependencies
    compile project(':geth')
}
```

To get you hands dirty, here's a code snippet that will

* Start up an in-process light node inside your Android application
* Display some initial infos about your node
* Subscribe to new blocks and display them live as they arrive

<img src="http://i.imgur.com/LyTCCqg.png" width="512px" />

```java
import org.ethereum.geth.*;

public class MainActivity extends AppCompatActivity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        setTitle("Android In-Process Node");
        final TextView textbox = (TextView) findViewById(R.id.textbox);

        Context ctx = new Context();

        try {
            Node node = Geth.newNode(getFilesDir() + "/.ethereum", new NodeConfig());
            node.start();

            NodeInfo info = node.getNodeInfo();
            textbox.append("My name: " + info.getName() + "\n");
            textbox.append("My address: " + info.getListenerAddress() + "\n");
            textbox.append("My protocols: " + info.getProtocols() + "\n\n");

            EthereumClient ec = node.getEthereumClient();
            textbox.append("Latest block: " + ec.getBlockByNumber(ctx, -1).getNumber() + ", syncing...\n");

            NewHeadHandler handler = new NewHeadHandler() {
                @Override public void onError(String error) { }
                @Override public void onNewHead(final Header header) {
                    MainActivity.this.runOnUiThread(new Runnable() {
                        public void run() { textbox.append("#" + header.getNumber() + ": " + header.getHash().getHex().substring(0, 10) + ".\n"); }
                    });
                }
            };
            ec.subscribeNewHead(ctx, handler,  16);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
```

#### Known quirks

 * Many constructors (those that would throw exceptions) are of the form `Geth.newXXX()`, instead of simply the Java style `new XXX()` This is an upstream limitation of the [gomobile](https://github.com/golang/mobile) project, one which is currently being worked on to resolve.
 * There are zero documentations attached to the Java library methods. This too is a limitation of the [gomobile](https://github.com/golang/mobile) project. We will try to propose a fix upstream to make our docs from the Go codebase available in Java.
