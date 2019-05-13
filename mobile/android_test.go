// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package geth

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/internal/build"
)

// androidTestClass is a Java class to do some lightweight tests against the Android
// bindings. The goal is not to test each individual functionality, rather just to
// catch breaking API and/or implementation changes.
const androidTestClass = `
package go;

import android.test.InstrumentationTestCase;
import android.test.MoreAsserts;

import java.math.BigInteger;
import java.util.Arrays;

import org.ethereum.geth.*;

public class AndroidTest extends InstrumentationTestCase {
	public AndroidTest() {}

	public void testAccountManagement() {
		// Create an encrypted keystore with light crypto parameters.
		KeyStore ks = new KeyStore(getInstrumentation().getContext().getFilesDir() + "/keystore", Geth.LightScryptN, Geth.LightScryptP);

		try {
			// Create a new account with the specified encryption passphrase.
			Account newAcc = ks.newAccount("Creation password");

			// Export the newly created account with a different passphrase. The returned
			// data from this method invocation is a JSON encoded, encrypted key-file.
			byte[] jsonAcc = ks.exportKey(newAcc, "Creation password", "Export password");

			// Update the passphrase on the account created above inside the local keystore.
			ks.updateAccount(newAcc, "Creation password", "Update password");

			// Delete the account updated above from the local keystore.
			ks.deleteAccount(newAcc, "Update password");

			// Import back the account we've exported (and then deleted) above with yet
			// again a fresh passphrase.
			Account impAcc = ks.importKey(jsonAcc, "Export password", "Import password");

			// Create a new account to sign transactions with
			Account signer = ks.newAccount("Signer password");

			Transaction tx = new Transaction(
				1, new Address("0x0000000000000000000000000000000000000000"),
				new BigInt(0), 0, new BigInt(1), null); // Random empty transaction
			BigInt chain = new BigInt(1); // Chain identifier of the main net

			// Sign a transaction with a single authorization
			Transaction signed = ks.signTxPassphrase(signer, "Signer password", tx, chain);

			// Sign a transaction with multiple manually cancelled authorizations
			ks.unlock(signer, "Signer password");
			signed = ks.signTx(signer, tx, chain);
			ks.lock(signer.getAddress());

			// Sign a transaction with multiple automatically cancelled authorizations
			ks.timedUnlock(signer, "Signer password", 1000000000);
			signed = ks.signTx(signer, tx, chain);
		} catch (Exception e) {
			fail(e.toString());
		}
	}

	public void testInprocNode() {
		Context ctx = new Context();

		try {
			// Start up a new inprocess node
			Node node = new Node(getInstrumentation().getContext().getFilesDir() + "/.ethereum", new NodeConfig());
			node.start();

			// Retrieve some data via function calls (we don't really care about the results)
			NodeInfo info = node.getNodeInfo();
			info.getName();
			info.getListenerAddress();
			info.getProtocols();

			// Retrieve some data via the APIs (we don't really care about the results)
			EthereumClient ec = node.getEthereumClient();
			ec.getBlockByNumber(ctx, -1).getNumber();

			NewHeadHandler handler = new NewHeadHandler() {
				@Override public void onError(String error)          {}
				@Override public void onNewHead(final Header header) {}
			};
			ec.subscribeNewHead(ctx, handler,  16);
		} catch (Exception e) {
			fail(e.toString());
		}
	}

	// Tests that recovering transaction signers works for both Homestead and EIP155
	// signatures too. Regression test for go-ethereum issue #14599.
	public void testIssue14599() {
		try {
			byte[] preEIP155RLP = new BigInteger("f901fc8032830138808080b901ae60056013565b6101918061001d6000396000f35b3360008190555056006001600060e060020a6000350480630a874df61461003a57806341c0e1b514610058578063a02b161e14610066578063dbbdf0831461007757005b610045600435610149565b80600160a060020a031660005260206000f35b610060610161565b60006000f35b6100716004356100d4565b60006000f35b61008560043560243561008b565b60006000f35b600054600160a060020a031632600160a060020a031614156100ac576100b1565b6100d0565b8060018360005260205260406000208190555081600060005260206000a15b5050565b600054600160a060020a031633600160a060020a031614158015610118575033600160a060020a0316600182600052602052604060002054600160a060020a031614155b61012157610126565b610146565b600060018260005260205260406000208190555080600060005260206000a15b50565b60006001826000526020526040600020549050919050565b600054600160a060020a031633600160a060020a0316146101815761018f565b600054600160a060020a0316ff5b561ca0c5689ed1ad124753d54576dfb4b571465a41900a1dff4058d8adf16f752013d0a01221cbd70ec28c94a3b55ec771bcbc70778d6ee0b51ca7ea9514594c861b1884", 16).toByteArray();
			preEIP155RLP = Arrays.copyOfRange(preEIP155RLP, 1, preEIP155RLP.length);

			byte[] postEIP155RLP = new BigInteger("f86b80847735940082520894ef5bbb9bba2e1ca69ef81b23a8727d889f3ef0a1880de0b6b3a7640000802ba06fef16c44726a102e6d55a651740636ef8aec6df3ebf009e7b0c1f29e4ac114aa057e7fbc69760b522a78bb568cfc37a58bfdcf6ea86cb8f9b550263f58074b9cc", 16).toByteArray();
			postEIP155RLP = Arrays.copyOfRange(postEIP155RLP, 1, postEIP155RLP.length);

			Transaction preEIP155 =  new Transaction(preEIP155RLP);
			Transaction postEIP155 = new Transaction(postEIP155RLP);

			preEIP155.getFrom(null);           // Homestead should accept homestead
			preEIP155.getFrom(new BigInt(4));  // EIP155 should accept homestead (missing chain ID)
			postEIP155.getFrom(new BigInt(4)); // EIP155 should accept EIP 155

			try {
				postEIP155.getFrom(null);
				fail("EIP155 transaction accepted by Homestead");
			} catch (Exception e) {}
		} catch (Exception e) {
			fail(e.toString());
		}
	}
}
`

