#/bin/bash -eu
# Copyright 2020 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
################################################################################

# This file is for integration with Google OSS-Fuzz.
# The following ENV variables are available when executing on OSS-fuzz:
#
# /out/         $OUT    Directory to store build artifacts (fuzz targets, dictionaries, options files, seed corpus archives).
# /src/         $SRC    Directory to checkout source files.
# /work/        $WORK   Directory to store intermediate files.
#
# $CC, $CXX, $CCC       The C and C++ compiler binaries.
# $CFLAGS, $CXXFLAGS    C and C++ compiler flags.
# $LIB_FUZZING_ENGINE   C++ compiler argument to link fuzz target against the prebuilt engine library (e.g. libFuzzer).

function compile_fuzzer {
  path=$SRC/go-ethereum/$1
  func=$2
  fuzzer=$3
  corpusfile="${path}/testdata/${fuzzer}_seed_corpus.zip"
  echo "Building $fuzzer (expecting corpus at $corpusfile)"
  (cd $path && \
        go-fuzz -func $func -o $WORK/$fuzzer.a . && \
        echo "First stage built OK" && \
        $CXX $CXXFLAGS $LIB_FUZZING_ENGINE $WORK/$fuzzer.a -o $OUT/$fuzzer && \
        echo "Second stage built ok" )

        ## Check if there exists a seed corpus file
        if [ -f $corpusfile ]
        then
          cp $corpusfile $OUT/
          echo "Found seed corpus: $corpusfile"
        fi
}

compile_fuzzer common/bitutil  Fuzz      fuzzBitutilCompress
compile_fuzzer crypto/bn256    FuzzAdd   fuzzBn256Add
compile_fuzzer crypto/bn256    FuzzMul   fuzzBn256Mul
compile_fuzzer crypto/bn256    FuzzPair  fuzzBn256Pair
compile_fuzzer core/vm/runtime Fuzz      fuzzVmRuntime
compile_fuzzer crypto/blake2b  Fuzz      fuzzBlake2b
compile_fuzzer tests/fuzzers/keystore   Fuzz fuzzKeystore
compile_fuzzer tests/fuzzers/txfetcher  Fuzz fuzzTxfetcher
compile_fuzzer tests/fuzzers/rlp        Fuzz fuzzRlp
compile_fuzzer tests/fuzzers/trie       Fuzz fuzzTrie
compile_fuzzer tests/fuzzers/stacktrie  Fuzz fuzzStackTrie

compile_fuzzer tests/fuzzers/bls12381  FuzzG1Add fuzz_g1_add
compile_fuzzer tests/fuzzers/bls12381  FuzzG1Mul fuzz_g1_mul
compile_fuzzer tests/fuzzers/bls12381  FuzzG1MultiExp fuzz_g1_multiexp
compile_fuzzer tests/fuzzers/bls12381  FuzzG2Add fuzz_g2_add
compile_fuzzer tests/fuzzers/bls12381  FuzzG2Mul fuzz_g2_mul
compile_fuzzer tests/fuzzers/bls12381  FuzzG2MultiExp fuzz_g2_multiexp
compile_fuzzer tests/fuzzers/bls12381  FuzzPairing fuzz_pairing
compile_fuzzer tests/fuzzers/bls12381  FuzzMapG1 fuzz_map_g1
compile_fuzzer tests/fuzzers/bls12381  FuzzMapG2 fuzz_map_g2

# This doesn't work very well @TODO
#compile_fuzzertests/fuzzers/abi Fuzz fuzzAbi

