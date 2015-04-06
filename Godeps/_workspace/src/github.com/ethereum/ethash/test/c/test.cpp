#include <iomanip>
#include <libethash/fnv.h>
#include <libethash/ethash.h>
#include <libethash/internal.h>
#include <libethash/io.h>

#ifdef WITH_CRYPTOPP

#include <libethash/sha3_cryptopp.h>

#else
#include <libethash/sha3.h>
#endif // WITH_CRYPTOPP

#define BOOST_TEST_MODULE Daggerhashimoto
#define BOOST_TEST_MAIN

#include <iostream>
#include <fstream>
#include <vector>
#include <boost/filesystem.hpp>
#include <boost/test/unit_test.hpp>

using namespace std;
namespace fs = boost::filesystem;

// Just an alloca "wrapper" to silence uint64_t to size_t conversion warnings in windows
// consider replacing alloca calls with something better though!
#define our_alloca(param__) alloca((size_t)(param__))

std::string bytesToHexString(const uint8_t *str, const uint64_t s) {
    std::ostringstream ret;

    for (size_t i = 0; i < s; ++i)
        ret << std::hex << std::setfill('0') << std::setw(2) << std::nouppercase << (int) str[i];

    return ret.str();
}

BOOST_AUTO_TEST_CASE(fnv_hash_check) {
    uint32_t x = 1235U;
    const uint32_t
            y = 9999999U,
            expected = (FNV_PRIME * x) ^y;

    x = fnv_hash(x, y);

    BOOST_REQUIRE_MESSAGE(x == expected,
            "\nexpected: " << expected << "\n"
                    << "actual: " << x << "\n");

}

BOOST_AUTO_TEST_CASE(SHA256_check) {
    uint8_t input[32], out[32];
    memcpy(input, "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~", 32);
    SHA3_256(out, input, 32);
    const std::string
            expected = "2b5ddf6f4d21c23de216f44d5e4bdc68e044b71897837ea74c83908be7037cd7",
            actual = bytesToHexString(out, 32);
    BOOST_REQUIRE_MESSAGE(expected == actual,
            "\nexpected: " << expected.c_str() << "\n"
                    << "actual: " << actual.c_str() << "\n");
}

BOOST_AUTO_TEST_CASE(SHA512_check) {
    uint8_t input[64], out[64];
    memcpy(input, "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~", 64);
    SHA3_512(out, input, 64);
    const std::string
            expected = "0be8a1d334b4655fe58c6b38789f984bb13225684e86b20517a55ab2386c7b61c306f25e0627c60064cecd6d80cd67a82b3890bd1289b7ceb473aad56a359405",
            actual = bytesToHexString(out, 64);
    BOOST_REQUIRE_MESSAGE(expected == actual,
            "\nexpected: " << expected.c_str() << "\n"
                    << "actual: " << actual.c_str() << "\n");
}

BOOST_AUTO_TEST_CASE(ethash_params_init_genesis_check) {
    ethash_params params;
    ethash_params_init(&params, 0);
    BOOST_REQUIRE_MESSAGE(params.full_size < DATASET_BYTES_INIT,
            "\nfull size: " << params.full_size << "\n"
                    << "should be less than or equal to: " << DATASET_BYTES_INIT << "\n");
    BOOST_REQUIRE_MESSAGE(params.full_size + 20 * MIX_BYTES >= DATASET_BYTES_INIT,
            "\nfull size + 20*MIX_BYTES: " << params.full_size + 20 * MIX_BYTES << "\n"
                    << "should be greater than or equal to: " << DATASET_BYTES_INIT << "\n");
    BOOST_REQUIRE_MESSAGE(params.cache_size < DATASET_BYTES_INIT / 32,
            "\ncache size: " << params.cache_size << "\n"
                    << "should be less than or equal to: " << DATASET_BYTES_INIT / 32 << "\n");
}

BOOST_AUTO_TEST_CASE(ethash_params_init_genesis_calcifide_check) {
    ethash_params params;
    ethash_params_init(&params, 0);
    const uint32_t expected_full_size = 1073739904;
    const uint32_t expected_cache_size = 16776896;
    BOOST_REQUIRE_MESSAGE(params.full_size == expected_full_size,
            "\nexpected: " << expected_cache_size << "\n"
                    << "actual: " << params.full_size << "\n");
    BOOST_REQUIRE_MESSAGE(params.cache_size == expected_cache_size,
            "\nexpected: " << expected_cache_size << "\n"
                    << "actual: " << params.cache_size << "\n");
}