// TestAndroid runs the Android java test class specified above.
//
// This requires the gradle command in PATH and the Android SDK whose path is available
// through ANDROID_HOME environment variable. To successfully run the tests, an Android
// device must also be available with debugging enabled.
//
// This method has been adapted from golang.org/x/mobile/bind/java/seq_test.go/runTest
func TestAndroid(t *testing.T) {
	// Skip tests on Windows altogether
	if runtime.GOOS == "windows" {
		t.Skip("cannot test Android bindings on Windows, skipping")
	}
	// Make sure all the Android tools are installed
	if _, err := exec.Command("which", "gradle").CombinedOutput(); err != nil {
		t.Skip("command gradle not found, skipping")
	}
	if sdk := os.Getenv("ANDROID_HOME"); sdk == "" {
		// Android SDK not explicitly given, try to auto-resolve
		autopath := filepath.Join(os.Getenv("HOME"), "Android", "Sdk")
		if _, err := os.Stat(autopath); err != nil {
			t.Skip("ANDROID_HOME environment var not set, skipping")
		}
		os.Setenv("ANDROID_HOME", autopath)
	}
	if _, err := exec.Command("which", "gomobile").CombinedOutput(); err != nil {
		t.Log("gomobile missing, installing it...")
		if out, err := exec.Command("go", "get", "golang.org/x/mobile/cmd/gomobile").CombinedOutput(); err != nil {
			t.Fatalf("install failed: %v\n%s", err, string(out))
		}
		t.Log("initializing gomobile...")
		start := time.Now()
		if _, err := exec.Command("gomobile", "init").CombinedOutput(); err != nil {
			t.Fatalf("initialization failed: %v", err)
		}
		t.Logf("initialization took %v", time.Since(start))
	}
	// Create and switch to a temporary workspace
	workspace, err := ioutil.TempDir("", "geth-android-")
	if err != nil {
		t.Fatalf("failed to create temporary workspace: %v", err)
	}
	defer os.RemoveAll(workspace)

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("failed to switch to temporary workspace: %v", err)
	}
	defer os.Chdir(pwd)

	// Create the skeleton of the Android project
	for _, dir := range []string{"src/main", "src/androidTest/java/org/ethereum/gethtest", "libs"} {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Generate the mobile bindings for Geth and add the tester class
	gobind := exec.Command("gomobile", "bind", "-javapkg", "org.ethereum", "github.com/ethereum/go-ethereum/mobile")
	if output, err := gobind.CombinedOutput(); err != nil {
		t.Logf("%s", output)
		t.Fatalf("failed to run gomobile bind: %v", err)
	}
	build.CopyFile(filepath.Join("libs", "geth.aar"), "geth.aar", os.ModePerm)

	if err = ioutil.WriteFile(filepath.Join("src", "androidTest", "java", "org", "ethereum", "gethtest", "AndroidTest.java"), []byte(androidTestClass), os.ModePerm); err != nil {
		t.Fatalf("failed to write Android test class: %v", err)
	}
	// Finish creating the project and run the tests via gradle
	if err = ioutil.WriteFile(filepath.Join("src", "main", "AndroidManifest.xml"), []byte(androidManifest), os.ModePerm); err != nil {
		t.Fatalf("failed to write Android manifest: %v", err)
	}
	if err = ioutil.WriteFile("build.gradle", []byte(gradleConfig), os.ModePerm); err != nil {
		t.Fatalf("failed to write gradle build file: %v", err)
	}
	if output, err := exec.Command("gradle", "connectedAndroidTest").CombinedOutput(); err != nil {
		t.Logf("%s", output)
		t.Errorf("failed to run gradle test: %v", err)
	}
}

const androidManifest = `<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android"
          package="org.ethereum.gethtest"
	  android:versionCode="1"
	  android:versionName="1.0">

		<uses-permission android:name="android.permission.INTERNET" />
</manifest>`

const gradleConfig = `buildscript {
    repositories {
        jcenter()
    }
    dependencies {
        classpath 'com.android.tools.build:gradle:2.2.3'
    }
}
allprojects {
    repositories { jcenter() }
}
apply plugin: 'com.android.library'
android {
    compileSdkVersion 'android-19'
    buildToolsVersion '21.1.2'
    defaultConfig { minSdkVersion 15 }
}
repositories {
    flatDir { dirs 'libs' }
}
dependencies {
    compile 'com.android.support:appcompat-v7:19.0.0'
    compile(name: "geth", ext: "aar")
}
`
