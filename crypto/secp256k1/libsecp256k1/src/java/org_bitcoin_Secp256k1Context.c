#include <stdlib.h>
#include <stdint.h>
#include "org_bitcoin_Secp256k1Context.h"
#include "include/secp256k1.h"

SECP256K1_API jlong JNICALL Java_org_bitcoin_Secp256k1Context_secp256k1_1init_1context
  (JNIEnv* env, jclass classObject)
{
    secp256k1_context *ctx = secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);

    if (ctx == NULL) {
        (*env)->ThrowNew(env, (*env)->FindClass(env, "java/lang/RuntimeException"), "Failed to initialize secp256k1 context");
        return 0;
    }

    (void)classObject; (void)env;
    return (jlong)(intptr_t)ctx;
}

SECP256K1_API void JNICALL Java_org_bitcoin_Secp256k1Context_secp256k1_1destroy_1context
  (JNIEnv* env, jclass classObject, jlong ctx_l)
{
    secp256k1_context *ctx = (secp256k1_context*) (intptr_t) ctx_l;
    secp256k1_context_destroy(ctx);

    (void)classObject; (void)env;
}