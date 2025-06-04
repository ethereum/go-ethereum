#include <secp256k1.h>
#include <secp256k1/include/secp256k1_ecdh.h>
#include <secp256k1/include/secp256k1_preallocated.h>
#include <secp256k1/include/secp256k1_recovery.h>

// Local helpers
#define RETURN(result) return Napi::Number::New(info.Env(), result)

#define RETURN_INVERTED(result) RETURN(result == 1 ? 0 : 1)

#define RETURN_IF_ZERO(result, retcode)                                        \
  do {                                                                         \
    if (result == 0) {                                                         \
      RETURN(retcode);                                                         \
    }                                                                          \
  } while (0)

#define PUBKEY_SERIALIZE(retcode)                                              \
  do {                                                                         \
    size_t outputlen = output.Length();                                        \
    int flags =                                                                \
        outputlen == 33 ? SECP256K1_EC_COMPRESSED : SECP256K1_EC_UNCOMPRESSED; \
    RETURN_IF_ZERO(secp256k1_ec_pubkey_serialize(                              \
                       this->ctx_, output.Data(), &outputlen, &pubkey, flags), \
                   retcode);                                                   \
  } while (0)

// Secp256k1
Napi::FunctionReference Secp256k1Addon::constructor;
unsigned int Secp256k1Addon::secp256k1_context_flags =
    SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY;

Napi::Value Secp256k1Addon::Init(Napi::Env env) {
  Napi::Function func = DefineClass(
      env,
      "Secp256k1Addon",
      {
          InstanceMethod("contextRandomize", &Secp256k1Addon::ContextRandomize),

          InstanceMethod("privateKeyVerify", &Secp256k1Addon::PrivateKeyVerify),
          InstanceMethod("privateKeyNegate", &Secp256k1Addon::PrivateKeyNegate),
          InstanceMethod("privateKeyTweakAdd",
                         &Secp256k1Addon::PrivateKeyTweakAdd),
          InstanceMethod("privateKeyTweakMul",
                         &Secp256k1Addon::PrivateKeyTweakMul),

          InstanceMethod("publicKeyVerify", &Secp256k1Addon::PublicKeyVerify),
          InstanceMethod("publicKeyCreate", &Secp256k1Addon::PublicKeyCreate),
          InstanceMethod("publicKeyConvert", &Secp256k1Addon::PublicKeyConvert),
          InstanceMethod("publicKeyNegate", &Secp256k1Addon::PublicKeyNegate),
          InstanceMethod("publicKeyCombine", &Secp256k1Addon::PublicKeyCombine),
          InstanceMethod("publicKeyTweakAdd",
                         &Secp256k1Addon::PublicKeyTweakAdd),
          InstanceMethod("publicKeyTweakMul",
                         &Secp256k1Addon::PublicKeyTweakMul),

          InstanceMethod("signatureNormalize",
                         &Secp256k1Addon::SignatureNormalize),
          InstanceMethod("signatureExport", &Secp256k1Addon::SignatureExport),
          InstanceMethod("signatureImport", &Secp256k1Addon::SignatureImport),

          InstanceMethod("ecdsaSign", &Secp256k1Addon::ECDSASign),
          InstanceMethod("ecdsaVerify", &Secp256k1Addon::ECDSAVerify),
          InstanceMethod("ecdsaRecover", &Secp256k1Addon::ECDSARecover),

          InstanceMethod("ecdh", &Secp256k1Addon::ECDH),
      });

  constructor = Napi::Persistent(func);
  constructor.SuppressDestruct();

  return func;
}

Secp256k1Addon::Secp256k1Addon(const Napi::CallbackInfo& info)
    : Napi::ObjectWrap<Secp256k1Addon>(info) {
  ctx_ = secp256k1_context_create(secp256k1_context_flags);

  size_t size = secp256k1_context_preallocated_size(secp256k1_context_flags);
  Napi::MemoryManagement::AdjustExternalMemory(info.Env(), size);
}

void Secp256k1Addon::Finalize(Napi::Env env) {
  secp256k1_context_destroy(const_cast<secp256k1_context*>(ctx_));

  size_t size = secp256k1_context_preallocated_size(secp256k1_context_flags);
  Napi::MemoryManagement::AdjustExternalMemory(env, -size);
}

