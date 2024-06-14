#ifndef ADDON_SECP256K1
#define ADDON_SECP256K1

#include <napi.h>
#include <secp256k1/include/secp256k1.h>

class Secp256k1Addon : public Napi::ObjectWrap<Secp256k1Addon> {
 public:
  static Napi::Value Init(Napi::Env env);

  Secp256k1Addon(const Napi::CallbackInfo& info);
  void Finalize(Napi::Env env);

  struct ECDSASignData {
    napi_env env;
    Napi::Function fn;
    Napi::Value msg32;
    Napi::Value key32;
    Napi::Value data;
  };

  struct ECDHData {
    Napi::Function fn;
    Napi::Value xbuf;
    Napi::Value ybuf;
    Napi::Value data;
    size_t outputlen;
  };

 private:
  static Napi::FunctionReference constructor;
  static unsigned int secp256k1_context_flags;
  const secp256k1_context* ctx_;
  ECDSASignData ecdsa_sign_data;
  ECDHData ecdh_data;

  Napi::Value ContextRandomize(const Napi::CallbackInfo& info);

  Napi::Value PrivateKeyVerify(const Napi::CallbackInfo& info);
  Napi::Value PrivateKeyNegate(const Napi::CallbackInfo& info);
  Napi::Value PrivateKeyTweakAdd(const Napi::CallbackInfo& info);
  Napi::Value PrivateKeyTweakMul(const Napi::CallbackInfo& info);

  Napi::Value PublicKeyVerify(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyCreate(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyConvert(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyNegate(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyCombine(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyTweakAdd(const Napi::CallbackInfo& info);
  Napi::Value PublicKeyTweakMul(const Napi::CallbackInfo& info);

  Napi::Value SignatureNormalize(const Napi::CallbackInfo& info);
  Napi::Value SignatureExport(const Napi::CallbackInfo& info);
  Napi::Value SignatureImport(const Napi::CallbackInfo& info);

  Napi::Value ECDSASign(const Napi::CallbackInfo& info);
  Napi::Value ECDSAVerify(const Napi::CallbackInfo& info);
  Napi::Value ECDSARecover(const Napi::CallbackInfo& info);

  Napi::Value ECDH(const Napi::CallbackInfo& info);
};

#endif  // ADDON_SECP256K1
