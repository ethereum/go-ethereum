// Copyright (c) 2014, Facebook, Inc.  All rights reserved.
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree. An additional grant
// of patent rights can be found in the PATENTS file in the same directory.
package org.rocksdb.util;

import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.Test;

import java.lang.reflect.Field;
import java.lang.reflect.Modifier;

import static org.assertj.core.api.Assertions.assertThat;

public class EnvironmentTest {
  private final static String ARCH_FIELD_NAME = "ARCH";
  private final static String OS_FIELD_NAME = "OS";

  private static String INITIAL_OS;
  private static String INITIAL_ARCH;

  @BeforeClass
  public static void saveState() {
    INITIAL_ARCH = getEnvironmentClassField(ARCH_FIELD_NAME);
    INITIAL_OS = getEnvironmentClassField(OS_FIELD_NAME);
  }

  @Test
  public void mac32() {
    setEnvironmentClassFields("mac", "32");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".jnilib");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-osx.jnilib");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.dylib");
  }

  @Test
  public void mac64() {
    setEnvironmentClassFields("mac", "64");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".jnilib");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-osx.jnilib");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.dylib");
  }

  @Test
  public void nix32() {
    // Linux
    setEnvironmentClassFields("Linux", "32");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux32.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
    // UNIX
    setEnvironmentClassFields("Unix", "32");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux32.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
    // AIX
    setEnvironmentClassFields("aix", "32");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux32.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
  }

  @Test
  public void nix64() {
    setEnvironmentClassFields("Linux", "x64");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux64.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
    // UNIX
    setEnvironmentClassFields("Unix", "x64");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux64.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
    // AIX
    setEnvironmentClassFields("aix", "x64");
    assertThat(Environment.isWindows()).isFalse();
    assertThat(Environment.getJniLibraryExtension()).
        isEqualTo(".so");
    assertThat(Environment.getJniLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni-linux64.so");
    assertThat(Environment.getSharedLibraryFileName("rocksdb")).
        isEqualTo("librocksdbjni.so");
  }

  @Test
  public void detectWindows(){
    setEnvironmentClassFields("win", "x64");
    assertThat(Environment.isWindows()).isTrue();
  }

  @Test(expected = UnsupportedOperationException.class)
  public void failWinJniLibraryName(){
    setEnvironmentClassFields("win", "x64");
    Environment.getJniLibraryFileName("rocksdb");
  }

  @Test(expected = UnsupportedOperationException.class)
  public void failWinSharedLibrary(){
    setEnvironmentClassFields("win", "x64");
    Environment.getSharedLibraryFileName("rocksdb");
  }

  private void setEnvironmentClassFields(String osName,
      String osArch) {
    setEnvironmentClassField(OS_FIELD_NAME, osName);
    setEnvironmentClassField(ARCH_FIELD_NAME, osArch);
  }

  @AfterClass
  public static void restoreState() {
    setEnvironmentClassField(OS_FIELD_NAME, INITIAL_OS);
    setEnvironmentClassField(ARCH_FIELD_NAME, INITIAL_ARCH);
  }

  private static String getEnvironmentClassField(String fieldName) {
    final Field field;
    try {
      field = Environment.class.getDeclaredField(fieldName);
      field.setAccessible(true);
      final Field modifiersField = Field.class.getDeclaredField("modifiers");
      modifiersField.setAccessible(true);
      modifiersField.setInt(field, field.getModifiers() & ~Modifier.FINAL);
      return (String)field.get(null);
    } catch (NoSuchFieldException | IllegalAccessException e) {
      throw new RuntimeException(e);
    }
  }

  private static void setEnvironmentClassField(String fieldName, String value) {
    final Field field;
    try {
      field = Environment.class.getDeclaredField(fieldName);
      field.setAccessible(true);
      final Field modifiersField = Field.class.getDeclaredField("modifiers");
      modifiersField.setAccessible(true);
      modifiersField.setInt(field, field.getModifiers() & ~Modifier.FINAL);
      field.set(null, value);
    } catch (NoSuchFieldException | IllegalAccessException e) {
      throw new RuntimeException(e);
    }
  }
}
