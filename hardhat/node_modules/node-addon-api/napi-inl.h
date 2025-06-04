#ifndef SRC_NAPI_INL_H_
#define SRC_NAPI_INL_H_

////////////////////////////////////////////////////////////////////////////////
// N-API C++ Wrapper Classes
//
// Inline header-only implementations for "N-API" ABI-stable C APIs for Node.js.
////////////////////////////////////////////////////////////////////////////////

// Note: Do not include this file directly! Include "napi.h" instead.

#include <algorithm>
#include <atomic>
#include <cstring>
#include <mutex>
#include <type_traits>

namespace Napi {

// Helpers to handle functions exposed from C++.
namespace details {

extern std::atomic_bool needs_objectwrap_destructor_fix;

// Attach a data item to an object and delete it when the object gets
// garbage-collected.
// TODO: Replace this code with `napi_add_finalizer()` whenever it becomes
// available on all supported versions of Node.js.
template <typename FreeType>
static inline napi_status AttachData(napi_env env,
                                     napi_value obj,
                                     FreeType* data,
                                     napi_finalize finalizer = nullptr,
                                     void* hint = nullptr) {
  napi_status status;
  if (finalizer == nullptr) {
    finalizer = [](napi_env /*env*/, void* data, void* /*hint*/) {
      delete static_cast<FreeType*>(data);
    };
  }
#if (NAPI_VERSION < 5)
  napi_value symbol, external;
  status = napi_create_symbol(env, nullptr, &symbol);
  if (status == napi_ok) {
    status = napi_create_external(env,
                              data,
                              finalizer,
                              hint,
                              &external);
    if (status == napi_ok) {
      napi_property_descriptor desc = {
        nullptr,
        symbol,
        nullptr,
        nullptr,
        nullptr,
        external,
        napi_default,
        nullptr
      };
      status = napi_define_properties(env, obj, 1, &desc);
    }
  }
#else  // NAPI_VERSION >= 5
  status = napi_add_finalizer(env, obj, data, finalizer, hint, nullptr);
#endif
  return status;
}

// For use in JS to C++ callback wrappers to catch any Napi::Error exceptions
// and rethrow them as JavaScript exceptions before returning from the callback.
template <typename Callable>
inline napi_value WrapCallback(Callable callback) {
#ifdef NAPI_CPP_EXCEPTIONS
  try {
    return callback();
  } catch (const Error& e) {
    e.ThrowAsJavaScriptException();
    return nullptr;
  }
#else // NAPI_CPP_EXCEPTIONS
  // When C++ exceptions are disabled, errors are immediately thrown as JS
  // exceptions, so there is no need to catch and rethrow them here.
  return callback();
#endif // NAPI_CPP_EXCEPTIONS
}

template <typename Callable, typename Return>
struct CallbackData {
  static inline
  napi_value Wrapper(napi_env env, napi_callback_info info) {
    return details::WrapCallback([&] {
      CallbackInfo callbackInfo(env, info);
      CallbackData* callbackData =
        static_cast<CallbackData*>(callbackInfo.Data());
      callbackInfo.SetData(callbackData->data);
      return callbackData->callback(callbackInfo);
    });
  }

  Callable callback;
  void* data;
};

template <typename Callable>
struct CallbackData<Callable, void> {
  static inline
  napi_value Wrapper(napi_env env, napi_callback_info info) {
    return details::WrapCallback([&] {
      CallbackInfo callbackInfo(env, info);
      CallbackData* callbackData =
        static_cast<CallbackData*>(callbackInfo.Data());
      callbackInfo.SetData(callbackData->data);
      callbackData->callback(callbackInfo);
      return nullptr;
    });
  }

  Callable callback;
  void* data;
};

template <typename T, typename Finalizer, typename Hint = void>
struct FinalizeData {
  static inline
  void Wrapper(napi_env env, void* data, void* finalizeHint) {
    FinalizeData* finalizeData = static_cast<FinalizeData*>(finalizeHint);
    finalizeData->callback(Env(env), static_cast<T*>(data));
    delete finalizeData;
  }

  static inline
  void WrapperWithHint(napi_env env, void* data, void* finalizeHint) {
    FinalizeData* finalizeData = static_cast<FinalizeData*>(finalizeHint);
    finalizeData->callback(Env(env), static_cast<T*>(data), finalizeData->hint);
    delete finalizeData;
  }

  Finalizer callback;
  Hint* hint;
};

#if (NAPI_VERSION > 3)
template <typename ContextType=void,
          typename Finalizer=std::function<void(Env, void*, ContextType*)>,
          typename FinalizerDataType=void>
struct ThreadSafeFinalize {
  static inline
  void Wrapper(napi_env env, void* rawFinalizeData, void* /* rawContext */) {
    if (rawFinalizeData == nullptr)
      return;

    ThreadSafeFinalize* finalizeData =
        static_cast<ThreadSafeFinalize*>(rawFinalizeData);
    finalizeData->callback(Env(env));
    if (finalizeData->tsfn) {
      *finalizeData->tsfn = nullptr;
    }
    delete finalizeData;
  }

  static inline
  void FinalizeWrapperWithData(napi_env env,
                               void* rawFinalizeData,
                               void* /* rawContext */) {
    if (rawFinalizeData == nullptr)
      return;

    ThreadSafeFinalize* finalizeData =
        static_cast<ThreadSafeFinalize*>(rawFinalizeData);
    finalizeData->callback(Env(env), finalizeData->data);
    if (finalizeData->tsfn) {
      *finalizeData->tsfn = nullptr;
    }
    delete finalizeData;
  }

  static inline
  void FinalizeWrapperWithContext(napi_env env,
                                  void* rawFinalizeData,
                                  void* rawContext) {
    if (rawFinalizeData == nullptr)
      return;

    ThreadSafeFinalize* finalizeData =
        static_cast<ThreadSafeFinalize*>(rawFinalizeData);
    finalizeData->callback(Env(env), static_cast<ContextType*>(rawContext));
    if (finalizeData->tsfn) {
      *finalizeData->tsfn = nullptr;
    }
    delete finalizeData;
  }

  static inline
  void FinalizeFinalizeWrapperWithDataAndContext(napi_env env,
                                         void* rawFinalizeData,
                                         void* rawContext) {
    if (rawFinalizeData == nullptr)
      return;

    ThreadSafeFinalize* finalizeData =
        static_cast<ThreadSafeFinalize*>(rawFinalizeData);
    finalizeData->callback(Env(env), finalizeData->data,
        static_cast<ContextType*>(rawContext));
    if (finalizeData->tsfn) {
      *finalizeData->tsfn = nullptr;
    }
    delete finalizeData;
  }

  FinalizerDataType* data;
  Finalizer callback;
  napi_threadsafe_function* tsfn;
};
#endif

template <typename Getter, typename Setter>
struct AccessorCallbackData {
  static inline
  napi_value GetterWrapper(napi_env env, napi_callback_info info) {
    return details::WrapCallback([&] {
      CallbackInfo callbackInfo(env, info);
      AccessorCallbackData* callbackData =
        static_cast<AccessorCallbackData*>(callbackInfo.Data());
      callbackInfo.SetData(callbackData->data);
      return callbackData->getterCallback(callbackInfo);
    });
  }

  static inline
  napi_value SetterWrapper(napi_env env, napi_callback_info info) {
    return details::WrapCallback([&] {
      CallbackInfo callbackInfo(env, info);
      AccessorCallbackData* callbackData =
        static_cast<AccessorCallbackData*>(callbackInfo.Data());
      callbackInfo.SetData(callbackData->data);
      callbackData->setterCallback(callbackInfo);
      return nullptr;
    });
  }