BOOST_AUTO_TEST_CASE(light_and_full_client_checks) {
    ethash_params params;
    uint8_t seed[32], hash[32], difficulty[32];
    ethash_return_value light_out, full_out;
    memcpy(seed, "~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~", 32);
    memcpy(hash, "~~~X~~~~~~~~~~~~~~~~~~~~~~~~~~~~", 32);

    // Set the difficulty
    difficulty[0] = 197;
    difficulty[1] = 90;
    for (int i = 2; i < 32; i++)
        difficulty[i] = (uint8_t) 255;

    ethash_params_init(&params, 0);
    params.cache_size = 1024;
    params.full_size = 1024 * 32;
    ethash_cache cache;
    cache.mem = our_alloca(params.cache_size);
    ethash_mkcache(&cache, &params, seed);
    node *full_mem = (node *) our_alloca(params.full_size);
    ethash_compute_full_data(full_mem, &params, &cache);

    {
        const std::string
                expected = "2da2b506f21070e1143d908e867962486d6b0a02e31d468fd5e3a7143aafa76a14201f63374314e2a6aaf84ad2eb57105dea3378378965a1b3873453bb2b78f9a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c259440b89fa3481c2c33171477c305c8e1e421f8d8f6d59585449d0034f3e421808d8da6bbd0b6378f567647cc6c4ba6c434592b198ad444e7284905b7c6adaf70bf43ec2daa7bd5e8951aa609ab472c124cf9eba3d38cff5091dc3f58409edcc386c743c3bd66f92408796ee1e82dd149eaefbf52b00ce33014a6eb3e50625413b072a58bc01da28262f42cbe4f87d4abc2bf287d15618405a1fe4e386fcdafbb171064bd99901d8f81dd6789396ce5e364ac944bbbd75a7827291c70b42d26385910cd53ca535ab29433dd5c5714d26e0dce95514c5ef866329c12e958097e84462197c2b32087849dab33e88b11da61d52f9dbc0b92cc61f742c07dbbf751c49d7678624ee60dfbe62e5e8c47a03d8247643f3d16ad8c8e663953bcda1f59d7e2d4a9bf0768e789432212621967a8f41121ad1df6ae1fa78782530695414c6213942865b2730375019105cae91a4c17a558d4b63059661d9f108362143107babe0b848de412e4da59168cce82bfbff3c99e022dd6ac1e559db991f2e3f7bb910cefd173e65ed00a8d5d416534e2c8416ff23977dbf3eb7180b75c71580d08ce95efeb9b0afe904ea12285a392aff0c8561ff79fca67f694a62b9e52377485c57cc3598d84cac0a9d27960de0cc31ff9bbfe455acaa62c8aa5d2cce96f345da9afe843d258a99c4eaf3650fc62efd81c7b81cd0d534d2d71eeda7a6e315d540b4473c80f8730037dc2ae3e47b986240cfc65ccc565f0d8cde0bc68a57e39a271dda57440b3598bee19f799611d25731a96b5dbbbefdff6f4f656161462633030d62560ea4e9c161cf78fc96a2ca5aaa32453a6c5dea206f766244e8c9d9a8dc61185ce37f1fc804459c5f07434f8ecb34141b8dcae7eae704c950b55556c5f40140c3714b45eddb02637513268778cbf937a33e4e33183685f9deb31ef54e90161e76d969587dd782eaa94e289420e7c2ee908517f5893a26fdb5873d68f92d118d4bcf98d7a4916794d6ab290045e30f9ea00ca547c584b8482b0331ba1539a0f2714fddc3a0b06b0cfbb6a607b8339c39bcfd6640b1f653e9d70ef6c985b",
                actual = bytesToHexString((uint8_t const *) cache.mem, params.cache_size);

        BOOST_REQUIRE_MESSAGE(expected == actual,
                "\nexpected: " << expected.c_str() << "\n"
                        << "actual: " << actual.c_str() << "\n");
    }


    {
        node node;
        ethash_calculate_dag_item(&node, 0, &params, &cache);
        const std::string
                actual = bytesToHexString((uint8_t const *) &node, sizeof(node)),
                expected = "b1698f829f90b35455804e5185d78f549fcb1bdce2bee006d4d7e68eb154b596be1427769eb1c3c3e93180c760af75f81d1023da6a0ffbe321c153a7c0103597";
        BOOST_REQUIRE_MESSAGE(actual == expected,
                "\n" << "expected: " << expected.c_str() << "\n"
                        << "actual: " << actual.c_str() << "\n");
    }

    {
        for (int i = 0; i < params.full_size / sizeof(node); ++i) {
            for (uint32_t j = 0; j < 32; ++j) {
                node expected_node;
                ethash_calculate_dag_item(&expected_node, j, &params, &cache);
                const std::string
                        actual = bytesToHexString((uint8_t const *) &(full_mem[j]), sizeof(node)),
                        expected = bytesToHexString((uint8_t const *) &expected_node, sizeof(node));
                BOOST_REQUIRE_MESSAGE(actual == expected,
                        "\ni: " << j << "\n"
                                << "expected: " << expected.c_str() << "\n"
                                << "actual: " << actual.c_str() << "\n");
            }
        }
    }

    {
        uint64_t nonce = 0x7c7c597c;
        ethash_full(&full_out, full_mem, &params, hash, nonce);
        ethash_light(&light_out, &cache, &params, hash, nonce);
        const std::string
                light_result_string = bytesToHexString(light_out.result, 32),
                full_result_string = bytesToHexString(full_out.result, 32);
        BOOST_REQUIRE_MESSAGE(light_result_string == full_result_string,
                "\nlight result: " << light_result_string.c_str() << "\n"
                        << "full result: " << full_result_string.c_str() << "\n");
        const std::string
                light_mix_hash_string = bytesToHexString(light_out.mix_hash, 32),
                full_mix_hash_string = bytesToHexString(full_out.mix_hash, 32);
        BOOST_REQUIRE_MESSAGE(full_mix_hash_string == light_mix_hash_string,
                "\nlight mix hash: " << light_mix_hash_string.c_str() << "\n"
                        << "full mix hash: " << full_mix_hash_string.c_str() << "\n");
        uint8_t check_hash[32];
        ethash_quick_hash(check_hash, hash, nonce, full_out.mix_hash);
        const std::string check_hash_string = bytesToHexString(check_hash, 32);
        BOOST_REQUIRE_MESSAGE(check_hash_string == full_result_string,
                "\ncheck hash string: " << check_hash_string.c_str() << "\n"
                        << "full result: " << full_result_string.c_str() << "\n");
    }
    {
        ethash_full(&full_out, full_mem, &params, hash, 5);
        std::string
                light_result_string = bytesToHexString(light_out.result, 32),
                full_result_string = bytesToHexString(full_out.result, 32);

        BOOST_REQUIRE_MESSAGE(light_result_string != full_result_string,
                "\nlight result and full result should differ: " << light_result_string.c_str() << "\n");

        ethash_light(&light_out, &cache, &params, hash, 5);
        light_result_string = bytesToHexString(light_out.result, 32);
        BOOST_REQUIRE_MESSAGE(light_result_string == full_result_string,
                "\nlight result and full result should be the same\n"
                        << "light result: " << light_result_string.c_str() << "\n"
                        << "full result: " << full_result_string.c_str() << "\n");
        std::string
                light_mix_hash_string = bytesToHexString(light_out.mix_hash, 32),
                full_mix_hash_string = bytesToHexString(full_out.mix_hash, 32);
        BOOST_REQUIRE_MESSAGE(full_mix_hash_string == light_mix_hash_string,
                "\nlight mix hash: " << light_mix_hash_string.c_str() << "\n"
                        << "full mix hash: " << full_mix_hash_string.c_str() << "\n");
        BOOST_REQUIRE_MESSAGE(ethash_check_difficulty(full_out.result, difficulty),
                "ethash_check_difficulty failed"
        );
        BOOST_REQUIRE_MESSAGE(ethash_quick_check_difficulty(hash, 5U, full_out.mix_hash, difficulty),
                "ethash_quick_check_difficulty failed"
        );
    }
}

