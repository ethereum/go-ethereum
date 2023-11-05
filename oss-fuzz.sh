#!/bin/bash -eu
# Copyright 2022 Google LLC
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

function compile_fuzzer() {
  package=$1
  function=$2
  file=$3
  fuzzer=$4

  path=$GOPATH/src/$package

  echo "Building $fuzzer"
  cd $path

  # Install build dependencies
  go mod tidy
  go get github.com/holiman/gofuzz-shim/testing

	if [[ $SANITIZER == *coverage* ]]; then
		coverbuild $path $function $fuzzer $coverpkg
	else
	  gofuzz-shim --func $function --package $package -f $file -o $fuzzer.a
		$CXX $CXXFLAGS $LIB_FUZZING_ENGINE $fuzzer.a -o $OUT/$fuzzer
	fi

  ## Check if there exists a seed corpus file
  corpusfile="${path}/testdata/${fuzzer}_seed_corpus.zip"
  if [ -f $corpusfile ]
  then
    cp $corpusfile $OUT/
    echo "Found seed corpus: $corpusfile"
  fi
  cd -
}

go install github.com/holiman/gofuzz-shim@latest
repo=$GOPATH/src/github.com/ethereum/go-ethereum
# compile_fuzzer github.com/ethereum/go-ethereum/accounts/abi \
#   FuzzABI \
#   $repo/accounts/abi/abifuzzer_test.go \
#   fuzzAbi
#
# # compile_fuzzer github.com/ethereum/go-ethereum/common/bitutil FuzzEncoder fuzzBitutilEncoder
# compile_fuzzer github.com/ethereum/go-ethereum/common/bitutil \
#   FuzzEncoder \
#   $repo/common/bitutil/compress_test.go \
#   fuzzBitutilEncoder
#
# # compile_fuzzer github.com/ethereum/go-ethereum/common/bitutil FuzzDecoder fuzzBitutilDecoder
# compile_fuzzer github.com/ethereum/go-ethereum/common/bitutil \
#   FuzzDecoder \
#   $repo/common/bitutil/compress_test.go \
#   fuzzBitutilDecoder
#
# # compile_fuzzer github.com/ethereum/go-ethereum/core/vm/runtime FuzzVmRuntime fuzzVmRuntime
# compile_fuzzer github.com/ethereum/go-ethereum/core/vm/runtime \
#   FuzzVmRuntime \
#   $repo/core/vm/runtime/runtime_fuzz_test.go \
#   fuzzVmRuntime
#
# # compile_fuzzer github.com/ethereum/go-ethereum/core/vm FuzzPrecompiledContracts fuzzPrecompiledContracts
# compile_fuzzer github.com/ethereum/go-ethereum/core/vm \
#   FuzzPrecompiledContracts \
#   $repo/core/vm/contracts_fuzz_test.go \
#   fuzzPrecompiledContracts
# 
# # compile_fuzzer github.com/ethereum/go-ethereum/core/types FuzzRLP fuzzRlp
# compile_fuzzer github.com/ethereum/go-ethereum/core/types \
#   FuzzRLP \
#   $repo/core/types/rlp_fuzzer_test.go \
#   fuzzRlp
# 
# #compile_fuzzer github.com/ethereum/go-ethereum/crypto/blake2b  Fuzz      fuzzBlake2b
# compile_fuzzer github.com/ethereum/go-ethereum/crypto/blake2b \
#   Fuzz \
#   $repo/crypto/blake2b/blake2b_f_fuzz_test.go \
#   fuzzBlake2b
# 
# #compile_fuzzer github.com/ethereum/go-ethereum/accounts/keystore FuzzPassword fuzzKeystore
# compile_fuzzer github.com/ethereum/go-ethereum/accounts/keystore \
#   FuzzPassword \
#   $repo/accounts/keystore/keystore_fuzzing_test.go \
#   fuzzKeystore
# #
# #compile_fuzzer github.com/ethereum/go-ethereum/trie       FuzzTrie fuzzTrie
# pkg=$repo/trie/
# compile_fuzzer github.com/ethereum/go-ethereum/trie \
#   FuzzTrie \
#   $pkg/trie_test.go,$pkg/database_test.go,$pkg/tracer_test.go,$pkg/proof_test.go,$pkg/iterator_test.go,$pkg/sync_test.go \
#   fuzzTrie
#
# #compile_fuzzer github.com/ethereum/go-ethereum/trie       FuzzStackTrie fuzzStackTrie
# compile_fuzzer github.com/ethereum/go-ethereum/trie \
#   FuzzStackTrie \
#   $pkg/stacktrie_fuzzer_test.go,$pkg/iterator_test.go,$pkg/trie_test.go,$pkg/database_test.go,$pkg/tracer_test.go,$pkg/proof_test.go,$pkg/sync_test.go \
#   fuzzStackTrie
#
# #compile_fuzzer tests/fuzzers/snap  FuzzARange fuzz_account_range
# compile_fuzzer github.com/ethereum/go-ethereum/eth/protocols/snap \
#   FuzzARange \
#   $repo/eth/protocols/snap/handler_fuzzing_test.go \
#   fuzz_account_range
#
# #compile_fuzzer tests/fuzzers/snap  FuzzSRange fuzz_storage_range
# compile_fuzzer github.com/ethereum/go-ethereum/eth/protocols/snap \
#   FuzzSRange \
#   $repo/eth/protocols/snap/handler_fuzzing_test.go \
#   fuzz_storage_range
#
# #compile_fuzzer tests/fuzzers/snap  FuzzByteCodes fuzz_byte_codes
# compile_fuzzer github.com/ethereum/go-ethereum/eth/protocols/snap \
#   FuzzByteCodes \
#   $repo/eth/protocols/snap/handler_fuzzing_test.go \
#   fuzz_byte_codes
# #compile_fuzzer tests/fuzzers/snap  FuzzTrieNodes fuzz_trie_nodes
# compile_fuzzer github.com/ethereum/go-ethereum/eth/protocols/snap \
#   FuzzTrieNodes \
#   $repo/eth/protocols/snap/handler_fuzzing_test.go \
#   fuzz_trie_nodes
#
#
# #compile_fuzzer tests/fuzzers/bn256    FuzzAdd   fuzzBn256Add
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bn256 \
#   FuzzAdd \
#   $repo/tests/fuzzers/bn256/bn256_test.go  \
#   fuzzBn256Add
#
# #compile_fuzzer tests/fuzzers/bn256    FuzzMul   fuzzBn256Mul
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bn256 \
#   FuzzMul \
#   $repo/tests/fuzzers/bn256/bn256_test.go  \
#   fuzzBn256Mul
# #compile_fuzzer tests/fuzzers/bn256    FuzzPair  fuzzBn256Pair
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bn256 \
#   FuzzPair \
#   $repo/tests/fuzzers/bn256/bn256_test.go  \
#   fuzzBn256Pair
#
# #compile_fuzzer tests/fuzzers/txfetcher  Fuzz fuzzTxfetcher
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/txfetcher \
#   Fuzz \
#   $repo/tests/fuzzers/txfetcher/txfetcher_test.go  \
#   fuzzTxfetcher
#
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG1Add fuzz_g1_add
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG1Add \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g1_add
#
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG1Mul fuzz_g1_mul
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG1Mul \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g1_mul
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG1MultiExp fuzz_g1_multiexp
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG1MultiExp \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g1_multiexp
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG2Add fuzz_g2_add
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG2Add \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g2_add
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG2Mul fuzz_g2_mul
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG2Mul \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g2_mul
# #compile_fuzzer tests/fuzzers/bls12381  FuzzG2MultiExp fuzz_g2_multiexp
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzG2MultiExp \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_g2_multiexp
# #compile_fuzzer tests/fuzzers/bls12381  FuzzPairing fuzz_pairing
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzPairing \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_pairing
# #compile_fuzzer tests/fuzzers/bls12381  FuzzMapG1 fuzz_map_g1
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzMapG1 \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_map_g1
# #compile_fuzzer tests/fuzzers/bls12381  FuzzMapG2 fuzz_map_g2
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzMapG2 \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_map_g2
#
# #compile_fuzzer tests/fuzzers/bls12381  FuzzCrossG1Add fuzz_cross_g1_add
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzCrossG1Add \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_cross_g1_add
# #compile_fuzzer tests/fuzzers/bls12381  FuzzCrossG1MultiExp fuzz_cross_g1_multiexp
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzCrossG1MultiExp \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_cross_g1_multiexp
# #compile_fuzzer tests/fuzzers/bls12381  FuzzCrossG2Add fuzz_cross_g2_add
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzCrossG2Add \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_cross_g2_add
# #compile_fuzzer tests/fuzzers/bls12381  FuzzCrossPairing fuzz_cross_pairing
# compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/bls12381 \
#   FuzzCrossPairing \
#   $repo/tests/fuzzers/bls12381/bls12381_test.go  \
#   fuzz_cross_pairing
#
#compile_fuzzer tests/fuzzers/secp256k1  Fuzz fuzzSecp256k1
compile_fuzzer github.com/ethereum/go-ethereum/tests/fuzzers/secp256k1 \
  Fuzz \
  $repo/tests/fuzzers/secp256k1/secp_test.go  \
  fuzzSecp256k1

#compile_fuzzer tests/fuzzers/vflux      FuzzClientPool fuzzClientPool
#compile_fuzzer tests/fuzzers/difficulty Fuzz fuzzDifficulty
#compile_fuzzer tests/fuzzers/les        Fuzz fuzzLes