Napi::Value Secp256k1Addon::ContextRandomize(const Napi::CallbackInfo& info) {
  const unsigned char* seed32 = NULL;
  if (!info[0].IsNull()) {
    seed32 = info[0].As<Napi::Buffer<const unsigned char>>().Data();
  }

  RETURN_INVERTED(secp256k1_context_randomize(
      const_cast<secp256k1_context*>(this->ctx_), seed32));
}

// PrivateKey
Napi::Value Secp256k1Addon::PrivateKeyVerify(const Napi::CallbackInfo& info) {
  auto seckey = info[0].As<Napi::Buffer<const unsigned char>>().Data();

  RETURN_INVERTED(secp256k1_ec_seckey_verify(this->ctx_, seckey));
}

Napi::Value Secp256k1Addon::PrivateKeyNegate(const Napi::CallbackInfo& info) {
  auto seckey = info[0].As<Napi::Buffer<unsigned char>>().Data();

  RETURN_IF_ZERO(secp256k1_ec_privkey_negate(this->ctx_, seckey), 1);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PrivateKeyTweakAdd(const Napi::CallbackInfo& info) {
  auto seckey = info[0].As<Napi::Buffer<unsigned char>>().Data();
  auto tweak = info[1].As<Napi::Buffer<const unsigned char>>().Data();

  RETURN_INVERTED(secp256k1_ec_privkey_tweak_add(this->ctx_, seckey, tweak));
}

Napi::Value Secp256k1Addon::PrivateKeyTweakMul(const Napi::CallbackInfo& info) {
  auto seckey = info[0].As<Napi::Buffer<unsigned char>>().Data();
  auto tweak = info[1].As<Napi::Buffer<const unsigned char>>().Data();

  RETURN_INVERTED(secp256k1_ec_privkey_tweak_mul(this->ctx_, seckey, tweak));
}

// PublicKey
Napi::Value Secp256k1Addon::PublicKeyVerify(const Napi::CallbackInfo& info) {
  auto input = info[0].As<Napi::Buffer<const unsigned char>>();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyCreate(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto seckey = info[1].As<Napi::Buffer<const unsigned char>>().Data();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_create(this->ctx_, &pubkey, seckey), 1);
  PUBKEY_SERIALIZE(2);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyConvert(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  PUBKEY_SERIALIZE(2);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyNegate(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  RETURN_IF_ZERO(secp256k1_ec_pubkey_negate(this->ctx_, &pubkey), 2);
  PUBKEY_SERIALIZE(3);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyCombine(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto inputs = info[1].As<Napi::Array>();

  std::unique_ptr<secp256k1_pubkey[]> pubkeys(
      new secp256k1_pubkey[inputs.Length()]);
  std::unique_ptr<secp256k1_pubkey*[]> ptrs(
      new secp256k1_pubkey*[inputs.Length()]);
  for (unsigned int i = 0; i < inputs.Length(); ++i) {
    auto input = inputs.Get(i).As<Napi::Buffer<const unsigned char>>();
    RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                       this->ctx_, &pubkeys[i], input.Data(), input.Length()),
                   1);
    ptrs[i] = &pubkeys[i];
  }

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_combine(
                     this->ctx_, &pubkey, ptrs.get(), inputs.Length()),
                 2);
  PUBKEY_SERIALIZE(3);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyTweakAdd(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();
  auto tweak = info[2].As<Napi::Buffer<const unsigned char>>().Data();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  RETURN_IF_ZERO(secp256k1_ec_pubkey_tweak_add(this->ctx_, &pubkey, tweak), 2);
  PUBKEY_SERIALIZE(3);
  RETURN(0);
}

Napi::Value Secp256k1Addon::PublicKeyTweakMul(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();
  auto tweak = info[2].As<Napi::Buffer<const unsigned char>>().Data();

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  RETURN_IF_ZERO(secp256k1_ec_pubkey_tweak_mul(this->ctx_, &pubkey, tweak), 2);
  PUBKEY_SERIALIZE(3);
  RETURN(0);
}

// Signature
Napi::Value Secp256k1Addon::SignatureNormalize(const Napi::CallbackInfo& info) {
  auto sig = info[0].As<Napi::Buffer<unsigned char>>().Data();

  secp256k1_ecdsa_signature sigin, sigout;
  RETURN_IF_ZERO(
      secp256k1_ecdsa_signature_parse_compact(this->ctx_, &sigin, sig), 1);
  secp256k1_ecdsa_signature_normalize(this->ctx_, &sigout, &sigin);
  secp256k1_ecdsa_signature_serialize_compact(this->ctx_, sig, &sigout);
  RETURN(0);
}

Napi::Value Secp256k1Addon::SignatureExport(const Napi::CallbackInfo& info) {
  auto obj = info[0].As<Napi::Object>();
  auto output = obj.Get("output").As<Napi::Buffer<unsigned char>>().Data();
  size_t outputlen = 72;
  auto input = info[1].As<Napi::Buffer<const unsigned char>>().Data();

  secp256k1_ecdsa_signature sig;
  RETURN_IF_ZERO(
      secp256k1_ecdsa_signature_parse_compact(this->ctx_, &sig, input), 1);
  RETURN_IF_ZERO(secp256k1_ecdsa_signature_serialize_der(
                     this->ctx_, output, &outputlen, &sig),
                 2);

  obj.Set("outputlen", outputlen);
  RETURN(0);
}

Napi::Value Secp256k1Addon::SignatureImport(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>().Data();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();

  secp256k1_ecdsa_signature sig;
  RETURN_IF_ZERO(secp256k1_ecdsa_signature_parse_der(
                     this->ctx_, &sig, input.Data(), input.Length()),
                 1);
  RETURN_IF_ZERO(
      secp256k1_ecdsa_signature_serialize_compact(this->ctx_, output, &sig), 2);
  RETURN(0);
}

// ECDSA
int ecdsa_sign_nonce_function(unsigned char* nonce32,
                              const unsigned char* msg32,
                              const unsigned char* key32,
                              const unsigned char* algo16,
                              void* data,
                              unsigned int counter) {
  auto obj = static_cast<Secp256k1Addon::ECDSASignData*>(data);
  auto env = Napi::Env(obj->env);

  auto result = obj->fn.Call({obj->msg32,
                              obj->key32,
                              env.Null(),
                              obj->data,
                              Napi::Number::New(env, counter)});
  if (!result.IsTypedArray()) return 0;
  if (result.As<Napi::Uint8Array>().ByteLength() != 32) return 0;

  memcpy(nonce32, result.As<Napi::Uint8Array>().Data(), 32);
  return 1;
}

Napi::Value Secp256k1Addon::ECDSASign(const Napi::CallbackInfo& info) {
  auto obj = info[0].As<Napi::Object>();
  auto output = obj.Get("signature").As<Napi::Buffer<unsigned char>>().Data();
  int recid;
  auto msg32 = info[1].As<Napi::Buffer<unsigned char>>().Data();
  auto seckey = info[2].As<Napi::Buffer<const unsigned char>>().Data();

  void* data = NULL;
  if (!info[3].IsUndefined()) {
    data = info[3].As<Napi::Buffer<unsigned char>>().Data();
  }

  secp256k1_nonce_function noncefn = secp256k1_nonce_function_rfc6979;
  if (!info[4].IsUndefined()) {
    this->ecdsa_sign_data.env = info.Env();
    this->ecdsa_sign_data.fn = info[4].As<Napi::Function>();
    this->ecdsa_sign_data.msg32 = info[1];
    this->ecdsa_sign_data.key32 = info[2];
    this->ecdsa_sign_data.data =
        info[3].IsUndefined() ? info.Env().Null() : info[3];

    noncefn = ecdsa_sign_nonce_function;
    data = static_cast<void*>(&this->ecdsa_sign_data);
  }

  secp256k1_ecdsa_recoverable_signature sig;
  RETURN_IF_ZERO(secp256k1_ecdsa_sign_recoverable(
                     this->ctx_, &sig, msg32, seckey, noncefn, data),
                 1);

  RETURN_IF_ZERO(secp256k1_ecdsa_recoverable_signature_serialize_compact(
                     this->ctx_, output, &recid, &sig),
                 2);

  obj.Set("recid", recid);
  RETURN(0);
}

Napi::Value Secp256k1Addon::ECDSAVerify(const Napi::CallbackInfo& info) {
  auto sigraw = info[0].As<Napi::Buffer<const unsigned char>>().Data();
  auto msg32 = info[1].As<Napi::Buffer<const unsigned char>>().Data();
  auto input = info[2].As<Napi::Buffer<const unsigned char>>();

  secp256k1_ecdsa_signature sig;
  RETURN_IF_ZERO(
      secp256k1_ecdsa_signature_parse_compact(this->ctx_, &sig, sigraw), 1);

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 2);

  RETURN_IF_ZERO(secp256k1_ecdsa_verify(this->ctx_, &sig, msg32, &pubkey), 3);
  RETURN(0);
}

Napi::Value Secp256k1Addon::ECDSARecover(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto sigraw = info[1].As<Napi::Buffer<const unsigned char>>().Data();
  auto recid = info[2].As<Napi::Number>().Int32Value();
  auto msg32 = info[3].As<Napi::Buffer<const unsigned char>>().Data();

  secp256k1_ecdsa_recoverable_signature sig;
  RETURN_IF_ZERO(secp256k1_ecdsa_recoverable_signature_parse_compact(
                     this->ctx_, &sig, sigraw, recid),
                 1);

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ecdsa_recover(this->ctx_, &pubkey, &sig, msg32), 2);

  PUBKEY_SERIALIZE(3);
  RETURN(0);
}

