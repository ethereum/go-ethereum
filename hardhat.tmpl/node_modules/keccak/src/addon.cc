#include <napi.h>

extern "C" {
#include <KeccakSpongeWidth1600.h>
}

class KeccakWrapper : public Napi::ObjectWrap<KeccakWrapper> {
 public:
  static Napi::Object Init(Napi::Env env);

  KeccakWrapper(const Napi::CallbackInfo& info);

 private:
  KeccakWidth1600_SpongeInstance sponge;

  Napi::Value Initialize(const Napi::CallbackInfo& info);
  Napi::Value Absorb(const Napi::CallbackInfo& info);
  Napi::Value AbsorbLastFewBits(const Napi::CallbackInfo& info);
  Napi::Value Squeeze(const Napi::CallbackInfo& info);
  Napi::Value Copy(const Napi::CallbackInfo& info);
};

Napi::Object KeccakWrapper::Init(Napi::Env env) {
  Napi::Function func =
      DefineClass(env,
                  "KeccakWrapper",
                  {
                      InstanceMethod("initialize", &KeccakWrapper::Initialize),
                      InstanceMethod("absorb", &KeccakWrapper::Absorb),
                      InstanceMethod("absorbLastFewBits",
                                     &KeccakWrapper::AbsorbLastFewBits),
                      InstanceMethod("squeeze", &KeccakWrapper::Squeeze),
                      InstanceMethod("copy", &KeccakWrapper::Copy),
                  });

  return func;
}

KeccakWrapper::KeccakWrapper(const Napi::CallbackInfo& info)
    : Napi::ObjectWrap<KeccakWrapper>(info) {}

Napi::Value KeccakWrapper::Initialize(const Napi::CallbackInfo& info) {
  auto rate = info[0].As<Napi::Number>().Uint32Value();
  auto capacity = info[1].As<Napi::Number>().Uint32Value();

  // ignore return code,
  // rate & capacity always will right because internal object
  KeccakWidth1600_SpongeInitialize(&sponge, rate, capacity);

  return info.Env().Undefined();
}

Napi::Value KeccakWrapper::Absorb(const Napi::CallbackInfo& info) {
  auto buf = info[0].As<Napi::Buffer<const unsigned char>>();

  // ignore return code, bcause internal object
  KeccakWidth1600_SpongeAbsorb(&sponge, buf.Data(), buf.Length());

  return info.Env().Undefined();
}

Napi::Value KeccakWrapper::AbsorbLastFewBits(const Napi::CallbackInfo& info) {
  auto bits = info[0].As<Napi::Number>().Uint32Value();

  // ignore return code, bcause internal object
  KeccakWidth1600_SpongeAbsorbLastFewBits(&sponge, bits);

  return info.Env().Undefined();
}

Napi::Value KeccakWrapper::Squeeze(const Napi::CallbackInfo& info) {
  auto length = info[0].As<Napi::Number>().Uint32Value();
  auto buf = Napi::Buffer<unsigned char>::New(info.Env(), length);

  KeccakWidth1600_SpongeSqueeze(&sponge, buf.Data(), length);

  return buf;
}

Napi::Value KeccakWrapper::Copy(const Napi::CallbackInfo& info) {
  auto to = Napi::ObjectWrap<KeccakWrapper>::Unwrap(info[0].As<Napi::Object>());

  memcpy(&to->sponge, &sponge, sizeof(KeccakWidth1600_SpongeInstance));

  return info.Env().Undefined();
}

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  return KeccakWrapper::Init(env);
}

NODE_API_MODULE(NODE_GYP_MODULE_NAME, Init)