BOOST_AUTO_TEST_CASE(ethash_check_difficulty_check) {
    uint8_t hash[32], target[32];
    memset(hash, 0, 32);
    memset(target, 0, 32);

    memcpy(hash, "11111111111111111111111111111111", 32);
    memcpy(target, "22222222222222222222222222222222", 32);
    BOOST_REQUIRE_MESSAGE(
            ethash_check_difficulty(hash, target),
            "\nexpected \"" << std::string((char *) hash, 32).c_str() << "\" to have the same or less difficulty than \"" << std::string((char *) target, 32).c_str() << "\"\n");
    BOOST_REQUIRE_MESSAGE(
            ethash_check_difficulty(hash, hash),
            "\nexpected \"" << hash << "\" to have the same or less difficulty than \"" << hash << "\"\n");
    memcpy(target, "11111111111111111111111111111112", 32);
    BOOST_REQUIRE_MESSAGE(
            ethash_check_difficulty(hash, target),
            "\nexpected \"" << hash << "\" to have the same or less difficulty than \"" << target << "\"\n");
    memcpy(target, "11111111111111111111111111111110", 32);
    BOOST_REQUIRE_MESSAGE(
            !ethash_check_difficulty(hash, target),
            "\nexpected \"" << hash << "\" to have more difficulty than \"" << target << "\"\n");
}

BOOST_AUTO_TEST_CASE(test_ethash_dir_creation) {
    ethash_blockhash_t seedhash;
    memset(&seedhash, 0, 32);
    BOOST_REQUIRE_EQUAL(
        ETHASH_IO_MEMO_MISMATCH,
        ethash_io_prepare("./test_ethash_directory/", seedhash)
    );

    // let's make sure that the directory was created
    BOOST_REQUIRE(fs::is_directory(fs::path("./test_ethash_directory/")));

    // cleanup
    fs::remove_all("./test_ethash_directory/");
}