// ECDH
int ecdh_hash_function(unsigned char* output,
                       const unsigned char* x,
                       const unsigned char* y,
                       void* data) {
  auto obj = static_cast<Secp256k1Addon::ECDHData*>(data);

  memcpy(obj->xbuf.As<Napi::Uint8Array>().Data(), x, 32);
  memcpy(obj->ybuf.As<Napi::Uint8Array>().Data(), y, 32);

  auto result = obj->fn.Call({obj->xbuf, obj->ybuf, obj->data});
  if (!result.IsTypedArray()) return 0;
  if (result.As<Napi::Uint8Array>().ByteLength() != obj->outputlen) return 0;

  memcpy(output, result.As<Napi::Uint8Array>().Data(), obj->outputlen);
  return 1;
}

Napi::Value Secp256k1Addon::ECDH(const Napi::CallbackInfo& info) {
  auto output = info[0].As<Napi::Buffer<unsigned char>>();
  auto input = info[1].As<Napi::Buffer<const unsigned char>>();
  auto seckey = info[2].As<Napi::Buffer<const unsigned char>>().Data();

  void* data = NULL;
  if (!info[3].IsUndefined()) {
    data = info[3].As<Napi::Buffer<unsigned char>>().Data();
  }

  secp256k1_ecdh_hash_function hashfn = secp256k1_ecdh_hash_function_sha256;
  if (!info[4].IsUndefined()) {
    auto env = info.Env();
    this->ecdh_data.fn = info[4].As<Napi::Function>();
    this->ecdh_data.xbuf =
        info[5].IsUndefined() ? Napi::Uint8Array::New(env, 32) : info[5];
    this->ecdh_data.ybuf =
        info[6].IsUndefined() ? Napi::Uint8Array::New(env, 32) : info[6];
    this->ecdh_data.data = info[3].IsUndefined() ? env.Null() : info[3];
    this->ecdh_data.outputlen = output.Length();

    hashfn = ecdh_hash_function;
    data = static_cast<void*>(&this->ecdh_data);
  }

  secp256k1_pubkey pubkey;
  RETURN_IF_ZERO(secp256k1_ec_pubkey_parse(
                     this->ctx_, &pubkey, input.Data(), input.Length()),
                 1);
  RETURN_IF_ZERO(
      secp256k1_ecdh(this->ctx_, output.Data(), &pubkey, seckey, hashfn, data),
      2);
  RETURN(0);
}
