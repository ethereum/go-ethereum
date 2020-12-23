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

# This sets the -coverpgk for the coverage report when the corpus is executed through go test
coverpkg="github.com/ethereum/go-ethereum/..."

function coverbuild {
  path=$1
  function=$2
  fuzzer=$3
  tags=""

  if [[ $#  -eq 4 ]]; then
    tags="-tags $4"
  fi
  cd $path
  fuzzed_package=`pwd | rev | cut -d'/' -f 1 | rev`
  cp $GOPATH/ossfuzz_coverage_runner.go ./"${function,,}"_test.go
  sed -i -e 's/FuzzFunction/'$function'/' ./"${function,,}"_test.go
  sed -i -e 's/mypackagebeingfuzzed/'$fuzzed_package'/' ./"${function,,}"_test.go
  sed -i -e 's/TestFuzzCorpus/Test'$function'Corpus/' ./"${function,,}"_test.go

cat << DOG > $OUT/$fuzzer
#/bin/sh

  cd $OUT/$path
  go test -run Test${function}Corpus -v $tags -coverprofile \$1 -coverpkg $coverpkg

DOG

  chmod +x $OUT/$fuzzer
  #echo "Built script $OUT/$fuzzer"
  #cat $OUT/$fuzzer
  cd -
}

function compile_fuzzer {
  # Inputs:
  # $1: The package to fuzz, within go-ethereum
  # $2: The name of the fuzzing function
  # $3: The name to give to the final fuzzing-binary

  path=$GOPATH/src/github.com/ethereum/go-ethereum/$1
  func=$2
  fuzzer=$3

  echo "Building $fuzzer"

  # Do a coverage-build or a regular build
  if [[ $SANITIZER = *coverage* ]]; then
    coverbuild $path $func $fuzzer $coverpkg
  else
    (cd $path && \
        go-fuzz -func $func -o $WORK/$fuzzer.a . && \
        $CXX $CXXFLAGS $LIB_FUZZING_ENGINE $WORK/$fuzzer.a -o $OUT/$fuzzer)
  fi

  ## Check if there exists a seed corpus file
  corpusfile="${path}/testdata/${fuzzer}_seed_corpus.zip"
  if [ -f $corpusfile ]
  then
    cp $corpusfile $OUT/
    echo "Found seed corpus: $corpusfile"
  fi
}

compile_fuzzer tests/fuzzers/bitutil  Fuzz      fuzzBitutilCompress
compile_fuzzer tests/fuzzers/bn256    FuzzAdd   fuzzBn256Add
compile_fuzzer tests/fuzzers/bn256    FuzzMul   fuzzBn256Mul
compile_fuzzer tests/fuzzers/bn256    FuzzPair  fuzzBn256Pair
compile_fuzzer tests/fuzzers/runtime  Fuzz      fuzzVmRuntime
compile_fuzzer tests/fuzzers/keystore   Fuzz fuzzKeystore
compile_fuzzer tests/fuzzers/txfetcher  Fuzz fuzzTxfetcher
compile_fuzzer tests/fuzzers/rlp        Fuzz fuzzRlp
compile_fuzzer tests/fuzzers/trie       Fuzz fuzzTrie
compile_fuzzer tests/fuzzers/stacktrie  Fuzz fuzzStackTrie
compile_fuzzer tests/fuzzers/difficulty Fuzz fuzzDifficulty

compile_fuzzer tests/fuzzers/bls12381  FuzzG1Add fuzz_g1_add
compile_fuzzer tests/fuzzers/bls12381  FuzzG1Mul fuzz_g1_mul
compile_fuzzer tests/fuzzers/bls12381  FuzzG1MultiExp fuzz_g1_multiexp
compile_fuzzer tests/fuzzers/bls12381  FuzzG2Add fuzz_g2_add
compile_fuzzer tests/fuzzers/bls12381  FuzzG2Mul fuzz_g2_mul
compile_fuzzer tests/fuzzers/bls12381  FuzzG2MultiExp fuzz_g2_multiexp
compile_fuzzer tests/fuzzers/bls12381  FuzzPairing fuzz_pairing
compile_fuzzer tests/fuzzers/bls12381  FuzzMapG1 fuzz_map_g1
compile_fuzzer tests/fuzzers/bls12381  FuzzMapG2 fuzz_map_g2

#TODO: move this to tests/fuzzers, if possible
compile_fuzzer crypto/blake2b  Fuzz      fuzzBlake2b


# This doesn't work very well @TODO
#compile_fuzzertests/fuzzers/abi Fuzz fuzzAbi

