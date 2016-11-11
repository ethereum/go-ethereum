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

// Contains all the wrappers from the accounts package to support client side key
// management on mobile platforms.

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

import org.ethereum.geth.*;

public class AndroidTest extends InstrumentationTestCase {
	public AndroidTest() {}

	public void testAccountManagement() {
		try {
			AccountManager am = new AccountManager(getInstrumentation().getContext().getFilesDir() + "/keystore", Geth.LightScryptN, Geth.LightScryptP);

			Account newAcc = am.newAccount("Creation password");
			byte[] jsonAcc = am.exportKey(newAcc, "Creation password", "Export password");

			am.deleteAccount(newAcc, "Creation password");
			Account impAcc = am.importKey(jsonAcc, "Export password", "Import password");
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
		t.Skip("ANDROID_HOME environment var not set, skipping")
	}
	if _, err := exec.Command("which", "gomobile").CombinedOutput(); err != nil {
		t.Log("gomobile missing, installing it...")
		if _, err := exec.Command("go", "install", "golang.org/x/mobile/cmd/gomobile").CombinedOutput(); err != nil {
			t.Fatalf("install failed: %v", err)
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
        classpath 'com.android.tools.build:gradle:1.5.0'
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