BOOST_AUTO_TEST_CASE(test_ethash_io_write_files_are_created) {
    ethash_blockhash_t seedhash;
    static const int blockn = 0;
    ethash_get_seedhash((uint8_t*)&seedhash, blockn);
    BOOST_REQUIRE_EQUAL(
        ETHASH_IO_MEMO_MISMATCH,
        ethash_io_prepare("./test_ethash_directory/", seedhash)
    );

 // let's make sure that the directory was created
    BOOST_REQUIRE(fs::is_directory(fs::path("./test_ethash_directory/")));

    ethash_cache cache;
    ethash_params params;
    uint8_t *data;
    uint64_t size;
    ethash_params_init(&params, blockn);
    params.cache_size = 1024;
    params.full_size = 1024 * 32;
    cache.mem = our_alloca(params.cache_size);
    ethash_mkcache(&cache, &params, (uint8_t*)&seedhash);

    BOOST_REQUIRE(
        ethash_io_write("./test_ethash_directory/", &params, seedhash, &cache, &data, &size)
    );

    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full")));
    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full.info")));

    // cleanup
    fs::remove_all("./test_ethash_directory/");
    free(data);
}

BOOST_AUTO_TEST_CASE(test_ethash_io_memo_file_match) {
    ethash_blockhash_t seedhash;
    static const int blockn = 0;
    ethash_get_seedhash((uint8_t*)&seedhash, blockn);
    BOOST_REQUIRE_EQUAL(
        ETHASH_IO_MEMO_MISMATCH,
        ethash_io_prepare("./test_ethash_directory/", seedhash)
    );

    // let's make sure that the directory was created
    BOOST_REQUIRE(fs::is_directory(fs::path("./test_ethash_directory/")));

    ethash_cache cache;
    ethash_params params;
    uint8_t *data;
    uint64_t size;
    ethash_params_init(&params, blockn);
    params.cache_size = 1024;
    params.full_size = 1024 * 32;
    cache.mem = our_alloca(params.cache_size);
    ethash_mkcache(&cache, &params, (uint8_t*)&seedhash);

    BOOST_REQUIRE(
        ethash_io_write("./test_ethash_directory/", &params, seedhash, &cache, &data, &size)
    );

    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full")));
    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full.info")));

    BOOST_REQUIRE_EQUAL(
        ETHASH_IO_MEMO_MATCH,
        ethash_io_prepare("./test_ethash_directory/", seedhash)
    );

    // cleanup
    fs::remove_all("./test_ethash_directory/");
    free(data);
}

// could have used dev::contentsNew but don't wanna try to import
// libdevcore just for one function
static std::vector<char> readFileIntoVector(char const* filename)
{
    ifstream ifs(filename, ios::binary|ios::ate);
    ifstream::pos_type pos = ifs.tellg();

    std::vector<char> result((unsigned int)pos);

    ifs.seekg(0, ios::beg);
    ifs.read(&result[0], pos);

    return result;
}

BOOST_AUTO_TEST_CASE(test_ethash_io_memo_file_contents) {
    ethash_blockhash_t seedhash;
    static const int blockn = 0;
    ethash_get_seedhash((uint8_t*)&seedhash, blockn);
    BOOST_REQUIRE_EQUAL(
        ETHASH_IO_MEMO_MISMATCH,
        ethash_io_prepare("./test_ethash_directory/", seedhash)
    );

    // let's make sure that the directory was created
    BOOST_REQUIRE(fs::is_directory(fs::path("./test_ethash_directory/")));

    ethash_cache cache;
    ethash_params params;
    uint8_t *data;
    uint64_t size;
    ethash_params_init(&params, blockn);
    params.cache_size = 1024;
    params.full_size = 1024 * 32;
    cache.mem = our_alloca(params.cache_size);
    ethash_mkcache(&cache, &params, (uint8_t*)&seedhash);

    BOOST_REQUIRE(
        ethash_io_write("./test_ethash_directory/", &params, seedhash, &cache, &data, &size)
    );

    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full")));
    BOOST_REQUIRE(fs::exists(fs::path("./test_ethash_directory/full.info")));

    char expect_buffer[DAG_MEMO_BYTESIZE];
    ethash_io_serialize_info(REVISION, seedhash, expect_buffer);
    auto vec = readFileIntoVector("./test_ethash_directory/full.info");
    BOOST_REQUIRE_EQUAL(vec.size(), DAG_MEMO_BYTESIZE);
    BOOST_REQUIRE(memcmp(expect_buffer, &vec[0], DAG_MEMO_BYTESIZE) == 0);

    // cleanup
    fs::remove_all("./test_ethash_directory/");
    free(data);
}