  Getter getterCallback;
  Setter setterCallback;
  void* data;
};

}  // namespace details

#ifndef NODE_ADDON_API_DISABLE_DEPRECATED
# include "napi-inl.deprecated.h"
#endif // !NODE_ADDON_API_DISABLE_DEPRECATED

////////////////////////////////////////////////////////////////////////////////
// Module registration
////////////////////////////////////////////////////////////////////////////////

#define NODE_API_MODULE(modname, regfunc)                      \
  namespace Napi {                                             \
    namespace details {                                        \
      std::atomic_bool needs_objectwrap_destructor_fix(false); \
    }                                                          \
  }                                                            \
  napi_value __napi_ ## regfunc(napi_env env,                  \
                                napi_value exports) {          \
    return Napi::RegisterModule(env, exports, regfunc);        \
  }                                                            \
  NAPI_MODULE(modname, __napi_ ## regfunc)

// Adapt the NAPI_MODULE registration function:
//  - Wrap the arguments in NAPI wrappers.
//  - Catch any NAPI errors and rethrow as JS exceptions.
inline napi_value RegisterModule(napi_env env,
                                 napi_value exports,
                                 ModuleRegisterCallback registerCallback) {
  const napi_node_version* nver = Napi::VersionManagement::GetNodeVersion(env);
  Napi::details::needs_objectwrap_destructor_fix =
    (nver->major < 10 ||
      (nver->major == 10 && nver->minor < 15) ||
      (nver->major == 10 && nver->minor == 15 && nver->patch < 3));

  return details::WrapCallback([&] {
    return napi_value(registerCallback(Napi::Env(env),
                                       Napi::Object(env, exports)));
  });
}

////////////////////////////////////////////////////////////////////////////////
// Env class
////////////////////////////////////////////////////////////////////////////////

inline Env::Env(napi_env env) : _env(env) {
}

inline Env::operator napi_env() const {
  return _env;
}

inline Object Env::Global() const {
  napi_value value;
  napi_status status = napi_get_global(*this, &value);
  NAPI_THROW_IF_FAILED(*this, status, Object());
  return Object(*this, value);
}

inline Value Env::Undefined() const {
  napi_value value;
  napi_status status = napi_get_undefined(*this, &value);
  NAPI_THROW_IF_FAILED(*this, status, Value());
  return Value(*this, value);
}

inline Value Env::Null() const {
  napi_value value;
  napi_status status = napi_get_null(*this, &value);
  NAPI_THROW_IF_FAILED(*this, status, Value());
  return Value(*this, value);
}

inline bool Env::IsExceptionPending() const {
  bool result;
  napi_status status = napi_is_exception_pending(_env, &result);
  if (status != napi_ok) result = false; // Checking for a pending exception shouldn't throw.
  return result;
}

inline Error Env::GetAndClearPendingException() {
  napi_value value;
  napi_status status = napi_get_and_clear_last_exception(_env, &value);
  if (status != napi_ok) {
    // Don't throw another exception when failing to get the exception!
    return Error();
  }
  return Error(_env, value);
}

////////////////////////////////////////////////////////////////////////////////
// Value class
////////////////////////////////////////////////////////////////////////////////

inline Value::Value() : _env(nullptr), _value(nullptr) {
}

inline Value::Value(napi_env env, napi_value value) : _env(env), _value(value) {
}

inline Value::operator napi_value() const {
  return _value;
}

inline bool Value::operator ==(const Value& other) const {
  return StrictEquals(other);
}

inline bool Value::operator !=(const Value& other) const {
  return !this->operator ==(other);
}

inline bool Value::StrictEquals(const Value& other) const {
  bool result;
  napi_status status = napi_strict_equals(_env, *this, other, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline Napi::Env Value::Env() const {
  return Napi::Env(_env);
}

inline bool Value::IsEmpty() const {
  return _value == nullptr;
}

inline napi_valuetype Value::Type() const {
  if (IsEmpty()) {
    return napi_undefined;
  }

  napi_valuetype type;
  napi_status status = napi_typeof(_env, _value, &type);
  NAPI_THROW_IF_FAILED(_env, status, napi_undefined);
  return type;
}

inline bool Value::IsUndefined() const {
  return Type() == napi_undefined;
}

inline bool Value::IsNull() const {
  return Type() == napi_null;
}

inline bool Value::IsBoolean() const {
  return Type() == napi_boolean;
}

inline bool Value::IsNumber() const {
  return Type() == napi_number;
}

// Currently experimental guard with the definition of NAPI_EXPERIMENTAL.
// Once it is no longer experimental guard with the NAPI_VERSION in which it is
// released instead.
#ifdef NAPI_EXPERIMENTAL
inline bool Value::IsBigInt() const {
  return Type() == napi_bigint;
}
#endif  // NAPI_EXPERIMENTAL

#if (NAPI_VERSION > 4)
inline bool Value::IsDate() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_date(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}
#endif

inline bool Value::IsString() const {
  return Type() == napi_string;
}

inline bool Value::IsSymbol() const {
  return Type() == napi_symbol;
}

inline bool Value::IsArray() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_array(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsArrayBuffer() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_arraybuffer(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsTypedArray() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_typedarray(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsObject() const {
  return Type() == napi_object || IsFunction();
}

inline bool Value::IsFunction() const {
  return Type() == napi_function;
}

inline bool Value::IsPromise() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_promise(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsDataView() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_dataview(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsBuffer() const {
  if (IsEmpty()) {
    return false;
  }

  bool result;
  napi_status status = napi_is_buffer(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Value::IsExternal() const {
  return Type() == napi_external;
}

template <typename T>
inline T Value::As() const {
  return T(_env, _value);
}

inline Boolean Value::ToBoolean() const {
  napi_value result;
  napi_status status = napi_coerce_to_bool(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, Boolean());
  return Boolean(_env, result);
}

inline Number Value::ToNumber() const {
  napi_value result;
  napi_status status = napi_coerce_to_number(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, Number());
  return Number(_env, result);
}

inline String Value::ToString() const {
  napi_value result;
  napi_status status = napi_coerce_to_string(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, String());
  return String(_env, result);
}

inline Object Value::ToObject() const {
  napi_value result;
  napi_status status = napi_coerce_to_object(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, Object());
  return Object(_env, result);
}

////////////////////////////////////////////////////////////////////////////////
// Boolean class
////////////////////////////////////////////////////////////////////////////////

inline Boolean Boolean::New(napi_env env, bool val) {
  napi_value value;
  napi_status status = napi_get_boolean(env, val, &value);
  NAPI_THROW_IF_FAILED(env, status, Boolean());
  return Boolean(env, value);
}

inline Boolean::Boolean() : Napi::Value() {
}

inline Boolean::Boolean(napi_env env, napi_value value) : Napi::Value(env, value) {
}

inline Boolean::operator bool() const {
  return Value();
}

inline bool Boolean::Value() const {
  bool result;
  napi_status status = napi_get_value_bool(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

////////////////////////////////////////////////////////////////////////////////
// Number class
////////////////////////////////////////////////////////////////////////////////

inline Number Number::New(napi_env env, double val) {
  napi_value value;
  napi_status status = napi_create_double(env, val, &value);
  NAPI_THROW_IF_FAILED(env, status, Number());
  return Number(env, value);
}

inline Number::Number() : Value() {
}

inline Number::Number(napi_env env, napi_value value) : Value(env, value) {
}

inline Number::operator int32_t() const {
  return Int32Value();
}

inline Number::operator uint32_t() const {
  return Uint32Value();
}

inline Number::operator int64_t() const {
  return Int64Value();
}

inline Number::operator float() const {
  return FloatValue();
}

inline Number::operator double() const {
  return DoubleValue();
}

inline int32_t Number::Int32Value() const {
  int32_t result;
  napi_status status = napi_get_value_int32(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

inline uint32_t Number::Uint32Value() const {
  uint32_t result;
  napi_status status = napi_get_value_uint32(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

inline int64_t Number::Int64Value() const {
  int64_t result;
  napi_status status = napi_get_value_int64(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

inline float Number::FloatValue() const {
  return static_cast<float>(DoubleValue());
}

inline double Number::DoubleValue() const {
  double result;
  napi_status status = napi_get_value_double(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

// Currently experimental guard with the definition of NAPI_EXPERIMENTAL.
// Once it is no longer experimental guard with the NAPI_VERSION in which it is
// released instead.
#ifdef NAPI_EXPERIMENTAL
////////////////////////////////////////////////////////////////////////////////
// BigInt Class
////////////////////////////////////////////////////////////////////////////////

inline BigInt BigInt::New(napi_env env, int64_t val) {
  napi_value value;
  napi_status status = napi_create_bigint_int64(env, val, &value);
  NAPI_THROW_IF_FAILED(env, status, BigInt());
  return BigInt(env, value);
}

inline BigInt BigInt::New(napi_env env, uint64_t val) {
  napi_value value;
  napi_status status = napi_create_bigint_uint64(env, val, &value);
  NAPI_THROW_IF_FAILED(env, status, BigInt());
  return BigInt(env, value);
}

inline BigInt BigInt::New(napi_env env, int sign_bit, size_t word_count, const uint64_t* words) {
  napi_value value;
  napi_status status = napi_create_bigint_words(env, sign_bit, word_count, words, &value);
  NAPI_THROW_IF_FAILED(env, status, BigInt());
  return BigInt(env, value);
}

inline BigInt::BigInt() : Value() {
}

inline BigInt::BigInt(napi_env env, napi_value value) : Value(env, value) {
}

inline int64_t BigInt::Int64Value(bool* lossless) const {
  int64_t result;
  napi_status status = napi_get_value_bigint_int64(
      _env, _value, &result, lossless);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

inline uint64_t BigInt::Uint64Value(bool* lossless) const {
  uint64_t result;
  napi_status status = napi_get_value_bigint_uint64(
      _env, _value, &result, lossless);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

inline size_t BigInt::WordCount() const {
  size_t word_count;
  napi_status status = napi_get_value_bigint_words(
      _env, _value, nullptr, &word_count, nullptr);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return word_count;
}

inline void BigInt::ToWords(int* sign_bit, size_t* word_count, uint64_t* words) {
  napi_status status = napi_get_value_bigint_words(
      _env, _value, sign_bit, word_count, words);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}
#endif  // NAPI_EXPERIMENTAL

#if (NAPI_VERSION > 4)
////////////////////////////////////////////////////////////////////////////////
// Date Class
////////////////////////////////////////////////////////////////////////////////

inline Date Date::New(napi_env env, double val) {
  napi_value value;
  napi_status status = napi_create_date(env, val, &value);
  NAPI_THROW_IF_FAILED(env, status, Date());
  return Date(env, value);
}

inline Date::Date() : Value() {
}

inline Date::Date(napi_env env, napi_value value) : Value(env, value) {
}

inline Date::operator double() const {
  return ValueOf();
}

inline double Date::ValueOf() const {
  double result;
  napi_status status = napi_get_date_value(
      _env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}
#endif

////////////////////////////////////////////////////////////////////////////////
// Name class
////////////////////////////////////////////////////////////////////////////////

inline Name::Name() : Value() {
}

inline Name::Name(napi_env env, napi_value value) : Value(env, value) {
}

////////////////////////////////////////////////////////////////////////////////
// String class
////////////////////////////////////////////////////////////////////////////////

inline String String::New(napi_env env, const std::string& val) {
  return String::New(env, val.c_str(), val.size());
}

inline String String::New(napi_env env, const std::u16string& val) {
  return String::New(env, val.c_str(), val.size());
}

inline String String::New(napi_env env, const char* val) {
  napi_value value;
  napi_status status = napi_create_string_utf8(env, val, std::strlen(val), &value);
  NAPI_THROW_IF_FAILED(env, status, String());
  return String(env, value);
}

inline String String::New(napi_env env, const char16_t* val) {
  napi_value value;
  napi_status status = napi_create_string_utf16(env, val, std::u16string(val).size(), &value);
  NAPI_THROW_IF_FAILED(env, status, String());
  return String(env, value);
}

inline String String::New(napi_env env, const char* val, size_t length) {
  napi_value value;
  napi_status status = napi_create_string_utf8(env, val, length, &value);
  NAPI_THROW_IF_FAILED(env, status, String());
  return String(env, value);
}

inline String String::New(napi_env env, const char16_t* val, size_t length) {
  napi_value value;
  napi_status status = napi_create_string_utf16(env, val, length, &value);
  NAPI_THROW_IF_FAILED(env, status, String());
  return String(env, value);
}

inline String::String() : Name() {
}

inline String::String(napi_env env, napi_value value) : Name(env, value) {
}

inline String::operator std::string() const {
  return Utf8Value();
}

inline String::operator std::u16string() const {
  return Utf16Value();
}

inline std::string String::Utf8Value() const {
  size_t length;
  napi_status status = napi_get_value_string_utf8(_env, _value, nullptr, 0, &length);
  NAPI_THROW_IF_FAILED(_env, status, "");

  std::string value;
  value.reserve(length + 1);
  value.resize(length);
  status = napi_get_value_string_utf8(_env, _value, &value[0], value.capacity(), nullptr);
  NAPI_THROW_IF_FAILED(_env, status, "");
  return value;
}

inline std::u16string String::Utf16Value() const {
  size_t length;
  napi_status status = napi_get_value_string_utf16(_env, _value, nullptr, 0, &length);
  NAPI_THROW_IF_FAILED(_env, status, NAPI_WIDE_TEXT(""));

  std::u16string value;
  value.reserve(length + 1);
  value.resize(length);
  status = napi_get_value_string_utf16(_env, _value, &value[0], value.capacity(), nullptr);
  NAPI_THROW_IF_FAILED(_env, status, NAPI_WIDE_TEXT(""));
  return value;
}

////////////////////////////////////////////////////////////////////////////////
// Symbol class
////////////////////////////////////////////////////////////////////////////////

inline Symbol Symbol::New(napi_env env, const char* description) {
  napi_value descriptionValue = description != nullptr ?
    String::New(env, description) : static_cast<napi_value>(nullptr);
  return Symbol::New(env, descriptionValue);
}

inline Symbol Symbol::New(napi_env env, const std::string& description) {
  napi_value descriptionValue = String::New(env, description);
  return Symbol::New(env, descriptionValue);
}

inline Symbol Symbol::New(napi_env env, String description) {
  napi_value descriptionValue = description;
  return Symbol::New(env, descriptionValue);
}

inline Symbol Symbol::New(napi_env env, napi_value description) {
  napi_value value;
  napi_status status = napi_create_symbol(env, description, &value);
  NAPI_THROW_IF_FAILED(env, status, Symbol());
  return Symbol(env, value);
}

inline Symbol Symbol::WellKnown(napi_env env, const std::string& name) {
  return Napi::Env(env).Global().Get("Symbol").As<Object>().Get(name).As<Symbol>();
}

inline Symbol::Symbol() : Name() {
}

inline Symbol::Symbol(napi_env env, napi_value value) : Name(env, value) {
}

////////////////////////////////////////////////////////////////////////////////
// Automagic value creation
////////////////////////////////////////////////////////////////////////////////

namespace details {
template <typename T>
struct vf_number {
  static Number From(napi_env env, T value) {
    return Number::New(env, static_cast<double>(value));
  }
};

template<>
struct vf_number<bool> {
  static Boolean From(napi_env env, bool value) {
    return Boolean::New(env, value);
  }
};

struct vf_utf8_charp {
  static String From(napi_env env, const char* value) {
    return String::New(env, value);
  }
};

struct vf_utf16_charp {
  static String From(napi_env env, const char16_t* value) {
    return String::New(env, value);
  }
};
struct vf_utf8_string {
  static String From(napi_env env, const std::string& value) {
    return String::New(env, value);
  }
};

struct vf_utf16_string {
  static String From(napi_env env, const std::u16string& value) {
    return String::New(env, value);
  }
};

template <typename T>
struct vf_fallback {
  static Value From(napi_env env, const T& value) {
    return Value(env, value);
  }
};

template <typename...> struct disjunction : std::false_type {};
template <typename B> struct disjunction<B> : B {};
template <typename B, typename... Bs>
struct disjunction<B, Bs...>
    : std::conditional<bool(B::value), B, disjunction<Bs...>>::type {};

template <typename T>
struct can_make_string
    : disjunction<typename std::is_convertible<T, const char *>::type,
                  typename std::is_convertible<T, const char16_t *>::type,
                  typename std::is_convertible<T, std::string>::type,
                  typename std::is_convertible<T, std::u16string>::type> {};
}

template <typename T>
Value Value::From(napi_env env, const T& value) {
  using Helper = typename std::conditional<
    std::is_integral<T>::value || std::is_floating_point<T>::value,
    details::vf_number<T>,
    typename std::conditional<
      details::can_make_string<T>::value,
      String,
      details::vf_fallback<T>
    >::type
  >::type;
  return Helper::From(env, value);
}

template <typename T>
String String::From(napi_env env, const T& value) {
  struct Dummy {};
  using Helper = typename std::conditional<
    std::is_convertible<T, const char*>::value,
    details::vf_utf8_charp,
    typename std::conditional<
      std::is_convertible<T, const char16_t*>::value,
      details::vf_utf16_charp,
      typename std::conditional<
        std::is_convertible<T, std::string>::value,
        details::vf_utf8_string,
        typename std::conditional<
          std::is_convertible<T, std::u16string>::value,
          details::vf_utf16_string,
          Dummy
        >::type
      >::type
    >::type
  >::type;
  return Helper::From(env, value);
}

////////////////////////////////////////////////////////////////////////////////
// Object class
////////////////////////////////////////////////////////////////////////////////

template <typename Key>
inline Object::PropertyLValue<Key>::operator Value() const {
  return Object(_env, _object).Get(_key);
}

template <typename Key> template <typename ValueType>
inline Object::PropertyLValue<Key>& Object::PropertyLValue<Key>::operator =(ValueType value) {
  Object(_env, _object).Set(_key, value);
  return *this;
}

template <typename Key>
inline Object::PropertyLValue<Key>::PropertyLValue(Object object, Key key)
  : _env(object.Env()), _object(object), _key(key) {}

inline Object Object::New(napi_env env) {
  napi_value value;
  napi_status status = napi_create_object(env, &value);
  NAPI_THROW_IF_FAILED(env, status, Object());
  return Object(env, value);
}

inline Object::Object() : Value() {
}

inline Object::Object(napi_env env, napi_value value) : Value(env, value) {
}

inline Object::PropertyLValue<std::string> Object::operator [](const char* utf8name) {
  return PropertyLValue<std::string>(*this, utf8name);
}

inline Object::PropertyLValue<std::string> Object::operator [](const std::string& utf8name) {
  return PropertyLValue<std::string>(*this, utf8name);
}

inline Object::PropertyLValue<uint32_t> Object::operator [](uint32_t index) {
  return PropertyLValue<uint32_t>(*this, index);
}

inline Value Object::operator [](const char* utf8name) const {
  return Get(utf8name);
}

inline Value Object::operator [](const std::string& utf8name) const {
  return Get(utf8name);
}

inline Value Object::operator [](uint32_t index) const {
  return Get(index);
}

inline bool Object::Has(napi_value key) const {
  bool result;
  napi_status status = napi_has_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::Has(Value key) const {
  bool result;
  napi_status status = napi_has_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::Has(const char* utf8name) const {
  bool result;
  napi_status status = napi_has_named_property(_env, _value, utf8name, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::Has(const std::string& utf8name) const {
  return Has(utf8name.c_str());
}

inline bool Object::HasOwnProperty(napi_value key) const {
  bool result;
  napi_status status = napi_has_own_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::HasOwnProperty(Value key) const {
  bool result;
  napi_status status = napi_has_own_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::HasOwnProperty(const char* utf8name) const {
  napi_value key;
  napi_status status = napi_create_string_utf8(_env, utf8name, std::strlen(utf8name), &key);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return HasOwnProperty(key);
}

inline bool Object::HasOwnProperty(const std::string& utf8name) const {
  return HasOwnProperty(utf8name.c_str());
}

inline Value Object::Get(napi_value key) const {
  napi_value result;
  napi_status status = napi_get_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}

inline Value Object::Get(Value key) const {
  napi_value result;
  napi_status status = napi_get_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}

inline Value Object::Get(const char* utf8name) const {
  napi_value result;
  napi_status status = napi_get_named_property(_env, _value, utf8name, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}

inline Value Object::Get(const std::string& utf8name) const {
  return Get(utf8name.c_str());
}

template <typename ValueType>
inline void Object::Set(napi_value key, const ValueType& value) {
  napi_status status =
      napi_set_property(_env, _value, key, Value::From(_env, value));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

template <typename ValueType>
inline void Object::Set(Value key, const ValueType& value) {
  napi_status status =
      napi_set_property(_env, _value, key, Value::From(_env, value));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

template <typename ValueType>
inline void Object::Set(const char* utf8name, const ValueType& value) {
  napi_status status =
      napi_set_named_property(_env, _value, utf8name, Value::From(_env, value));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

template <typename ValueType>
inline void Object::Set(const std::string& utf8name, const ValueType& value) {
  Set(utf8name.c_str(), value);
}

inline bool Object::Delete(napi_value key) {
  bool result;
  napi_status status = napi_delete_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::Delete(Value key) {
  bool result;
  napi_status status = napi_delete_property(_env, _value, key, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline bool Object::Delete(const char* utf8name) {
  return Delete(String::New(_env, utf8name));
}

inline bool Object::Delete(const std::string& utf8name) {
  return Delete(String::New(_env, utf8name));
}

inline bool Object::Has(uint32_t index) const {
  bool result;
  napi_status status = napi_has_element(_env, _value, index, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline Value Object::Get(uint32_t index) const {
  napi_value value;
  napi_status status = napi_get_element(_env, _value, index, &value);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, value);
}

template <typename ValueType>
inline void Object::Set(uint32_t index, const ValueType& value) {
  napi_status status =
      napi_set_element(_env, _value, index, Value::From(_env, value));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline bool Object::Delete(uint32_t index) {
  bool result;
  napi_status status = napi_delete_element(_env, _value, index, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

inline Array Object::GetPropertyNames() const {
  napi_value result;
  napi_status status = napi_get_property_names(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, Array());
  return Array(_env, result);
}

inline void Object::DefineProperty(const PropertyDescriptor& property) {
  napi_status status = napi_define_properties(_env, _value, 1,
    reinterpret_cast<const napi_property_descriptor*>(&property));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline void Object::DefineProperties(const std::initializer_list<PropertyDescriptor>& properties) {
  napi_status status = napi_define_properties(_env, _value, properties.size(),
    reinterpret_cast<const napi_property_descriptor*>(properties.begin()));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline void Object::DefineProperties(const std::vector<PropertyDescriptor>& properties) {
  napi_status status = napi_define_properties(_env, _value, properties.size(),
    reinterpret_cast<const napi_property_descriptor*>(properties.data()));
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline bool Object::InstanceOf(const Function& constructor) const {
  bool result;
  napi_status status = napi_instanceof(_env, _value, constructor, &result);
  NAPI_THROW_IF_FAILED(_env, status, false);
  return result;
}

template <typename Finalizer, typename T>
inline void Object::AddFinalizer(Finalizer finalizeCallback, T* data) {
  details::FinalizeData<T, Finalizer>* finalizeData =
    new details::FinalizeData<T, Finalizer>({ finalizeCallback, nullptr });
  napi_status status =
      details::AttachData(_env,
                          *this,
                          data,
                          details::FinalizeData<T, Finalizer>::Wrapper,
                          finalizeData);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED_VOID(_env, status);
  }
}

template <typename Finalizer, typename T, typename Hint>
inline void Object::AddFinalizer(Finalizer finalizeCallback,
                                 T* data,
                                 Hint* finalizeHint) {
  details::FinalizeData<T, Finalizer, Hint>* finalizeData =
    new details::FinalizeData<T, Finalizer, Hint>({ finalizeCallback, finalizeHint });
  napi_status status =
      details::AttachData(_env,
                          *this,
                          data,
                          details::FinalizeData<T, Finalizer, Hint>::WrapperWithHint,
                          finalizeData);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED_VOID(_env, status);
  }
}

////////////////////////////////////////////////////////////////////////////////
// External class
////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline External<T> External<T>::New(napi_env env, T* data) {
  napi_value value;
  napi_status status = napi_create_external(env, data, nullptr, nullptr, &value);
  NAPI_THROW_IF_FAILED(env, status, External());
  return External(env, value);
}

template <typename T>
template <typename Finalizer>
inline External<T> External<T>::New(napi_env env,
                                    T* data,
                                    Finalizer finalizeCallback) {
  napi_value value;
  details::FinalizeData<T, Finalizer>* finalizeData =
    new details::FinalizeData<T, Finalizer>({ finalizeCallback, nullptr });
  napi_status status = napi_create_external(
    env,
    data,
    details::FinalizeData<T, Finalizer>::Wrapper,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, External());
  }
  return External(env, value);
}

template <typename T>
template <typename Finalizer, typename Hint>
inline External<T> External<T>::New(napi_env env,
                                    T* data,
                                    Finalizer finalizeCallback,
                                    Hint* finalizeHint) {
  napi_value value;
  details::FinalizeData<T, Finalizer, Hint>* finalizeData =
    new details::FinalizeData<T, Finalizer, Hint>({ finalizeCallback, finalizeHint });
  napi_status status = napi_create_external(
    env,
    data,
    details::FinalizeData<T, Finalizer, Hint>::WrapperWithHint,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, External());
  }
  return External(env, value);
}

template <typename T>
inline External<T>::External() : Value() {
}

template <typename T>
inline External<T>::External(napi_env env, napi_value value) : Value(env, value) {
}

template <typename T>
inline T* External<T>::Data() const {
  void* data;
  napi_status status = napi_get_value_external(_env, _value, &data);
  NAPI_THROW_IF_FAILED(_env, status, nullptr);
  return reinterpret_cast<T*>(data);
}

////////////////////////////////////////////////////////////////////////////////
// Array class
////////////////////////////////////////////////////////////////////////////////

inline Array Array::New(napi_env env) {
  napi_value value;
  napi_status status = napi_create_array(env, &value);
  NAPI_THROW_IF_FAILED(env, status, Array());
  return Array(env, value);
}

inline Array Array::New(napi_env env, size_t length) {
  napi_value value;
  napi_status status = napi_create_array_with_length(env, length, &value);
  NAPI_THROW_IF_FAILED(env, status, Array());
  return Array(env, value);
}

inline Array::Array() : Object() {
}

inline Array::Array(napi_env env, napi_value value) : Object(env, value) {
}

inline uint32_t Array::Length() const {
  uint32_t result;
  napi_status status = napi_get_array_length(_env, _value, &result);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return result;
}

////////////////////////////////////////////////////////////////////////////////
// ArrayBuffer class
////////////////////////////////////////////////////////////////////////////////

inline ArrayBuffer ArrayBuffer::New(napi_env env, size_t byteLength) {
  napi_value value;
  void* data;
  napi_status status = napi_create_arraybuffer(env, byteLength, &data, &value);
  NAPI_THROW_IF_FAILED(env, status, ArrayBuffer());

  return ArrayBuffer(env, value, data, byteLength);
}

inline ArrayBuffer ArrayBuffer::New(napi_env env,
                                    void* externalData,
                                    size_t byteLength) {
  napi_value value;
  napi_status status = napi_create_external_arraybuffer(
    env, externalData, byteLength, nullptr, nullptr, &value);
  NAPI_THROW_IF_FAILED(env, status, ArrayBuffer());

  return ArrayBuffer(env, value, externalData, byteLength);
}

template <typename Finalizer>
inline ArrayBuffer ArrayBuffer::New(napi_env env,
                                    void* externalData,
                                    size_t byteLength,
                                    Finalizer finalizeCallback) {
  napi_value value;
  details::FinalizeData<void, Finalizer>* finalizeData =
    new details::FinalizeData<void, Finalizer>({ finalizeCallback, nullptr });
  napi_status status = napi_create_external_arraybuffer(
    env,
    externalData,
    byteLength,
    details::FinalizeData<void, Finalizer>::Wrapper,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, ArrayBuffer());
  }

  return ArrayBuffer(env, value, externalData, byteLength);
}

template <typename Finalizer, typename Hint>
inline ArrayBuffer ArrayBuffer::New(napi_env env,
                                    void* externalData,
                                    size_t byteLength,
                                    Finalizer finalizeCallback,
                                    Hint* finalizeHint) {
  napi_value value;
  details::FinalizeData<void, Finalizer, Hint>* finalizeData =
    new details::FinalizeData<void, Finalizer, Hint>({ finalizeCallback, finalizeHint });
  napi_status status = napi_create_external_arraybuffer(
    env,
    externalData,
    byteLength,
    details::FinalizeData<void, Finalizer, Hint>::WrapperWithHint,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, ArrayBuffer());
  }

  return ArrayBuffer(env, value, externalData, byteLength);
}

inline ArrayBuffer::ArrayBuffer() : Object(), _data(nullptr), _length(0) {
}

inline ArrayBuffer::ArrayBuffer(napi_env env, napi_value value)
  : Object(env, value), _data(nullptr), _length(0) {
}

inline ArrayBuffer::ArrayBuffer(napi_env env, napi_value value, void* data, size_t length)
  : Object(env, value), _data(data), _length(length) {
}

inline void* ArrayBuffer::Data() {
  EnsureInfo();
  return _data;
}

inline size_t ArrayBuffer::ByteLength() {
  EnsureInfo();
  return _length;
}

inline void ArrayBuffer::EnsureInfo() const {
  // The ArrayBuffer instance may have been constructed from a napi_value whose
  // length/data are not yet known. Fetch and cache these values just once,
  // since they can never change during the lifetime of the ArrayBuffer.
  if (_data == nullptr) {
    napi_status status = napi_get_arraybuffer_info(_env, _value, &_data, &_length);
    NAPI_THROW_IF_FAILED_VOID(_env, status);
  }
}

////////////////////////////////////////////////////////////////////////////////
// DataView class
////////////////////////////////////////////////////////////////////////////////
inline DataView DataView::New(napi_env env,
                              Napi::ArrayBuffer arrayBuffer) {
  return New(env, arrayBuffer, 0, arrayBuffer.ByteLength());
}

inline DataView DataView::New(napi_env env,
                              Napi::ArrayBuffer arrayBuffer,
                              size_t byteOffset) {
  if (byteOffset > arrayBuffer.ByteLength()) {
    NAPI_THROW(RangeError::New(env,
        "Start offset is outside the bounds of the buffer"),
        DataView());
  }
  return New(env, arrayBuffer, byteOffset,
      arrayBuffer.ByteLength() - byteOffset);
}

inline DataView DataView::New(napi_env env,
                              Napi::ArrayBuffer arrayBuffer,
                              size_t byteOffset,
                              size_t byteLength) {
  if (byteOffset + byteLength > arrayBuffer.ByteLength()) {
    NAPI_THROW(RangeError::New(env, "Invalid DataView length"),
               DataView());
  }
  napi_value value;
  napi_status status = napi_create_dataview(
    env, byteLength, arrayBuffer, byteOffset, &value);
  NAPI_THROW_IF_FAILED(env, status, DataView());
  return DataView(env, value);
}

inline DataView::DataView() : Object() {
}

inline DataView::DataView(napi_env env, napi_value value) : Object(env, value) {
  napi_status status = napi_get_dataview_info(
    _env,
    _value   /* dataView */,
    &_length /* byteLength */,
    &_data   /* data */,
    nullptr  /* arrayBuffer */,
    nullptr  /* byteOffset */);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline Napi::ArrayBuffer DataView::ArrayBuffer() const {
  napi_value arrayBuffer;
  napi_status status = napi_get_dataview_info(
    _env,
    _value       /* dataView */,
    nullptr      /* byteLength */,
    nullptr      /* data */,
    &arrayBuffer /* arrayBuffer */,
    nullptr      /* byteOffset */);
  NAPI_THROW_IF_FAILED(_env, status, Napi::ArrayBuffer());
  return Napi::ArrayBuffer(_env, arrayBuffer);
}

inline size_t DataView::ByteOffset() const {
  size_t byteOffset;
  napi_status status = napi_get_dataview_info(
    _env,
    _value      /* dataView */,
    nullptr     /* byteLength */,
    nullptr     /* data */,
    nullptr     /* arrayBuffer */,
    &byteOffset /* byteOffset */);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return byteOffset;
}

inline size_t DataView::ByteLength() const {
  return _length;
}

inline void* DataView::Data() const {
  return _data;
}

inline float DataView::GetFloat32(size_t byteOffset) const {
  return ReadData<float>(byteOffset);
}

inline double DataView::GetFloat64(size_t byteOffset) const {
  return ReadData<double>(byteOffset);
}

inline int8_t DataView::GetInt8(size_t byteOffset) const {
  return ReadData<int8_t>(byteOffset);
}

inline int16_t DataView::GetInt16(size_t byteOffset) const {
  return ReadData<int16_t>(byteOffset);
}

inline int32_t DataView::GetInt32(size_t byteOffset) const {
  return ReadData<int32_t>(byteOffset);
}

inline uint8_t DataView::GetUint8(size_t byteOffset) const {
  return ReadData<uint8_t>(byteOffset);
}

inline uint16_t DataView::GetUint16(size_t byteOffset) const {
  return ReadData<uint16_t>(byteOffset);
}

inline uint32_t DataView::GetUint32(size_t byteOffset) const {
  return ReadData<uint32_t>(byteOffset);
}

inline void DataView::SetFloat32(size_t byteOffset, float value) const {
  WriteData<float>(byteOffset, value);
}

inline void DataView::SetFloat64(size_t byteOffset, double value) const {
  WriteData<double>(byteOffset, value);
}

inline void DataView::SetInt8(size_t byteOffset, int8_t value) const {
  WriteData<int8_t>(byteOffset, value);
}

inline void DataView::SetInt16(size_t byteOffset, int16_t value) const {
  WriteData<int16_t>(byteOffset, value);
}

inline void DataView::SetInt32(size_t byteOffset, int32_t value) const {
  WriteData<int32_t>(byteOffset, value);
}

inline void DataView::SetUint8(size_t byteOffset, uint8_t value) const {
  WriteData<uint8_t>(byteOffset, value);
}

inline void DataView::SetUint16(size_t byteOffset, uint16_t value) const {
  WriteData<uint16_t>(byteOffset, value);
}

inline void DataView::SetUint32(size_t byteOffset, uint32_t value) const {
  WriteData<uint32_t>(byteOffset, value);
}

template <typename T>
inline T DataView::ReadData(size_t byteOffset) const {
  if (byteOffset + sizeof(T) > _length ||
      byteOffset + sizeof(T) < byteOffset) {  // overflow
    NAPI_THROW(RangeError::New(_env,
        "Offset is outside the bounds of the DataView"), 0);
  }

  return *reinterpret_cast<T*>(static_cast<uint8_t*>(_data) + byteOffset);
}

template <typename T>
inline void DataView::WriteData(size_t byteOffset, T value) const {
  if (byteOffset + sizeof(T) > _length ||
      byteOffset + sizeof(T) < byteOffset) {  // overflow
    NAPI_THROW_VOID(RangeError::New(_env,
        "Offset is outside the bounds of the DataView"));
  }

  *reinterpret_cast<T*>(static_cast<uint8_t*>(_data) + byteOffset) = value;
}

////////////////////////////////////////////////////////////////////////////////
// TypedArray class
////////////////////////////////////////////////////////////////////////////////

inline TypedArray::TypedArray()
  : Object(), _type(TypedArray::unknown_array_type), _length(0) {
}

inline TypedArray::TypedArray(napi_env env, napi_value value)
  : Object(env, value), _type(TypedArray::unknown_array_type), _length(0) {
}

inline TypedArray::TypedArray(napi_env env,
                              napi_value value,
                              napi_typedarray_type type,
                              size_t length)
  : Object(env, value), _type(type), _length(length) {
}

inline napi_typedarray_type TypedArray::TypedArrayType() const {
  if (_type == TypedArray::unknown_array_type) {
    napi_status status = napi_get_typedarray_info(_env, _value,
      &const_cast<TypedArray*>(this)->_type, &const_cast<TypedArray*>(this)->_length,
      nullptr, nullptr, nullptr);
    NAPI_THROW_IF_FAILED(_env, status, napi_int8_array);
  }

  return _type;
}

inline uint8_t TypedArray::ElementSize() const {
  switch (TypedArrayType()) {
    case napi_int8_array:
    case napi_uint8_array:
    case napi_uint8_clamped_array:
      return 1;
    case napi_int16_array:
    case napi_uint16_array:
      return 2;
    case napi_int32_array:
    case napi_uint32_array:
    case napi_float32_array:
      return 4;
    case napi_float64_array:
      return 8;
    default:
      return 0;
  }
}

inline size_t TypedArray::ElementLength() const {
  if (_type == TypedArray::unknown_array_type) {
    napi_status status = napi_get_typedarray_info(_env, _value,
      &const_cast<TypedArray*>(this)->_type, &const_cast<TypedArray*>(this)->_length,
      nullptr, nullptr, nullptr);
    NAPI_THROW_IF_FAILED(_env, status, 0);
  }

  return _length;
}

inline size_t TypedArray::ByteOffset() const {
  size_t byteOffset;
  napi_status status = napi_get_typedarray_info(
    _env, _value, nullptr, nullptr, nullptr, nullptr, &byteOffset);
  NAPI_THROW_IF_FAILED(_env, status, 0);
  return byteOffset;
}

inline size_t TypedArray::ByteLength() const {
  return ElementSize() * ElementLength();
}

inline Napi::ArrayBuffer TypedArray::ArrayBuffer() const {
  napi_value arrayBuffer;
  napi_status status = napi_get_typedarray_info(
    _env, _value, nullptr, nullptr, nullptr, &arrayBuffer, nullptr);
  NAPI_THROW_IF_FAILED(_env, status, Napi::ArrayBuffer());
  return Napi::ArrayBuffer(_env, arrayBuffer);
}

////////////////////////////////////////////////////////////////////////////////
// TypedArrayOf<T> class
////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline TypedArrayOf<T> TypedArrayOf<T>::New(napi_env env,
                                            size_t elementLength,
                                            napi_typedarray_type type) {
  Napi::ArrayBuffer arrayBuffer = Napi::ArrayBuffer::New(env, elementLength * sizeof (T));
  return New(env, elementLength, arrayBuffer, 0, type);
}

template <typename T>
inline TypedArrayOf<T> TypedArrayOf<T>::New(napi_env env,
                                            size_t elementLength,
                                            Napi::ArrayBuffer arrayBuffer,
                                            size_t bufferOffset,
                                            napi_typedarray_type type) {
  napi_value value;
  napi_status status = napi_create_typedarray(
    env, type, elementLength, arrayBuffer, bufferOffset, &value);
  NAPI_THROW_IF_FAILED(env, status, TypedArrayOf<T>());

  return TypedArrayOf<T>(
    env, value, type, elementLength,
    reinterpret_cast<T*>(reinterpret_cast<uint8_t*>(arrayBuffer.Data()) + bufferOffset));
}

template <typename T>
inline TypedArrayOf<T>::TypedArrayOf() : TypedArray(), _data(nullptr) {
}

template <typename T>
inline TypedArrayOf<T>::TypedArrayOf(napi_env env, napi_value value)
  : TypedArray(env, value), _data(nullptr) {
  napi_status status = napi_get_typedarray_info(
    _env, _value, &_type, &_length, reinterpret_cast<void**>(&_data), nullptr, nullptr);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

template <typename T>
inline TypedArrayOf<T>::TypedArrayOf(napi_env env,
                                     napi_value value,
                                     napi_typedarray_type type,
                                     size_t length,
                                     T* data)
  : TypedArray(env, value, type, length), _data(data) {
  if (!(type == TypedArrayTypeForPrimitiveType<T>() ||
      (type == napi_uint8_clamped_array && std::is_same<T, uint8_t>::value))) {
    NAPI_THROW_VOID(TypeError::New(env, "Array type must match the template parameter. "
      "(Uint8 arrays may optionally have the \"clamped\" array type.)"));
  }
}

template <typename T>
inline T& TypedArrayOf<T>::operator [](size_t index) {
  return _data[index];
}

template <typename T>
inline const T& TypedArrayOf<T>::operator [](size_t index) const {
  return _data[index];
}

template <typename T>
inline T* TypedArrayOf<T>::Data() {
  return _data;
}

template <typename T>
inline const T* TypedArrayOf<T>::Data() const {
  return _data;
}

////////////////////////////////////////////////////////////////////////////////
// Function class
////////////////////////////////////////////////////////////////////////////////

template <typename CbData>
static inline napi_status
CreateFunction(napi_env env,
               const char* utf8name,
               napi_callback cb,
               CbData* data,
               napi_value* result) {
  napi_status status =
      napi_create_function(env, utf8name, NAPI_AUTO_LENGTH, cb, data, result);
  if (status == napi_ok) {
    status = Napi::details::AttachData(env, *result, data);
  }

  return status;
}

template <typename Callable>
inline Function Function::New(napi_env env,
                              Callable cb,
                              const char* utf8name,
                              void* data) {
  typedef decltype(cb(CallbackInfo(nullptr, nullptr))) ReturnType;
  typedef details::CallbackData<Callable, ReturnType> CbData;
  auto callbackData = new CbData({ cb, data });

  napi_value value;
  napi_status status = CreateFunction(env,
                                      utf8name,
                                      CbData::Wrapper,
                                      callbackData,
                                      &value);
  if (status != napi_ok) {
    delete callbackData;
    NAPI_THROW_IF_FAILED(env, status, Function());
  }

  return Function(env, value);
}

template <typename Callable>
inline Function Function::New(napi_env env,
                              Callable cb,
                              const std::string& utf8name,
                              void* data) {
  return New(env, cb, utf8name.c_str(), data);
}

inline Function::Function() : Object() {
}

inline Function::Function(napi_env env, napi_value value) : Object(env, value) {
}

inline Value Function::operator ()(const std::initializer_list<napi_value>& args) const {
  return Call(Env().Undefined(), args);
}

inline Value Function::Call(const std::initializer_list<napi_value>& args) const {
  return Call(Env().Undefined(), args);
}

inline Value Function::Call(const std::vector<napi_value>& args) const {
  return Call(Env().Undefined(), args);
}

inline Value Function::Call(size_t argc, const napi_value* args) const {
  return Call(Env().Undefined(), argc, args);
}

inline Value Function::Call(napi_value recv, const std::initializer_list<napi_value>& args) const {
  return Call(recv, args.size(), args.begin());
}

inline Value Function::Call(napi_value recv, const std::vector<napi_value>& args) const {
  return Call(recv, args.size(), args.data());
}

inline Value Function::Call(napi_value recv, size_t argc, const napi_value* args) const {
  napi_value result;
  napi_status status = napi_call_function(
    _env, recv, _value, argc, args, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}

inline Value Function::MakeCallback(
    napi_value recv,
    const std::initializer_list<napi_value>& args,
    napi_async_context context) const {
  return MakeCallback(recv, args.size(), args.begin(), context);
}

inline Value Function::MakeCallback(
    napi_value recv,
    const std::vector<napi_value>& args,
    napi_async_context context) const {
  return MakeCallback(recv, args.size(), args.data(), context);
}

inline Value Function::MakeCallback(
    napi_value recv,
    size_t argc,
    const napi_value* args,
    napi_async_context context) const {
  napi_value result;
  napi_status status = napi_make_callback(
    _env, context, recv, _value, argc, args, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}

inline Object Function::New(const std::initializer_list<napi_value>& args) const {
  return New(args.size(), args.begin());
}

inline Object Function::New(const std::vector<napi_value>& args) const {
  return New(args.size(), args.data());
}

inline Object Function::New(size_t argc, const napi_value* args) const {
  napi_value result;
  napi_status status = napi_new_instance(
    _env, _value, argc, args, &result);
  NAPI_THROW_IF_FAILED(_env, status, Object());
  return Object(_env, result);
}

////////////////////////////////////////////////////////////////////////////////
// Promise class
////////////////////////////////////////////////////////////////////////////////

inline Promise::Deferred Promise::Deferred::New(napi_env env) {
  return Promise::Deferred(env);
}

inline Promise::Deferred::Deferred(napi_env env) : _env(env) {
  napi_status status = napi_create_promise(_env, &_deferred, &_promise);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline Promise Promise::Deferred::Promise() const {
  return Napi::Promise(_env, _promise);
}

inline Napi::Env Promise::Deferred::Env() const {
  return Napi::Env(_env);
}

inline void Promise::Deferred::Resolve(napi_value value) const {
  napi_status status = napi_resolve_deferred(_env, _deferred, value);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline void Promise::Deferred::Reject(napi_value value) const {
  napi_status status = napi_reject_deferred(_env, _deferred, value);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline Promise::Promise(napi_env env, napi_value value) : Object(env, value) {
}

////////////////////////////////////////////////////////////////////////////////
// Buffer<T> class
////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline Buffer<T> Buffer<T>::New(napi_env env, size_t length) {
  napi_value value;
  void* data;
  napi_status status = napi_create_buffer(env, length * sizeof (T), &data, &value);
  NAPI_THROW_IF_FAILED(env, status, Buffer<T>());
  return Buffer(env, value, length, static_cast<T*>(data));
}

template <typename T>
inline Buffer<T> Buffer<T>::New(napi_env env, T* data, size_t length) {
  napi_value value;
  napi_status status = napi_create_external_buffer(
    env, length * sizeof (T), data, nullptr, nullptr, &value);
  NAPI_THROW_IF_FAILED(env, status, Buffer<T>());
  return Buffer(env, value, length, data);
}

template <typename T>
template <typename Finalizer>
inline Buffer<T> Buffer<T>::New(napi_env env,
                                T* data,
                                size_t length,
                                Finalizer finalizeCallback) {
  napi_value value;
  details::FinalizeData<T, Finalizer>* finalizeData =
    new details::FinalizeData<T, Finalizer>({ finalizeCallback, nullptr });
  napi_status status = napi_create_external_buffer(
    env,
    length * sizeof (T),
    data,
    details::FinalizeData<T, Finalizer>::Wrapper,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, Buffer());
  }
  return Buffer(env, value, length, data);
}

template <typename T>
template <typename Finalizer, typename Hint>
inline Buffer<T> Buffer<T>::New(napi_env env,
                                T* data,
                                size_t length,
                                Finalizer finalizeCallback,
                                Hint* finalizeHint) {
  napi_value value;
  details::FinalizeData<T, Finalizer, Hint>* finalizeData =
    new details::FinalizeData<T, Finalizer, Hint>({ finalizeCallback, finalizeHint });
  napi_status status = napi_create_external_buffer(
    env,
    length * sizeof (T),
    data,
    details::FinalizeData<T, Finalizer, Hint>::WrapperWithHint,
    finalizeData,
    &value);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, Buffer());
  }
  return Buffer(env, value, length, data);
}

template <typename T>
inline Buffer<T> Buffer<T>::Copy(napi_env env, const T* data, size_t length) {
  napi_value value;
  napi_status status = napi_create_buffer_copy(
    env, length * sizeof (T), data, nullptr, &value);
  NAPI_THROW_IF_FAILED(env, status, Buffer<T>());
  return Buffer<T>(env, value);
}

template <typename T>
inline Buffer<T>::Buffer() : Uint8Array(), _length(0), _data(nullptr) {
}

template <typename T>
inline Buffer<T>::Buffer(napi_env env, napi_value value)
  : Uint8Array(env, value), _length(0), _data(nullptr) {
}

template <typename T>
inline Buffer<T>::Buffer(napi_env env, napi_value value, size_t length, T* data)
  : Uint8Array(env, value), _length(length), _data(data) {
}

template <typename T>
inline size_t Buffer<T>::Length() const {
  EnsureInfo();
  return _length;
}

template <typename T>
inline T* Buffer<T>::Data() const {
  EnsureInfo();
  return _data;
}

template <typename T>
inline void Buffer<T>::EnsureInfo() const {
  // The Buffer instance may have been constructed from a napi_value whose
  // length/data are not yet known. Fetch and cache these values just once,
  // since they can never change during the lifetime of the Buffer.
  if (_data == nullptr) {
    size_t byteLength;
    void* voidData;
    napi_status status = napi_get_buffer_info(_env, _value, &voidData, &byteLength);
    NAPI_THROW_IF_FAILED_VOID(_env, status);
    _length = byteLength / sizeof (T);
    _data = static_cast<T*>(voidData);
  }
}

////////////////////////////////////////////////////////////////////////////////
// Error class
////////////////////////////////////////////////////////////////////////////////

inline Error Error::New(napi_env env) {
  napi_status status;
  napi_value error = nullptr;

  const napi_extended_error_info* info;
  status = napi_get_last_error_info(env, &info);
  NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_get_last_error_info");

  if (info->error_code == napi_pending_exception) {
    status = napi_get_and_clear_last_exception(env, &error);
    NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_get_and_clear_last_exception");
  }
  else {
    const char* error_message = info->error_message != nullptr ?
      info->error_message : "Error in native callback";

    bool isExceptionPending;
    status = napi_is_exception_pending(env, &isExceptionPending);
    NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_is_exception_pending");

    if (isExceptionPending) {
      status = napi_get_and_clear_last_exception(env, &error);
      NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_get_and_clear_last_exception");
    }

    napi_value message;
    status = napi_create_string_utf8(
      env,
      error_message,
      std::strlen(error_message),
      &message);
    NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_create_string_utf8");

    switch (info->error_code) {
    case napi_object_expected:
    case napi_string_expected:
    case napi_boolean_expected:
    case napi_number_expected:
      status = napi_create_type_error(env, nullptr, message, &error);
      break;
    default:
      status = napi_create_error(env, nullptr,  message, &error);
      break;
    }
    NAPI_FATAL_IF_FAILED(status, "Error::New", "napi_create_error");
  }

  return Error(env, error);
}

inline Error Error::New(napi_env env, const char* message) {
  return Error::New<Error>(env, message, std::strlen(message), napi_create_error);
}

inline Error Error::New(napi_env env, const std::string& message) {
  return Error::New<Error>(env, message.c_str(), message.size(), napi_create_error);
}

inline NAPI_NO_RETURN void Error::Fatal(const char* location, const char* message) {
  napi_fatal_error(location, NAPI_AUTO_LENGTH, message, NAPI_AUTO_LENGTH);
}

inline Error::Error() : ObjectReference() {
}

inline Error::Error(napi_env env, napi_value value) : ObjectReference(env, nullptr) {
  if (value != nullptr) {
    napi_status status = napi_create_reference(env, value, 1, &_ref);

    // Avoid infinite recursion in the failure case.
    // Don't try to construct & throw another Error instance.
    NAPI_FATAL_IF_FAILED(status, "Error::Error", "napi_create_reference");
  }
}

inline Error::Error(Error&& other) : ObjectReference(std::move(other)) {
}

inline Error& Error::operator =(Error&& other) {
  static_cast<Reference<Object>*>(this)->operator=(std::move(other));
  return *this;
}

inline Error::Error(const Error& other) : ObjectReference(other) {
}

inline Error& Error::operator =(Error& other) {
  Reset();

  _env = other.Env();
  HandleScope scope(_env);

  napi_value value = other.Value();
  if (value != nullptr) {
    napi_status status = napi_create_reference(_env, value, 1, &_ref);
    NAPI_THROW_IF_FAILED(_env, status, *this);
  }

  return *this;
}

inline const std::string& Error::Message() const NAPI_NOEXCEPT {
  if (_message.size() == 0 && _env != nullptr) {
#ifdef NAPI_CPP_EXCEPTIONS
    try {
      _message = Get("message").As<String>();
    }
    catch (...) {
      // Catch all errors here, to include e.g. a std::bad_alloc from
      // the std::string::operator=, because this method may not throw.
    }
#else // NAPI_CPP_EXCEPTIONS
    _message = Get("message").As<String>();
#endif // NAPI_CPP_EXCEPTIONS
  }
  return _message;
}

inline void Error::ThrowAsJavaScriptException() const {
  HandleScope scope(_env);
  if (!IsEmpty()) {

    // We intentionally don't use `NAPI_THROW_*` macros here to ensure
    // that there is no possible recursion as `ThrowAsJavaScriptException`
    // is part of `NAPI_THROW_*` macro definition for noexcept.

    napi_status status = napi_throw(_env, Value());

#ifdef NAPI_CPP_EXCEPTIONS
    if (status != napi_ok) {
      throw Error::New(_env);
    }
#else // NAPI_CPP_EXCEPTIONS
    NAPI_FATAL_IF_FAILED(status, "Error::ThrowAsJavaScriptException", "napi_throw");
#endif // NAPI_CPP_EXCEPTIONS
  }
}

#ifdef NAPI_CPP_EXCEPTIONS

inline const char* Error::what() const NAPI_NOEXCEPT {
  return Message().c_str();
}

#endif // NAPI_CPP_EXCEPTIONS

template <typename TError>
inline TError Error::New(napi_env env,
                         const char* message,
                         size_t length,
                         create_error_fn create_error) {
  napi_value str;
  napi_status status = napi_create_string_utf8(env, message, length, &str);
  NAPI_THROW_IF_FAILED(env, status, TError());

  napi_value error;
  status = create_error(env, nullptr, str, &error);
  NAPI_THROW_IF_FAILED(env, status, TError());

  return TError(env, error);
}

inline TypeError TypeError::New(napi_env env, const char* message) {
  return Error::New<TypeError>(env, message, std::strlen(message), napi_create_type_error);
}

inline TypeError TypeError::New(napi_env env, const std::string& message) {
  return Error::New<TypeError>(env, message.c_str(), message.size(), napi_create_type_error);
}

inline TypeError::TypeError() : Error() {
}

inline TypeError::TypeError(napi_env env, napi_value value) : Error(env, value) {
}

inline RangeError RangeError::New(napi_env env, const char* message) {
  return Error::New<RangeError>(env, message, std::strlen(message), napi_create_range_error);
}

inline RangeError RangeError::New(napi_env env, const std::string& message) {
  return Error::New<RangeError>(env, message.c_str(), message.size(), napi_create_range_error);
}

inline RangeError::RangeError() : Error() {
}

inline RangeError::RangeError(napi_env env, napi_value value) : Error(env, value) {
}

////////////////////////////////////////////////////////////////////////////////
// Reference<T> class
////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline Reference<T> Reference<T>::New(const T& value, uint32_t initialRefcount) {
  napi_env env = value.Env();
  napi_value val = value;

  if (val == nullptr) {
    return Reference<T>(env, nullptr);
  }

  napi_ref ref;
  napi_status status = napi_create_reference(env, value, initialRefcount, &ref);
  NAPI_THROW_IF_FAILED(env, status, Reference<T>());

  return Reference<T>(env, ref);
}


template <typename T>
inline Reference<T>::Reference() : _env(nullptr), _ref(nullptr), _suppressDestruct(false) {
}

template <typename T>
inline Reference<T>::Reference(napi_env env, napi_ref ref)
  : _env(env), _ref(ref), _suppressDestruct(false) {
}

template <typename T>
inline Reference<T>::~Reference() {
  if (_ref != nullptr) {
    if (!_suppressDestruct) {
      napi_delete_reference(_env, _ref);
    }

    _ref = nullptr;
  }
}

template <typename T>
inline Reference<T>::Reference(Reference<T>&& other)
  : _env(other._env), _ref(other._ref), _suppressDestruct(other._suppressDestruct) {
  other._env = nullptr;
  other._ref = nullptr;
  other._suppressDestruct = false;
}

template <typename T>
inline Reference<T>& Reference<T>::operator =(Reference<T>&& other) {
  Reset();
  _env = other._env;
  _ref = other._ref;
  _suppressDestruct = other._suppressDestruct;
  other._env = nullptr;
  other._ref = nullptr;
  other._suppressDestruct = false;
  return *this;
}

template <typename T>
inline Reference<T>::Reference(const Reference<T>& other)
  : _env(other._env), _ref(nullptr), _suppressDestruct(false) {
  HandleScope scope(_env);

  napi_value value = other.Value();
  if (value != nullptr) {
    // Copying is a limited scenario (currently only used for Error object) and always creates a
    // strong reference to the given value even if the incoming reference is weak.
    napi_status status = napi_create_reference(_env, value, 1, &_ref);
    NAPI_FATAL_IF_FAILED(status, "Reference<T>::Reference", "napi_create_reference");
  }
}

template <typename T>
inline Reference<T>::operator napi_ref() const {
  return _ref;
}

template <typename T>
inline bool Reference<T>::operator ==(const Reference<T> &other) const {
  HandleScope scope(_env);
  return this->Value().StrictEquals(other.Value());
}

template <typename T>
inline bool Reference<T>::operator !=(const Reference<T> &other) const {
  return !this->operator ==(other);
}

template <typename T>
inline Napi::Env Reference<T>::Env() const {
  return Napi::Env(_env);
}

template <typename T>
inline bool Reference<T>::IsEmpty() const {
  return _ref == nullptr;
}

template <typename T>
inline T Reference<T>::Value() const {
  if (_ref == nullptr) {
    return T(_env, nullptr);
  }

  napi_value value;
  napi_status status = napi_get_reference_value(_env, _ref, &value);
  NAPI_THROW_IF_FAILED(_env, status, T());
  return T(_env, value);
}

template <typename T>
inline uint32_t Reference<T>::Ref() {
  uint32_t result;
  napi_status status = napi_reference_ref(_env, _ref, &result);
  NAPI_THROW_IF_FAILED(_env, status, 1);
  return result;
}

template <typename T>
inline uint32_t Reference<T>::Unref() {
  uint32_t result;
  napi_status status = napi_reference_unref(_env, _ref, &result);
  NAPI_THROW_IF_FAILED(_env, status, 1);
  return result;
}

template <typename T>
inline void Reference<T>::Reset() {
  if (_ref != nullptr) {
    napi_status status = napi_delete_reference(_env, _ref);
    NAPI_THROW_IF_FAILED_VOID(_env, status);
    _ref = nullptr;
  }
}

template <typename T>
inline void Reference<T>::Reset(const T& value, uint32_t refcount) {
  Reset();
  _env = value.Env();

  napi_value val = value;
  if (val != nullptr) {
    napi_status status = napi_create_reference(_env, value, refcount, &_ref);
    NAPI_THROW_IF_FAILED_VOID(_env, status);
  }
}

template <typename T>
inline void Reference<T>::SuppressDestruct() {
  _suppressDestruct = true;
}

template <typename T>
inline Reference<T> Weak(T value) {
  return Reference<T>::New(value, 0);
}

inline ObjectReference Weak(Object value) {
  return Reference<Object>::New(value, 0);
}

inline FunctionReference Weak(Function value) {
  return Reference<Function>::New(value, 0);
}

template <typename T>
inline Reference<T> Persistent(T value) {
  return Reference<T>::New(value, 1);
}

inline ObjectReference Persistent(Object value) {
  return Reference<Object>::New(value, 1);
}

inline FunctionReference Persistent(Function value) {
  return Reference<Function>::New(value, 1);
}

////////////////////////////////////////////////////////////////////////////////
// ObjectReference class
////////////////////////////////////////////////////////////////////////////////

inline ObjectReference::ObjectReference(): Reference<Object>() {
}

inline ObjectReference::ObjectReference(napi_env env, napi_ref ref): Reference<Object>(env, ref) {
}

inline ObjectReference::ObjectReference(Reference<Object>&& other)
  : Reference<Object>(std::move(other)) {
}

inline ObjectReference& ObjectReference::operator =(Reference<Object>&& other) {
  static_cast<Reference<Object>*>(this)->operator=(std::move(other));
  return *this;
}

inline ObjectReference::ObjectReference(ObjectReference&& other)
  : Reference<Object>(std::move(other)) {
}

inline ObjectReference& ObjectReference::operator =(ObjectReference&& other) {
  static_cast<Reference<Object>*>(this)->operator=(std::move(other));
  return *this;
}

inline ObjectReference::ObjectReference(const ObjectReference& other)
  : Reference<Object>(other) {
}

inline Napi::Value ObjectReference::Get(const char* utf8name) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value().Get(utf8name));
}

inline Napi::Value ObjectReference::Get(const std::string& utf8name) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value().Get(utf8name));
}

inline void ObjectReference::Set(const char* utf8name, napi_value value) {
  HandleScope scope(_env);
  Value().Set(utf8name, value);
}

inline void ObjectReference::Set(const char* utf8name, Napi::Value value) {
  HandleScope scope(_env);
  Value().Set(utf8name, value);
}

inline void ObjectReference::Set(const char* utf8name, const char* utf8value) {
  HandleScope scope(_env);
  Value().Set(utf8name, utf8value);
}

inline void ObjectReference::Set(const char* utf8name, bool boolValue) {
  HandleScope scope(_env);
  Value().Set(utf8name, boolValue);
}

inline void ObjectReference::Set(const char* utf8name, double numberValue) {
  HandleScope scope(_env);
  Value().Set(utf8name, numberValue);
}

inline void ObjectReference::Set(const std::string& utf8name, napi_value value) {
  HandleScope scope(_env);
  Value().Set(utf8name, value);
}

inline void ObjectReference::Set(const std::string& utf8name, Napi::Value value) {
  HandleScope scope(_env);
  Value().Set(utf8name, value);
}

inline void ObjectReference::Set(const std::string& utf8name, std::string& utf8value) {
  HandleScope scope(_env);
  Value().Set(utf8name, utf8value);
}

inline void ObjectReference::Set(const std::string& utf8name, bool boolValue) {
  HandleScope scope(_env);
  Value().Set(utf8name, boolValue);
}

inline void ObjectReference::Set(const std::string& utf8name, double numberValue) {
  HandleScope scope(_env);
  Value().Set(utf8name, numberValue);
}

inline Napi::Value ObjectReference::Get(uint32_t index) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value().Get(index));
}

inline void ObjectReference::Set(uint32_t index, napi_value value) {
  HandleScope scope(_env);
  Value().Set(index, value);
}

inline void ObjectReference::Set(uint32_t index, Napi::Value value) {
  HandleScope scope(_env);
  Value().Set(index, value);
}

inline void ObjectReference::Set(uint32_t index, const char* utf8value) {
  HandleScope scope(_env);
  Value().Set(index, utf8value);
}

inline void ObjectReference::Set(uint32_t index, const std::string& utf8value) {
  HandleScope scope(_env);
  Value().Set(index, utf8value);
}

inline void ObjectReference::Set(uint32_t index, bool boolValue) {
  HandleScope scope(_env);
  Value().Set(index, boolValue);
}

inline void ObjectReference::Set(uint32_t index, double numberValue) {
  HandleScope scope(_env);
  Value().Set(index, numberValue);
}

////////////////////////////////////////////////////////////////////////////////
// FunctionReference class
////////////////////////////////////////////////////////////////////////////////

inline FunctionReference::FunctionReference(): Reference<Function>() {
}

inline FunctionReference::FunctionReference(napi_env env, napi_ref ref)
  : Reference<Function>(env, ref) {
}

inline FunctionReference::FunctionReference(Reference<Function>&& other)
  : Reference<Function>(std::move(other)) {
}

inline FunctionReference& FunctionReference::operator =(Reference<Function>&& other) {
  static_cast<Reference<Function>*>(this)->operator=(std::move(other));
  return *this;
}

inline FunctionReference::FunctionReference(FunctionReference&& other)
  : Reference<Function>(std::move(other)) {
}

inline FunctionReference& FunctionReference::operator =(FunctionReference&& other) {
  static_cast<Reference<Function>*>(this)->operator=(std::move(other));
  return *this;
}

inline Napi::Value FunctionReference::operator ()(
    const std::initializer_list<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value()(args));
}

inline Napi::Value FunctionReference::Call(const std::initializer_list<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().Call(args);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::Call(const std::vector<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().Call(args);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::Call(
    napi_value recv, const std::initializer_list<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().Call(recv, args);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::Call(
    napi_value recv, const std::vector<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().Call(recv, args);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::Call(
    napi_value recv, size_t argc, const napi_value* args) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().Call(recv, argc, args);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::MakeCallback(
    napi_value recv,
    const std::initializer_list<napi_value>& args,
    napi_async_context context) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().MakeCallback(recv, args, context);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::MakeCallback(
    napi_value recv,
    const std::vector<napi_value>& args,
    napi_async_context context) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().MakeCallback(recv, args, context);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Napi::Value FunctionReference::MakeCallback(
    napi_value recv,
    size_t argc,
    const napi_value* args,
    napi_async_context context) const {
  EscapableHandleScope scope(_env);
  Napi::Value result = Value().MakeCallback(recv, argc, args, context);
  if (scope.Env().IsExceptionPending()) {
    return Value();
  }
  return scope.Escape(result);
}

inline Object FunctionReference::New(const std::initializer_list<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value().New(args)).As<Object>();
}

inline Object FunctionReference::New(const std::vector<napi_value>& args) const {
  EscapableHandleScope scope(_env);
  return scope.Escape(Value().New(args)).As<Object>();
}

////////////////////////////////////////////////////////////////////////////////
// CallbackInfo class
////////////////////////////////////////////////////////////////////////////////

inline CallbackInfo::CallbackInfo(napi_env env, napi_callback_info info)
    : _env(env), _info(info), _this(nullptr), _dynamicArgs(nullptr), _data(nullptr) {
  _argc = _staticArgCount;
  _argv = _staticArgs;
  napi_status status = napi_get_cb_info(env, info, &_argc, _argv, &_this, &_data);
  NAPI_THROW_IF_FAILED_VOID(_env, status);

  if (_argc > _staticArgCount) {
    // Use either a fixed-size array (on the stack) or a dynamically-allocated
    // array (on the heap) depending on the number of args.
    _dynamicArgs = new napi_value[_argc];
    _argv = _dynamicArgs;

    status = napi_get_cb_info(env, info, &_argc, _argv, nullptr, nullptr);
    NAPI_THROW_IF_FAILED_VOID(_env, status);
  }
}

inline CallbackInfo::~CallbackInfo() {
  if (_dynamicArgs != nullptr) {
    delete[] _dynamicArgs;
  }
}

inline Value CallbackInfo::NewTarget() const {
  napi_value newTarget;
  napi_status status = napi_get_new_target(_env, _info, &newTarget);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, newTarget);
}

inline bool CallbackInfo::IsConstructCall() const {
  return !NewTarget().IsEmpty();
}

inline Napi::Env CallbackInfo::Env() const {
  return Napi::Env(_env);
}

inline size_t CallbackInfo::Length() const {
  return _argc;
}

inline const Value CallbackInfo::operator [](size_t index) const {
  return index < _argc ? Value(_env, _argv[index]) : Env().Undefined();
}

inline Value CallbackInfo::This() const {
  if (_this == nullptr) {
    return Env().Undefined();
  }
  return Object(_env, _this);
}

inline void* CallbackInfo::Data() const {
  return _data;
}

inline void CallbackInfo::SetData(void* data) {
  _data = data;
}

////////////////////////////////////////////////////////////////////////////////
// PropertyDescriptor class
////////////////////////////////////////////////////////////////////////////////

template <typename Getter>
inline PropertyDescriptor
PropertyDescriptor::Accessor(Napi::Env env,
                             Napi::Object object,
                             const char* utf8name,
                             Getter getter,
                             napi_property_attributes attributes,
                             void* data) {
  typedef details::CallbackData<Getter, Napi::Value> CbData;
  auto callbackData = new CbData({ getter, data });

  napi_status status = AttachData(env, object, callbackData);
  if (status != napi_ok) {
    delete callbackData;
    NAPI_THROW_IF_FAILED(env, status, napi_property_descriptor());
  }

  return PropertyDescriptor({
    utf8name,
    nullptr,
    nullptr,
    CbData::Wrapper,
    nullptr,
    nullptr,
    attributes,
    callbackData
  });
}

template <typename Getter>
inline PropertyDescriptor PropertyDescriptor::Accessor(Napi::Env env,
                                                       Napi::Object object,
                                                       const std::string& utf8name,
                                                       Getter getter,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  return Accessor(env, object, utf8name.c_str(), getter, attributes, data);
}

template <typename Getter>
inline PropertyDescriptor PropertyDescriptor::Accessor(Napi::Env env,
                                                       Napi::Object object,
                                                       Name name,
                                                       Getter getter,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  typedef details::CallbackData<Getter, Napi::Value> CbData;
  auto callbackData = new CbData({ getter, data });

  napi_status status = AttachData(env, object, callbackData);
  if (status != napi_ok) {
    delete callbackData;
    NAPI_THROW_IF_FAILED(env, status, napi_property_descriptor());
  }

  return PropertyDescriptor({
    nullptr,
    name,
    nullptr,
    CbData::Wrapper,
    nullptr,
    nullptr,
    attributes,
    callbackData
  });
}

template <typename Getter, typename Setter>
inline PropertyDescriptor PropertyDescriptor::Accessor(Napi::Env env,
                                                       Napi::Object object,
                                                       const char* utf8name,
                                                       Getter getter,
                                                       Setter setter,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  typedef details::AccessorCallbackData<Getter, Setter> CbData;
  auto callbackData = new CbData({ getter, setter, data });

  napi_status status = AttachData(env, object, callbackData);
  if (status != napi_ok) {
    delete callbackData;
    NAPI_THROW_IF_FAILED(env, status, napi_property_descriptor());
  }

  return PropertyDescriptor({
    utf8name,
    nullptr,
    nullptr,
    CbData::GetterWrapper,
    CbData::SetterWrapper,
    nullptr,
    attributes,
    callbackData
  });
}

template <typename Getter, typename Setter>
inline PropertyDescriptor PropertyDescriptor::Accessor(Napi::Env env,
                                                       Napi::Object object,
                                                       const std::string& utf8name,
                                                       Getter getter,
                                                       Setter setter,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  return Accessor(env, object, utf8name.c_str(), getter, setter, attributes, data);
}

template <typename Getter, typename Setter>
inline PropertyDescriptor PropertyDescriptor::Accessor(Napi::Env env,
                                                       Napi::Object object,
                                                       Name name,
                                                       Getter getter,
                                                       Setter setter,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  typedef details::AccessorCallbackData<Getter, Setter> CbData;
  auto callbackData = new CbData({ getter, setter, data });

  napi_status status = AttachData(env, object, callbackData);
  if (status != napi_ok) {
    delete callbackData;
    NAPI_THROW_IF_FAILED(env, status, napi_property_descriptor());
  }

  return PropertyDescriptor({
    nullptr,
    name,
    nullptr,
    CbData::GetterWrapper,
    CbData::SetterWrapper,
    nullptr,
    attributes,
    callbackData
  });
}

template <typename Callable>
inline PropertyDescriptor PropertyDescriptor::Function(Napi::Env env,
                                                       Napi::Object /*object*/,
                                                       const char* utf8name,
                                                       Callable cb,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  return PropertyDescriptor({
    utf8name,
    nullptr,
    nullptr,
    nullptr,
    nullptr,
    Napi::Function::New(env, cb, utf8name, data),
    attributes,
    nullptr
  });
}

template <typename Callable>
inline PropertyDescriptor PropertyDescriptor::Function(Napi::Env env,
                                                       Napi::Object object,
                                                       const std::string& utf8name,
                                                       Callable cb,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  return Function(env, object, utf8name.c_str(), cb, attributes, data);
}

template <typename Callable>
inline PropertyDescriptor PropertyDescriptor::Function(Napi::Env env,
                                                       Napi::Object /*object*/,
                                                       Name name,
                                                       Callable cb,
                                                       napi_property_attributes attributes,
                                                       void* data) {
  return PropertyDescriptor({
    nullptr,
    name,
    nullptr,
    nullptr,
    nullptr,
    Napi::Function::New(env, cb, nullptr, data),
    attributes,
    nullptr
  });
}

inline PropertyDescriptor PropertyDescriptor::Value(const char* utf8name,
                                                    napi_value value,
                                                    napi_property_attributes attributes) {
  return PropertyDescriptor({
    utf8name, nullptr, nullptr, nullptr, nullptr, value, attributes, nullptr
  });
}

inline PropertyDescriptor PropertyDescriptor::Value(const std::string& utf8name,
                                                    napi_value value,
                                                    napi_property_attributes attributes) {
  return Value(utf8name.c_str(), value, attributes);
}

inline PropertyDescriptor PropertyDescriptor::Value(napi_value name,
                                                    napi_value value,
                                                    napi_property_attributes attributes) {
  return PropertyDescriptor({
    nullptr, name, nullptr, nullptr, nullptr, value, attributes, nullptr
  });
}

inline PropertyDescriptor PropertyDescriptor::Value(Name name,
                                                    Napi::Value value,
                                                    napi_property_attributes attributes) {
  napi_value nameValue = name;
  napi_value valueValue = value;
  return PropertyDescriptor::Value(nameValue, valueValue, attributes);
}

inline PropertyDescriptor::PropertyDescriptor(napi_property_descriptor desc)
  : _desc(desc) {
}

inline PropertyDescriptor::operator napi_property_descriptor&() {
  return _desc;
}

inline PropertyDescriptor::operator const napi_property_descriptor&() const {
  return _desc;
}

////////////////////////////////////////////////////////////////////////////////
// ObjectWrap<T> class
////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline ObjectWrap<T>::ObjectWrap(const Napi::CallbackInfo& callbackInfo) {
  napi_env env = callbackInfo.Env();
  napi_value wrapper = callbackInfo.This();
  napi_status status;
  napi_ref ref;
  status = napi_wrap(env, wrapper, this, FinalizeCallback, nullptr, &ref);
  NAPI_THROW_IF_FAILED_VOID(env, status);

  Reference<Object>* instanceRef = this;
  *instanceRef = Reference<Object>(env, ref);
}

template <typename T>
inline ObjectWrap<T>::~ObjectWrap() {
  // If the JS object still exists at this point, remove the finalizer added
  // through `napi_wrap()`.
  if (!IsEmpty()) {
    Object object = Value();
    // It is not valid to call `napi_remove_wrap()` with an empty `object`.
    // This happens e.g. during garbage collection.
    if (!object.IsEmpty() && _construction_failed) {
      napi_remove_wrap(Env(), object, nullptr);

      if (Napi::details::needs_objectwrap_destructor_fix) {
        // If construction failed we delete the reference via
        // `napi_remove_wrap()`, not via `napi_delete_reference()` in the
        // `Reference<Object>` destructor. This will prevent the
        // `Reference<Object>` destructor from doing a double delete of this
        // reference.
        _ref = nullptr;
        _env = nullptr;
      }
    }
  }
}

template<typename T>
inline T* ObjectWrap<T>::Unwrap(Object wrapper) {
  T* unwrapped;
  napi_status status = napi_unwrap(wrapper.Env(), wrapper, reinterpret_cast<void**>(&unwrapped));
  NAPI_THROW_IF_FAILED(wrapper.Env(), status, nullptr);
  return unwrapped;
}

template <typename T>
inline Function
ObjectWrap<T>::DefineClass(Napi::Env env,
                           const char* utf8name,
                           const size_t props_count,
                           const napi_property_descriptor* descriptors,
                           void* data) {
  napi_status status;
  std::vector<napi_property_descriptor> props(props_count);

  // We copy the descriptors to a local array because before defining the class
  // we must replace static method property descriptors with value property
  // descriptors such that the value is a function-valued `napi_value` created
  // with `CreateFunction()`.
  //
  // This replacement could be made for instance methods as well, but V8 aborts
  // if we do that, because it expects methods defined on the prototype template
  // to have `FunctionTemplate`s.
  for (size_t index = 0; index < props_count; index++) {
    props[index] = descriptors[index];
    napi_property_descriptor* prop = &props[index];
    if (prop->method == T::StaticMethodCallbackWrapper) {
      status = CreateFunction(env,
               utf8name,
               prop->method,
               static_cast<StaticMethodCallbackData*>(prop->data),
               &(prop->value));
      NAPI_THROW_IF_FAILED(env, status, Function());
      prop->method = nullptr;
      prop->data = nullptr;
    } else if (prop->method == T::StaticVoidMethodCallbackWrapper) {
      status = CreateFunction(env,
               utf8name,
               prop->method,
               static_cast<StaticVoidMethodCallbackData*>(prop->data),
               &(prop->value));
      NAPI_THROW_IF_FAILED(env, status, Function());
      prop->method = nullptr;
      prop->data = nullptr;
    }
  }

  napi_value value;
  status = napi_define_class(env,
                             utf8name,
                             NAPI_AUTO_LENGTH,
                             T::ConstructorCallbackWrapper,
                             data,
                             props_count,
                             props.data(),
                             &value);
  NAPI_THROW_IF_FAILED(env, status, Function());

  // After defining the class we iterate once more over the property descriptors
  // and attach the data associated with accessors and instance methods to the
  // newly created JavaScript class.
  for (size_t idx = 0; idx < props_count; idx++) {
    const napi_property_descriptor* prop = &props[idx];

    if (prop->getter == T::StaticGetterCallbackWrapper ||
        prop->setter == T::StaticSetterCallbackWrapper) {
      status = Napi::details::AttachData(env,
                          value,
                          static_cast<StaticAccessorCallbackData*>(prop->data));
      NAPI_THROW_IF_FAILED(env, status, Function());
    } else if (prop->getter == T::InstanceGetterCallbackWrapper ||
        prop->setter == T::InstanceSetterCallbackWrapper) {
      status = Napi::details::AttachData(env,
                          value,
                          static_cast<InstanceAccessorCallbackData*>(prop->data));
      NAPI_THROW_IF_FAILED(env, status, Function());
    } else if (prop->method != nullptr && !(prop->attributes & napi_static)) {
      if (prop->method == T::InstanceVoidMethodCallbackWrapper) {
        status = Napi::details::AttachData(env,
                      value,
                      static_cast<InstanceVoidMethodCallbackData*>(prop->data));
        NAPI_THROW_IF_FAILED(env, status, Function());
      } else if (prop->method == T::InstanceMethodCallbackWrapper) {
        status = Napi::details::AttachData(env,
                          value,
                          static_cast<InstanceMethodCallbackData*>(prop->data));
        NAPI_THROW_IF_FAILED(env, status, Function());
      }
    }
  }

  return Function(env, value);
}

template <typename T>
inline Function ObjectWrap<T>::DefineClass(
    Napi::Env env,
    const char* utf8name,
    const std::initializer_list<ClassPropertyDescriptor<T>>& properties,
    void* data) {
  return DefineClass(env,
          utf8name,
          properties.size(),
          reinterpret_cast<const napi_property_descriptor*>(properties.begin()),
          data);
}

template <typename T>
inline Function ObjectWrap<T>::DefineClass(
    Napi::Env env,
    const char* utf8name,
    const std::vector<ClassPropertyDescriptor<T>>& properties,
    void* data) {
  return DefineClass(env,
           utf8name,
           properties.size(),
           reinterpret_cast<const napi_property_descriptor*>(properties.data()),
           data);
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticMethod(
    const char* utf8name,
    StaticVoidMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  StaticVoidMethodCallbackData* callbackData = new StaticVoidMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.method = T::StaticVoidMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticMethod(
    const char* utf8name,
    StaticMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  StaticMethodCallbackData* callbackData = new StaticMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.method = T::StaticMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticMethod(
    Symbol name,
    StaticVoidMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  StaticVoidMethodCallbackData* callbackData = new StaticVoidMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.method = T::StaticVoidMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticMethod(
    Symbol name,
    StaticMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  StaticMethodCallbackData* callbackData = new StaticMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.method = T::StaticMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticAccessor(
    const char* utf8name,
    StaticGetterCallback getter,
    StaticSetterCallback setter,
    napi_property_attributes attributes,
    void* data) {
  StaticAccessorCallbackData* callbackData =
    new StaticAccessorCallbackData({ getter, setter, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.getter = getter != nullptr ? T::StaticGetterCallbackWrapper : nullptr;
  desc.setter = setter != nullptr ? T::StaticSetterCallbackWrapper : nullptr;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticAccessor(
    Symbol name,
    StaticGetterCallback getter,
    StaticSetterCallback setter,
    napi_property_attributes attributes,
    void* data) {
  StaticAccessorCallbackData* callbackData =
    new StaticAccessorCallbackData({ getter, setter, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.getter = getter != nullptr ? T::StaticGetterCallbackWrapper : nullptr;
  desc.setter = setter != nullptr ? T::StaticSetterCallbackWrapper : nullptr;
  desc.data = callbackData;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceMethod(
    const char* utf8name,
    InstanceVoidMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  InstanceVoidMethodCallbackData* callbackData =
    new InstanceVoidMethodCallbackData({ method, data});

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.method = T::InstanceVoidMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceMethod(
    const char* utf8name,
    InstanceMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  InstanceMethodCallbackData* callbackData = new InstanceMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.method = T::InstanceMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceMethod(
    Symbol name,
    InstanceVoidMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  InstanceVoidMethodCallbackData* callbackData =
    new InstanceVoidMethodCallbackData({ method, data});

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.method = T::InstanceVoidMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceMethod(
    Symbol name,
    InstanceMethodCallback method,
    napi_property_attributes attributes,
    void* data) {
  InstanceMethodCallbackData* callbackData = new InstanceMethodCallbackData({ method, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.method = T::InstanceMethodCallbackWrapper;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceAccessor(
    const char* utf8name,
    InstanceGetterCallback getter,
    InstanceSetterCallback setter,
    napi_property_attributes attributes,
    void* data) {
  InstanceAccessorCallbackData* callbackData =
    new InstanceAccessorCallbackData({ getter, setter, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.getter = getter != nullptr ? T::InstanceGetterCallbackWrapper : nullptr;
  desc.setter = setter != nullptr ? T::InstanceSetterCallbackWrapper : nullptr;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceAccessor(
    Symbol name,
    InstanceGetterCallback getter,
    InstanceSetterCallback setter,
    napi_property_attributes attributes,
    void* data) {
  InstanceAccessorCallbackData* callbackData =
    new InstanceAccessorCallbackData({ getter, setter, data });

  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.getter = getter != nullptr ? T::InstanceGetterCallbackWrapper : nullptr;
  desc.setter = setter != nullptr ? T::InstanceSetterCallbackWrapper : nullptr;
  desc.data = callbackData;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticValue(const char* utf8name,
    Napi::Value value, napi_property_attributes attributes) {
  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.value = value;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::StaticValue(Symbol name,
    Napi::Value value, napi_property_attributes attributes) {
  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.value = value;
  desc.attributes = static_cast<napi_property_attributes>(attributes | napi_static);
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceValue(
    const char* utf8name,
    Napi::Value value,
    napi_property_attributes attributes) {
  napi_property_descriptor desc = napi_property_descriptor();
  desc.utf8name = utf8name;
  desc.value = value;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline ClassPropertyDescriptor<T> ObjectWrap<T>::InstanceValue(
    Symbol name,
    Napi::Value value,
    napi_property_attributes attributes) {
  napi_property_descriptor desc = napi_property_descriptor();
  desc.name = name;
  desc.value = value;
  desc.attributes = attributes;
  return desc;
}

template <typename T>
inline void ObjectWrap<T>::Finalize(Napi::Env /*env*/) {}

template <typename T>
inline napi_value ObjectWrap<T>::ConstructorCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  napi_value new_target;
  napi_status status = napi_get_new_target(env, info, &new_target);
  if (status != napi_ok) return nullptr;

  bool isConstructCall = (new_target != nullptr);
  if (!isConstructCall) {
    napi_throw_type_error(env, nullptr, "Class constructors cannot be invoked without 'new'");
    return nullptr;
  }

  napi_value wrapper = details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    T* instance = new T(callbackInfo);
#ifdef NAPI_CPP_EXCEPTIONS
    instance->_construction_failed = false;
#else
    if (callbackInfo.Env().IsExceptionPending()) {
      // We need to clear the exception so that removing the wrap might work.
      Error e = callbackInfo.Env().GetAndClearPendingException();
      delete instance;
      e.ThrowAsJavaScriptException();
    } else {
      instance->_construction_failed = false;
    }
# endif  // NAPI_CPP_EXCEPTIONS
    return callbackInfo.This();
  });

  return wrapper;
}

template <typename T>
inline napi_value ObjectWrap<T>::StaticVoidMethodCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    StaticVoidMethodCallbackData* callbackData =
      reinterpret_cast<StaticVoidMethodCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    callbackData->callback(callbackInfo);
    return nullptr;
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::StaticMethodCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    StaticMethodCallbackData* callbackData =
      reinterpret_cast<StaticMethodCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    return callbackData->callback(callbackInfo);
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::StaticGetterCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    StaticAccessorCallbackData* callbackData =
      reinterpret_cast<StaticAccessorCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    return callbackData->getterCallback(callbackInfo);
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::StaticSetterCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    StaticAccessorCallbackData* callbackData =
      reinterpret_cast<StaticAccessorCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    callbackData->setterCallback(callbackInfo, callbackInfo[0]);
    return nullptr;
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::InstanceVoidMethodCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    InstanceVoidMethodCallbackData* callbackData =
      reinterpret_cast<InstanceVoidMethodCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    T* instance = Unwrap(callbackInfo.This().As<Object>());
    auto cb = callbackData->callback;
    (instance->*cb)(callbackInfo);
    return nullptr;
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::InstanceMethodCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    InstanceMethodCallbackData* callbackData =
      reinterpret_cast<InstanceMethodCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    T* instance = Unwrap(callbackInfo.This().As<Object>());
    auto cb = callbackData->callback;
    return (instance->*cb)(callbackInfo);
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::InstanceGetterCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    InstanceAccessorCallbackData* callbackData =
      reinterpret_cast<InstanceAccessorCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    T* instance = Unwrap(callbackInfo.This().As<Object>());
    auto cb = callbackData->getterCallback;
    return (instance->*cb)(callbackInfo);
  });
}

template <typename T>
inline napi_value ObjectWrap<T>::InstanceSetterCallbackWrapper(
    napi_env env,
    napi_callback_info info) {
  return details::WrapCallback([&] {
    CallbackInfo callbackInfo(env, info);
    InstanceAccessorCallbackData* callbackData =
      reinterpret_cast<InstanceAccessorCallbackData*>(callbackInfo.Data());
    callbackInfo.SetData(callbackData->data);
    T* instance = Unwrap(callbackInfo.This().As<Object>());
    auto cb = callbackData->setterCallback;
    (instance->*cb)(callbackInfo, callbackInfo[0]);
    return nullptr;
  });
}

template <typename T>
inline void ObjectWrap<T>::FinalizeCallback(napi_env env, void* data, void* /*hint*/) {
  ObjectWrap<T>* instance = static_cast<ObjectWrap<T>*>(data);
  instance->Finalize(Napi::Env(env));
  delete instance;
}

////////////////////////////////////////////////////////////////////////////////
// HandleScope class
////////////////////////////////////////////////////////////////////////////////

inline HandleScope::HandleScope(napi_env env, napi_handle_scope scope)
    : _env(env), _scope(scope) {
}

inline HandleScope::HandleScope(Napi::Env env) : _env(env) {
  napi_status status = napi_open_handle_scope(_env, &_scope);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline HandleScope::~HandleScope() {
  napi_status status = napi_close_handle_scope(_env, _scope);
  NAPI_FATAL_IF_FAILED(status,
                       "HandleScope::~HandleScope",
                       "napi_close_handle_scope");
}

inline HandleScope::operator napi_handle_scope() const {
  return _scope;
}

inline Napi::Env HandleScope::Env() const {
  return Napi::Env(_env);
}

////////////////////////////////////////////////////////////////////////////////
// EscapableHandleScope class
////////////////////////////////////////////////////////////////////////////////

inline EscapableHandleScope::EscapableHandleScope(
  napi_env env, napi_escapable_handle_scope scope) : _env(env), _scope(scope) {
}

inline EscapableHandleScope::EscapableHandleScope(Napi::Env env) : _env(env) {
  napi_status status = napi_open_escapable_handle_scope(_env, &_scope);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline EscapableHandleScope::~EscapableHandleScope() {
  napi_status status = napi_close_escapable_handle_scope(_env, _scope);
  NAPI_FATAL_IF_FAILED(status,
                       "EscapableHandleScope::~EscapableHandleScope",
                       "napi_close_escapable_handle_scope");
}

inline EscapableHandleScope::operator napi_escapable_handle_scope() const {
  return _scope;
}

inline Napi::Env EscapableHandleScope::Env() const {
  return Napi::Env(_env);
}

inline Value EscapableHandleScope::Escape(napi_value escapee) {
  napi_value result;
  napi_status status = napi_escape_handle(_env, _scope, escapee, &result);
  NAPI_THROW_IF_FAILED(_env, status, Value());
  return Value(_env, result);
}


#if (NAPI_VERSION > 2)
////////////////////////////////////////////////////////////////////////////////
// CallbackScope class
////////////////////////////////////////////////////////////////////////////////

inline CallbackScope::CallbackScope(
  napi_env env, napi_callback_scope scope) : _env(env), _scope(scope) {
}

inline CallbackScope::CallbackScope(napi_env env, napi_async_context context)
    : _env(env) {
  napi_status status = napi_open_callback_scope(
      _env, Object::New(env), context, &_scope);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline CallbackScope::~CallbackScope() {
  napi_status status = napi_close_callback_scope(_env, _scope);
  NAPI_FATAL_IF_FAILED(status,
                       "CallbackScope::~CallbackScope",
                       "napi_close_callback_scope");
}

inline CallbackScope::operator napi_callback_scope() const {
  return _scope;
}

inline Napi::Env CallbackScope::Env() const {
  return Napi::Env(_env);
}
#endif

////////////////////////////////////////////////////////////////////////////////
// AsyncContext class
////////////////////////////////////////////////////////////////////////////////

inline AsyncContext::AsyncContext(napi_env env, const char* resource_name)
  : AsyncContext(env, resource_name, Object::New(env)) {
}

inline AsyncContext::AsyncContext(napi_env env,
		                  const char* resource_name,
                                  const Object& resource)
  : _env(env),
    _context(nullptr) {
  napi_value resource_id;
  napi_status status = napi_create_string_utf8(
      _env, resource_name, NAPI_AUTO_LENGTH, &resource_id);
  NAPI_THROW_IF_FAILED_VOID(_env, status);

  status = napi_async_init(_env, resource, resource_id, &_context);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline AsyncContext::~AsyncContext() {
  if (_context != nullptr) {
    napi_async_destroy(_env, _context);
    _context = nullptr;
  }
}

inline AsyncContext::AsyncContext(AsyncContext&& other) {
  _env = other._env;
  other._env = nullptr;
  _context = other._context;
  other._context = nullptr;
}

inline AsyncContext& AsyncContext::operator =(AsyncContext&& other) {
  _env = other._env;
  other._env = nullptr;
  _context = other._context;
  other._context = nullptr;
  return *this;
}

inline AsyncContext::operator napi_async_context() const {
  return _context;
}

inline Napi::Env AsyncContext::Env() const {
  return Napi::Env(_env);
}

////////////////////////////////////////////////////////////////////////////////
// AsyncWorker class
////////////////////////////////////////////////////////////////////////////////

inline AsyncWorker::AsyncWorker(const Function& callback)
  : AsyncWorker(callback, "generic") {
}

inline AsyncWorker::AsyncWorker(const Function& callback,
                                const char* resource_name)
  : AsyncWorker(callback, resource_name, Object::New(callback.Env())) {
}

inline AsyncWorker::AsyncWorker(const Function& callback,
                                const char* resource_name,
                                const Object& resource)
  : AsyncWorker(Object::New(callback.Env()),
                callback,
                resource_name,
                resource) {
}

inline AsyncWorker::AsyncWorker(const Object& receiver,
                                const Function& callback)
  : AsyncWorker(receiver, callback, "generic") {
}

inline AsyncWorker::AsyncWorker(const Object& receiver,
                                const Function& callback,
                                const char* resource_name)
  : AsyncWorker(receiver,
                callback,
                resource_name,
                Object::New(callback.Env())) {
}

inline AsyncWorker::AsyncWorker(const Object& receiver,
                                const Function& callback,
                                const char* resource_name,
                                const Object& resource)
  : _env(callback.Env()),
    _receiver(Napi::Persistent(receiver)),
    _callback(Napi::Persistent(callback)),
    _suppress_destruct(false) {
  napi_value resource_id;
  napi_status status = napi_create_string_latin1(
      _env, resource_name, NAPI_AUTO_LENGTH, &resource_id);
  NAPI_THROW_IF_FAILED_VOID(_env, status);

  status = napi_create_async_work(_env, resource, resource_id, OnExecute,
                                  OnWorkComplete, this, &_work);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline AsyncWorker::AsyncWorker(Napi::Env env)
  : AsyncWorker(env, "generic") {
}

inline AsyncWorker::AsyncWorker(Napi::Env env,
                                const char* resource_name)
  : AsyncWorker(env, resource_name, Object::New(env)) {
}

inline AsyncWorker::AsyncWorker(Napi::Env env,
                                const char* resource_name,
                                const Object& resource)
  : _env(env),
    _receiver(),
    _callback(),
    _suppress_destruct(false) {
  napi_value resource_id;
  napi_status status = napi_create_string_latin1(
      _env, resource_name, NAPI_AUTO_LENGTH, &resource_id);
  NAPI_THROW_IF_FAILED_VOID(_env, status);

  status = napi_create_async_work(_env, resource, resource_id, OnExecute,
                                  OnWorkComplete, this, &_work);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline AsyncWorker::~AsyncWorker() {
  if (_work != nullptr) {
    napi_delete_async_work(_env, _work);
    _work = nullptr;
  }
}

inline void AsyncWorker::Destroy() {
  delete this;
}

inline AsyncWorker::AsyncWorker(AsyncWorker&& other) {
  _env = other._env;
  other._env = nullptr;
  _work = other._work;
  other._work = nullptr;
  _receiver = std::move(other._receiver);
  _callback = std::move(other._callback);
  _error = std::move(other._error);
  _suppress_destruct = other._suppress_destruct;
}

inline AsyncWorker& AsyncWorker::operator =(AsyncWorker&& other) {
  _env = other._env;
  other._env = nullptr;
  _work = other._work;
  other._work = nullptr;
  _receiver = std::move(other._receiver);
  _callback = std::move(other._callback);
  _error = std::move(other._error);
  _suppress_destruct = other._suppress_destruct;
  return *this;
}

inline AsyncWorker::operator napi_async_work() const {
  return _work;
}

inline Napi::Env AsyncWorker::Env() const {
  return Napi::Env(_env);
}

inline void AsyncWorker::Queue() {
  napi_status status = napi_queue_async_work(_env, _work);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline void AsyncWorker::Cancel() {
  napi_status status = napi_cancel_async_work(_env, _work);
  NAPI_THROW_IF_FAILED_VOID(_env, status);
}

inline ObjectReference& AsyncWorker::Receiver() {
  return _receiver;
}

inline FunctionReference& AsyncWorker::Callback() {
  return _callback;
}

inline void AsyncWorker::SuppressDestruct() {
  _suppress_destruct = true;
}

inline void AsyncWorker::OnOK() {
  if (!_callback.IsEmpty()) {
    _callback.Call(_receiver.Value(), GetResult(_callback.Env()));
  }
}

inline void AsyncWorker::OnError(const Error& e) {
  if (!_callback.IsEmpty()) {
    _callback.Call(_receiver.Value(), std::initializer_list<napi_value>{ e.Value() });
  }
}

inline void AsyncWorker::SetError(const std::string& error) {
  _error = error;
}

inline std::vector<napi_value> AsyncWorker::GetResult(Napi::Env /*env*/) {
  return {};
}
// The OnExecute method receives an napi_env argument. However, do NOT
// use it within this method, as it does not run on the main thread and must
// not run any method that would cause JavaScript to run. In practice, this
// means that almost any use of napi_env will be incorrect.
inline void AsyncWorker::OnExecute(napi_env /*DO_NOT_USE*/, void* this_pointer) {
  AsyncWorker* self = static_cast<AsyncWorker*>(this_pointer);
#ifdef NAPI_CPP_EXCEPTIONS
  try {
    self->Execute();
  } catch (const std::exception& e) {
    self->SetError(e.what());
  }
#else // NAPI_CPP_EXCEPTIONS
  self->Execute();
#endif // NAPI_CPP_EXCEPTIONS
}

inline void AsyncWorker::OnWorkComplete(
    napi_env /*env*/, napi_status status, void* this_pointer) {
  AsyncWorker* self = static_cast<AsyncWorker*>(this_pointer);
  if (status != napi_cancelled) {
    HandleScope scope(self->_env);
    details::WrapCallback([&] {
      if (self->_error.size() == 0) {
        self->OnOK();
      }
      else {
        self->OnError(Error::New(self->_env, self->_error));
      }
      return nullptr;
    });
  }
  if (!self->_suppress_destruct) {
    self->Destroy();
  }
}

#if (NAPI_VERSION > 3)
////////////////////////////////////////////////////////////////////////////////
// ThreadSafeFunction class
////////////////////////////////////////////////////////////////////////////////

// static
template <typename ResourceString>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount);
}

// static
template <typename ResourceString, typename ContextType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  ContextType* context) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount, context);
}

// static
template <typename ResourceString, typename Finalizer>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  Finalizer finalizeCallback) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount, finalizeCallback);
}

// static
template <typename ResourceString, typename Finalizer,
          typename FinalizerDataType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  Finalizer finalizeCallback,
                                  FinalizerDataType* data) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount, finalizeCallback, data);
}

// static
template <typename ResourceString, typename ContextType, typename Finalizer>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  ContextType* context,
                                  Finalizer finalizeCallback) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount, context, finalizeCallback);
}

// static
template <typename ResourceString, typename ContextType,
          typename Finalizer, typename FinalizerDataType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                                  const Function& callback,
                                                  ResourceString resourceName,
                                                  size_t maxQueueSize,
                                                  size_t initialThreadCount,
                                                  ContextType* context,
                                                  Finalizer finalizeCallback,
                                                  FinalizerDataType* data) {
  return New(env, callback, Object(), resourceName, maxQueueSize,
             initialThreadCount, context, finalizeCallback, data);
}

// static
template <typename ResourceString>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  const Object& resource,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, static_cast<void*>(nullptr) /* context */);
}

// static
template <typename ResourceString, typename ContextType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  const Object& resource,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  ContextType* context) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, context,
             [](Env, ContextType*) {} /* empty finalizer */);
}

// static
template <typename ResourceString, typename Finalizer>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  const Object& resource,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  Finalizer finalizeCallback) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, static_cast<void*>(nullptr) /* context */,
             finalizeCallback, static_cast<void*>(nullptr) /* data */,
             details::ThreadSafeFinalize<void, Finalizer>::Wrapper);
}

// static
template <typename ResourceString, typename Finalizer,
          typename FinalizerDataType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  const Object& resource,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  Finalizer finalizeCallback,
                                  FinalizerDataType* data) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, static_cast<void*>(nullptr) /* context */,
             finalizeCallback, data,
             details::ThreadSafeFinalize<
                 void, Finalizer, FinalizerDataType>::FinalizeWrapperWithData);
}

// static
template <typename ResourceString, typename ContextType, typename Finalizer>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                  const Function& callback,
                                  const Object& resource,
                                  ResourceString resourceName,
                                  size_t maxQueueSize,
                                  size_t initialThreadCount,
                                  ContextType* context,
                                  Finalizer finalizeCallback) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, context, finalizeCallback,
             static_cast<void*>(nullptr) /* data */,
             details::ThreadSafeFinalize<
                 ContextType, Finalizer>::FinalizeWrapperWithContext);
}

// static
template <typename ResourceString, typename ContextType,
          typename Finalizer, typename FinalizerDataType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                                  const Function& callback,
                                                  const Object& resource,
                                                  ResourceString resourceName,
                                                  size_t maxQueueSize,
                                                  size_t initialThreadCount,
                                                  ContextType* context,
                                                  Finalizer finalizeCallback,
                                                  FinalizerDataType* data) {
  return New(env, callback, resource, resourceName, maxQueueSize,
             initialThreadCount, context, finalizeCallback, data,
             details::ThreadSafeFinalize<ContextType, Finalizer,
                 FinalizerDataType>::FinalizeFinalizeWrapperWithDataAndContext);
}

inline ThreadSafeFunction::ThreadSafeFunction()
  : _tsfn() {
}

inline ThreadSafeFunction::ThreadSafeFunction(
    napi_threadsafe_function tsfn)
  : _tsfn(tsfn) {
}

inline ThreadSafeFunction::operator napi_threadsafe_function() const {
  return _tsfn;
}

inline napi_status ThreadSafeFunction::BlockingCall() const {
  return CallInternal(nullptr, napi_tsfn_blocking);
}

template <>
inline napi_status ThreadSafeFunction::BlockingCall(
    void* data) const {
  return napi_call_threadsafe_function(_tsfn, data, napi_tsfn_blocking);
}

template <typename Callback>
inline napi_status ThreadSafeFunction::BlockingCall(
    Callback callback) const {
  return CallInternal(new CallbackWrapper(callback), napi_tsfn_blocking);
}

template <typename DataType, typename Callback>
inline napi_status ThreadSafeFunction::BlockingCall(
    DataType* data, Callback callback) const {
  auto wrapper = [data, callback](Env env, Function jsCallback) {
    callback(env, jsCallback, data);
  };
  return CallInternal(new CallbackWrapper(wrapper), napi_tsfn_blocking);
}

inline napi_status ThreadSafeFunction::NonBlockingCall() const {
  return CallInternal(nullptr, napi_tsfn_nonblocking);
}

template <>
inline napi_status ThreadSafeFunction::NonBlockingCall(
    void* data) const {
  return napi_call_threadsafe_function(_tsfn, data, napi_tsfn_nonblocking);
}

template <typename Callback>
inline napi_status ThreadSafeFunction::NonBlockingCall(
    Callback callback) const {
  return CallInternal(new CallbackWrapper(callback), napi_tsfn_nonblocking);
}

template <typename DataType, typename Callback>
inline napi_status ThreadSafeFunction::NonBlockingCall(
    DataType* data, Callback callback) const {
  auto wrapper = [data, callback](Env env, Function jsCallback) {
    callback(env, jsCallback, data);
  };
  return CallInternal(new CallbackWrapper(wrapper), napi_tsfn_nonblocking);
}

inline void ThreadSafeFunction::Ref(napi_env env) const {
  if (_tsfn != nullptr) {
    napi_status status = napi_ref_threadsafe_function(env, _tsfn);
    NAPI_THROW_IF_FAILED_VOID(env, status);
  }
}

inline void ThreadSafeFunction::Unref(napi_env env) const {
  if (_tsfn != nullptr) {
    napi_status status = napi_unref_threadsafe_function(env, _tsfn);
    NAPI_THROW_IF_FAILED_VOID(env, status);
  }
}

inline napi_status ThreadSafeFunction::Acquire() const {
  return napi_acquire_threadsafe_function(_tsfn);
}

inline napi_status ThreadSafeFunction::Release() {
  return napi_release_threadsafe_function(_tsfn, napi_tsfn_release);
}

inline napi_status ThreadSafeFunction::Abort() {
  return napi_release_threadsafe_function(_tsfn, napi_tsfn_abort);
}

inline ThreadSafeFunction::ConvertibleContext
ThreadSafeFunction::GetContext() const {
  void* context;
  napi_status status = napi_get_threadsafe_function_context(_tsfn, &context);
  NAPI_FATAL_IF_FAILED(status, "ThreadSafeFunction::GetContext", "napi_get_threadsafe_function_context");
  return ConvertibleContext({ context });
}

// static
template <typename ResourceString, typename ContextType,
          typename Finalizer, typename FinalizerDataType>
inline ThreadSafeFunction ThreadSafeFunction::New(napi_env env,
                                                  const Function& callback,
                                                  const Object& resource,
                                                  ResourceString resourceName,
                                                  size_t maxQueueSize,
                                                  size_t initialThreadCount,
                                                  ContextType* context,
                                                  Finalizer finalizeCallback,
                                                  FinalizerDataType* data,
                                                  napi_finalize wrapper) {
  static_assert(details::can_make_string<ResourceString>::value
      || std::is_convertible<ResourceString, napi_value>::value,
      "Resource name should be convertible to the string type");

  ThreadSafeFunction tsfn;
  auto* finalizeData = new details::ThreadSafeFinalize<ContextType, Finalizer,
      FinalizerDataType>({ data, finalizeCallback, &tsfn._tsfn });
  napi_status status = napi_create_threadsafe_function(env, callback, resource,
      Value::From(env, resourceName), maxQueueSize, initialThreadCount,
      finalizeData, wrapper, context, CallJS, &tsfn._tsfn);
  if (status != napi_ok) {
    delete finalizeData;
    NAPI_THROW_IF_FAILED(env, status, ThreadSafeFunction());
  }

  return tsfn;
}

inline napi_status ThreadSafeFunction::CallInternal(
    CallbackWrapper* callbackWrapper,
    napi_threadsafe_function_call_mode mode) const {
  napi_status status = napi_call_threadsafe_function(
      _tsfn, callbackWrapper, mode);
  if (status != napi_ok && callbackWrapper != nullptr) {
    delete callbackWrapper;
  }

  return status;
}

// static
inline void ThreadSafeFunction::CallJS(napi_env env,
                                       napi_value jsCallback,
                                       void* /* context */,
                                       void* data) {
  if (env == nullptr && jsCallback == nullptr) {
    return;
  }

  if (data != nullptr) {
    auto* callbackWrapper = static_cast<CallbackWrapper*>(data);
    (*callbackWrapper)(env, Function(env, jsCallback));
    delete callbackWrapper;
  } else if (jsCallback != nullptr) {
    Function(env, jsCallback).Call({});
  }
}

////////////////////////////////////////////////////////////////////////////////
// Async Progress Worker class
////////////////////////////////////////////////////////////////////////////////

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Function& callback)
  : AsyncProgressWorker(callback, "generic") {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Function& callback,
                                const char* resource_name)
  : AsyncProgressWorker(callback, resource_name, Object::New(callback.Env())) {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Function& callback,
                                const char* resource_name,
                                const Object& resource)
  : AsyncProgressWorker(Object::New(callback.Env()),
                callback,
                resource_name,
                resource) {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Object& receiver,
                                const Function& callback)
  : AsyncProgressWorker(receiver, callback, "generic") {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Object& receiver,
                                const Function& callback,
                                const char* resource_name)
  : AsyncProgressWorker(receiver,
                callback,
                resource_name,
                Object::New(callback.Env())) {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(const Object& receiver,
                                                   const Function& callback,
                                                   const char* resource_name,
                                                   const Object& resource)
  : AsyncWorker(receiver, callback, resource_name, resource),
    _asyncdata(nullptr),
    _asyncsize(0) {
  _tsfn = ThreadSafeFunction::New(callback.Env(), callback, resource_name, 1, 1);
}

#if NAPI_VERSION > 4
template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(Napi::Env env)
  : AsyncProgressWorker(env, "generic") {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(Napi::Env env,
                                const char* resource_name)
  : AsyncProgressWorker(env, resource_name, Object::New(env)) {
}

template<class T>
inline AsyncProgressWorker<T>::AsyncProgressWorker(Napi::Env env,
                                const char* resource_name,
                                const Object& resource)
  : AsyncWorker(env, resource_name, resource),
    _asyncdata(nullptr),
    _asyncsize(0) {
  // TODO: Once the changes to make the callback optional for threadsafe
  // functions are no longer optional we can remove the dummy Function here.
  Function callback;
  _tsfn = ThreadSafeFunction::New(env, callback, resource_name, 1, 1);
}
#endif

template<class T>
inline AsyncProgressWorker<T>::~AsyncProgressWorker() {
  // Abort pending tsfn call.
  // Don't send progress events after we've already completed.
  _tsfn.Abort();
  {
    std::lock_guard<std::mutex> lock(_mutex);
    _asyncdata = nullptr;
    _asyncsize = 0;
  }
  _tsfn.Release();
}

template<class T>
inline void AsyncProgressWorker<T>::Execute() {
  ExecutionProgress progress(this);
  Execute(progress);
}

template<class T>
inline void AsyncProgressWorker<T>::WorkProgress_(Napi::Env /* env */, Napi::Function /* jsCallback */, void* _data) {
  AsyncProgressWorker* self = static_cast<AsyncProgressWorker*>(_data);

  T* data;
  size_t size;
  {
    std::lock_guard<std::mutex> lock(self->_mutex);
    data = self->_asyncdata;
    size = self->_asyncsize;
    self->_asyncdata = nullptr;
    self->_asyncsize = 0;
  }

  self->OnProgress(data, size);
  delete[] data;
}

template<class T>
inline void AsyncProgressWorker<T>::SendProgress_(const T* data, size_t count) {
    T* new_data = new T[count];
    std::copy(data, data + count, new_data);

    T* old_data;
    {
      std::lock_guard<std::mutex> lock(_mutex);
      old_data = _asyncdata;
      _asyncdata = new_data;
      _asyncsize = count;
    }
    _tsfn.NonBlockingCall(this, WorkProgress_);

    delete[] old_data;
}

template<class T>
inline void AsyncProgressWorker<T>::Signal() const {
  _tsfn.NonBlockingCall(this, WorkProgress_);
}

template<class T>
inline void AsyncProgressWorker<T>::ExecutionProgress::Signal() const {
  _worker->Signal();
}

template<class T>
inline void AsyncProgressWorker<T>::ExecutionProgress::Send(const T* data, size_t count) const {
  _worker->SendProgress_(data, count);
}

#endif

////////////////////////////////////////////////////////////////////////////////
// Memory Management class
////////////////////////////////////////////////////////////////////////////////

inline int64_t MemoryManagement::AdjustExternalMemory(Env env, int64_t change_in_bytes) {
  int64_t result;
  napi_status status = napi_adjust_external_memory(env, change_in_bytes, &result);
  NAPI_THROW_IF_FAILED(env, status, 0);
  return result;
}

////////////////////////////////////////////////////////////////////////////////
// Version Management class
////////////////////////////////////////////////////////////////////////////////

inline uint32_t VersionManagement::GetNapiVersion(Env env) {
  uint32_t result;
  napi_status status = napi_get_version(env, &result);
  NAPI_THROW_IF_FAILED(env, status, 0);
  return result;
}

inline const napi_node_version* VersionManagement::GetNodeVersion(Env env) {
  const napi_node_version* result;
  napi_status status = napi_get_node_version(env, &result);
  NAPI_THROW_IF_FAILED(env, status, 0);
  return result;
}

} // namespace Napi

#endif // SRC_NAPI_INL_H_
