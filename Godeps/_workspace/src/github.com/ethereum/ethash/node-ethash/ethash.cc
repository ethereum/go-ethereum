#include <nan.h>
#include <iostream>
#include <node.h>
#include <stdint.h>
#include <stdlib.h>
#include "../libethash/ethash.h"

using namespace v8;

class EthashValidator : public NanAsyncWorker {
 public:
  // Constructor
  EthashValidator(NanCallback *callback, const unsigned blocknumber, const unsigned char * seed)
    : NanAsyncWorker(callback), blocknumber(blocknumber), seed(seed) {}
  // Destructor
  ~EthashValidator() {
  	free(this->cache);
	free(this->params);
  }

  // Executed inside the worker-thread.
  // It is not safe to access V8, or V8 data structures
  // here, so everything we need for input and output
  // should go on `this`.
  void Execute () {
	
    /* this->result = secp256k1_ecdsa_sign(this->msg, this->sig , &this->sig_len, this->pk, NULL, NULL); */
  }

  // Executed when the async work is complete
  // this function will be run inside the main event loop
  // so it is safe to use V8 again
  void HandleOKCallback () {
    NanScope();
    Handle<Value> argv[] = {
      NanNew<Number>(this->result)
    };
    callback->Call(2, argv);
  }

 protected:
  const unsigned blocknumber; 
  const unsigned char * seed;
  ethash_params * params;
  ethash_cache * cache;
  bool result;
  bool ready = 0;
};

/* class CompactSignWorker : public SignWorker { */
/*  public: */
/*   CompactSignWorker(NanCallback *callback, const unsigned char *msg, const unsigned char *pk ) */
/*     : SignWorker(callback, msg, pk){} */

/*   void Execute () { */
/*     this->result = secp256k1_ecdsa_sign_compact(this->msg, this->sig , this->pk, NULL, NULL,  &this->sig_len); */
/*   } */

/*   void HandleOKCallback () { */
/*     NanScope(); */
/*     Handle<Value> argv[] = { */
/*       NanNew<Number>(this->result), */
/*       NanNewBufferHandle((char *)this->sig, 64), */
/*       NanNew<Number>(this->sig_len) */
/*     }; */
/*     callback->Call(3, argv); */
/*   } */
/* }; */

/* class RecoverWorker : public NanAsyncWorker { */
/*  public: */
/*   // Constructor */
/*   RecoverWorker(NanCallback *callback, const unsigned char *msg, const unsigned char *sig, int compressed, int rec_id) */
/*     : NanAsyncWorker(callback), msg(msg), sig(sig), compressed(compressed), rec_id(rec_id) {} */
/*   // Destructor */
/*   ~RecoverWorker() {} */

/*   void Execute () { */
/*     if(this->compressed == 1){ */
/*       this->pubkey = new unsigned char[33]; */ 
/*     }else{ */
/*       this->pubkey = new unsigned char[65]; */ 
/*     } */

/*     this->result = secp256k1_ecdsa_recover_compact(this->msg, this->sig, this->pubkey, &this->pubkey_len, this->compressed, this->rec_id); */
/*   } */

/*   void HandleOKCallback () { */
/*     NanScope(); */
/*     Handle<Value> argv[] = { */
/*       NanNew<Number>(this->result), */
/*       NanNewBufferHandle((char *)this->pubkey, this->pubkey_len) */
/*     }; */
/*     callback->Call(2, argv); */
/*   } */

/*  protected: */
/*   const unsigned char * msg; */
/*   const unsigned char * sig; */ 
/*   int compressed; */
/*   int rec_id; */
/*   int result; */
/*   unsigned char * pubkey; */
/*   int pubkey_len; */
/* }; */

/* class VerifyWorker : public NanAsyncWorker { */
/*  public: */
/*   // Constructor */
/*   VerifyWorker(NanCallback *callback, const unsigned char *msg, const unsigned char *sig, int sig_len, const unsigned char *pub_key, int pub_key_len) */
/*     : NanAsyncWorker(callback), msg(msg), sig(sig), sig_len(sig_len), pub_key(pub_key), pub_key_len(pub_key_len) {} */
/*   // Destructor */
/*   ~VerifyWorker() {} */

/*   void Execute () { */
/*     this->result = secp256k1_ecdsa_verify(this->msg, this->sig, this->sig_len,  this->pub_key, this->pub_key_len); */
/*   } */

/*   void HandleOKCallback () { */
/*     NanScope(); */
/*     Handle<Value> argv[] = { */
/*       NanNew<Number>(this->result), */
/*     }; */
/*     callback->Call(1, argv); */
/*   } */

/*  protected: */
/*   int result; */
/*   const unsigned char * msg; */
/*   const unsigned char * sig; */
/*   int sig_len; */ 
/*   const unsigned char * pub_key; */
/*   int pub_key_len; */
/* }; */

/* NAN_METHOD(Verify){ */
/*   NanScope(); */

/*   Local<Object> pub_buf = args[0].As<Object>(); */
/*   const unsigned char *pub_data = (unsigned char *) node::Buffer::Data(pub_buf); */
/*   int pub_len = node::Buffer::Length(args[0]); */

/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */

/*   Local<Object> sig_buf = args[2].As<Object>(); */
/*   const unsigned char *sig_data = (unsigned char *) node::Buffer::Data(sig_buf); */
/*   int sig_len = node::Buffer::Length(args[2]); */

/*   int result = secp256k1_ecdsa_verify(msg_data, sig_data, sig_len, pub_data, pub_len ); */ 

/*   NanReturnValue(NanNew<Number>(result)); */
/* } */

/* NAN_METHOD(Verify_Async){ */
/*   NanScope(); */

/*   Local<Object> pub_buf = args[0].As<Object>(); */
/*   const unsigned char *pub_data = (unsigned char *) node::Buffer::Data(pub_buf); */
/*   int pub_len = node::Buffer::Length(args[0]); */

/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */

/*   Local<Object> sig_buf = args[2].As<Object>(); */
/*   const unsigned char *sig_data = (unsigned char *) node::Buffer::Data(sig_buf); */
/*   int sig_len = node::Buffer::Length(args[2]); */

/*   Local<Function> callback = args[3].As<Function>(); */
/*   NanCallback* nanCallback = new NanCallback(callback); */

/*   VerifyWorker* worker = new VerifyWorker(nanCallback, msg_data, sig_data, sig_len, pub_data, pub_len); */
/*   NanAsyncQueueWorker(worker); */

/*   NanReturnUndefined(); */
/* } */

/* NAN_METHOD(Sign){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Local<Object> pk_buf = args[0].As<Object>(); */
/*   const unsigned char *pk_data = (unsigned char *) node::Buffer::Data(pk_buf); */
/*   int sec_len = node::Buffer::Length(args[0]); */
/*   //the second argument is the message that we are signing */
/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */

/*   unsigned char sig[72]; */
/*   int sig_len = 72; */
/*   int msg_len = node::Buffer::Length(args[1]); */

/*   if(sec_len != 32){ */
/*     return NanThrowError("the secret key needs tobe 32 bytes"); */
/*   } */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   int result = secp256k1_ecdsa_sign(msg_data, sig , &sig_len, pk_data, NULL, NULL); */

/*   if(result == 1){ */
/*     NanReturnValue(NanNewBufferHandle((char *)sig, sig_len)); */
/*   }else{ */
/*     return NanThrowError("nonce invalid, try another one"); */
/*   } */
/* } */

/* NAN_METHOD(Sign_Async){ */

/*   NanScope(); */
/*   //the first argument should be the private key as a buffer */
/*   Local<Object> sec_buf = args[0].As<Object>(); */
/*   const unsigned char *sec_data = (unsigned char *) node::Buffer::Data(sec_buf); */
/*   int sec_len = node::Buffer::Length(args[0]); */
/*   //the second argument is the message that we are signing */
/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */

/*   Local<Function> callback = args[2].As<Function>(); */
/*   NanCallback* nanCallback = new NanCallback(callback); */

/*   int msg_len = node::Buffer::Length(args[1]); */

/*   if(sec_len != 32){ */
/*     return NanThrowError("the secret key needs tobe 32 bytes"); */
/*   } */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   SignWorker* worker = new SignWorker(nanCallback, msg_data, sec_data); */
/*   NanAsyncQueueWorker(worker); */

/*   NanReturnUndefined(); */
/* } */

/* NAN_METHOD(Sign_Compact){ */

/*   NanScope(); */

/*   Local<Object> seckey_buf = args[0].As<Object>(); */
/*   const unsigned char *seckey_data = (unsigned char *) node::Buffer::Data(seckey_buf); */
/*   int sec_len = node::Buffer::Length(args[0]); */

/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */
/*   int msg_len = node::Buffer::Length(args[1]); */

/*   if(sec_len != 32){ */
/*     return NanThrowError("the secret key needs tobe 32 bytes"); */
/*   } */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   unsigned char sig[64]; */
/*   int rec_id; */

/*   //TODO: change the nonce */
/*   int valid_nonce = secp256k1_ecdsa_sign_compact(msg_data, sig, seckey_data, NULL, NULL, &rec_id ); */

/*   Local<Array> array = NanNew<Array>(3); */
/*   array->Set(0, NanNew<Integer>(valid_nonce)); */
/*   array->Set(1, NanNew<Integer>(rec_id)); */
/*   array->Set(2, NanNewBufferHandle((char *)sig, 64)); */

/*   NanReturnValue(array); */
/* } */

/* NAN_METHOD(Sign_Compact_Async){ */
/*   NanScope(); */
/*   //the first argument should be the private key as a buffer */
/*   Local<Object> sec_buf = args[0].As<Object>(); */
/*   const unsigned char *sec_data = (unsigned char *) node::Buffer::Data(sec_buf); */
/*   int sec_len = node::Buffer::Length(args[0]); */

/*   //the second argument is the message that we are signing */
/*   Local<Object> msg_buf = args[1].As<Object>(); */
/*   const unsigned char *msg_data = (unsigned char *) node::Buffer::Data(msg_buf); */


/*   Local<Function> callback = args[2].As<Function>(); */
/*   NanCallback* nanCallback = new NanCallback(callback); */

/*   int msg_len = node::Buffer::Length(args[1]); */

/*   if(sec_len != 32){ */
/*     return NanThrowError("the secret key needs tobe 32 bytes"); */
/*   } */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   CompactSignWorker* worker = new CompactSignWorker(nanCallback, msg_data, sec_data); */ 
/*   NanAsyncQueueWorker(worker); */

/*   NanReturnUndefined(); */
/* } */

/* NAN_METHOD(Recover_Compact){ */

/*   NanScope(); */
  
/*   Local<Object> msg_buf = args[0].As<Object>(); */
/*   const unsigned char *msg = (unsigned char *) node::Buffer::Data(msg_buf); */
/*   int msg_len = node::Buffer::Length(args[0]); */

/*   Local<Object> sig_buf = args[1].As<Object>(); */
/*   const unsigned char *sig = (unsigned char *) node::Buffer::Data(sig_buf); */

/*   Local<Number> compressed = args[2].As<Number>(); */
/*   int int_compressed = compressed->IntegerValue(); */

/*   Local<Number> rec_id = args[3].As<Number>(); */
/*   int int_rec_id = rec_id->IntegerValue(); */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   unsigned char pubKey[65]; */ 

/*   int pubKeyLen; */

/*   int result = secp256k1_ecdsa_recover_compact(msg, sig, pubKey, &pubKeyLen, int_compressed, int_rec_id); */
/*   if(result == 1){ */
/*     NanReturnValue(NanNewBufferHandle((char *)pubKey, pubKeyLen)); */
/*   }else{ */
    
/*     NanReturnValue(NanFalse()); */
/*   } */
/* } */

/* NAN_METHOD(Recover_Compact_Async){ */

/*   NanScope(); */
  
/*   //the message */
/*   Local<Object> msg_buf = args[0].As<Object>(); */
/*   const unsigned char *msg = (unsigned char *) node::Buffer::Data(msg_buf); */
/*   int msg_len = node::Buffer::Length(args[0]); */

/*   //the signature length */
/*   Local<Object> sig_buf = args[1].As<Object>(); */
/*   const unsigned char *sig = (unsigned char *) node::Buffer::Data(sig_buf); */
/*   //todo sig len needs tobe 64 */
/*   int sig_len = node::Buffer::Length(args[1]); */

/*   //to compress or not? */
/*   Local<Number> compressed = args[2].As<Number>(); */
/*   int int_compressed = compressed->IntegerValue(); */

/*   //the rec_id */
/*   Local<Number> rec_id = args[3].As<Number>(); */
/*   int int_rec_id = rec_id->IntegerValue(); */

/*   //the callback */
/*   Local<Function> callback = args[4].As<Function>(); */
/*   NanCallback* nanCallback = new NanCallback(callback); */

/*   if(sig_len != 64){ */
/*     return NanThrowError("the signature needs to be 64 bytes"); */
/*   } */

/*   if(msg_len == 0){ */
/*     return NanThrowError("messgae cannot be null"); */ 
/*   } */

/*   RecoverWorker* worker = new RecoverWorker(nanCallback, msg, sig, int_compressed, int_rec_id); */
/*   NanAsyncQueueWorker(worker); */

/*   NanReturnUndefined(); */
/* } */

/* NAN_METHOD(Seckey_Verify){ */
/*   NanScope(); */

/*   const unsigned char *data = (const unsigned char*) node::Buffer::Data(args[0]); */
/*   int result =  secp256k1_ec_seckey_verify(data); */ 
/*   NanReturnValue(NanNew<Number>(result)); */ 
/* } */

/* NAN_METHOD(Pubkey_Verify){ */

/*   NanScope(); */
  
/*   Local<Object> pub_buf = args[0].As<Object>(); */
/*   const unsigned char *pub_key = (unsigned char *) node::Buffer::Data(pub_buf); */
/*   int pub_key_len = node::Buffer::Length(args[0]); */

/*   int result = secp256k1_ec_pubkey_verify(pub_key, pub_key_len); */

/*   NanReturnValue(NanNew<Number>(result)); */ 
/* } */

/* NAN_METHOD(Pubkey_Create){ */
/*   NanScope(); */

/*   Handle<Object> pk_buf = args[0].As<Object>(); */
/*   const unsigned char *pk_data = (unsigned char *) node::Buffer::Data(pk_buf); */
/*   int pk_len = node::Buffer::Length(args[0]); */

/*   Local<Number> l_compact = args[1].As<Number>(); */
/*   int compact = l_compact->IntegerValue(); */
/*   int pubKeyLen; */

/*   if(pk_len != 32){ */
/*     return NanThrowError("the secert key need to be 32 bytes"); */
/*   } */

/*   unsigned char *pubKey; */
/*   if(compact == 1){ */
/*     pubKey = new unsigned char[33]; */ 
/*   }else{ */
/*     pubKey = new unsigned char[65]; */ 
/*   } */

/*   int results = secp256k1_ec_pubkey_create(pubKey,&pubKeyLen, pk_data, compact ); */
/*   if(results == 0){ */
/*     return NanThrowError("secret was invalid, try again."); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)pubKey, pubKeyLen)); */
/*   } */
/* } */

/* NAN_METHOD(Pubkey_Decompress){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Local<Object> pk_buf = args[0].As<Object>(); */
/*   unsigned char *pk_data = (unsigned char *) node::Buffer::Data(pk_buf); */

/*   int pk_len = node::Buffer::Length(args[0]); */

/*   int results = secp256k1_ec_pubkey_decompress(pk_data, &pk_len); */

/*   if(results == 0){ */
/*     return NanThrowError("invalid public key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)pk_data, pk_len)); */
/*   } */
/* } */


/* NAN_METHOD(Privkey_Import){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> pk_buf = args[0].As<Object>(); */
/*   const unsigned char *pk_data = (unsigned char *) node::Buffer::Data(pk_buf); */

/*   int pk_len = node::Buffer::Length(args[0]); */

/*   unsigned char sec_key[32]; */
/*   int results = secp256k1_ec_privkey_import(sec_key, pk_data, pk_len); */

/*   if(results == 0){ */
/*     return NanThrowError("invalid private key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)sec_key, 32)); */
/*   } */
/* } */

/* NAN_METHOD(Privkey_Export){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> sk_buf = args[0].As<Object>(); */
/*   const unsigned char *sk_data = (unsigned char *) node::Buffer::Data(sk_buf); */

/*   Local<Number> l_compressed = args[1].As<Number>(); */
/*   int compressed = l_compressed->IntegerValue(); */

/*   unsigned char *privKey; */
/*   int pk_len; */
/*   int results = secp256k1_ec_privkey_export(sk_data, privKey, &pk_len, compressed); */
/*   if(results == 0){ */
/*     return NanThrowError("invalid private key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)privKey, pk_len)); */
/*   } */
/* } */

/* NAN_METHOD(Privkey_Tweak_Add){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> sk_buf = args[0].As<Object>(); */
/*   unsigned char *sk = (unsigned char *) node::Buffer::Data(sk_buf); */

/*   Handle<Object> tweak_buf = args[1].As<Object>(); */
/*   const unsigned char *tweak= (unsigned char *) node::Buffer::Data(tweak_buf); */

/*   int results = secp256k1_ec_privkey_tweak_add(sk, tweak); */
/*   if(results == 0){ */
/*     return NanThrowError("invalid key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)sk, 32)); */
/*   } */
/* } */

/* NAN_METHOD(Privkey_Tweak_Mul){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> sk_buf = args[0].As<Object>(); */
/*   unsigned char *sk = (unsigned char *) node::Buffer::Data(sk_buf); */

/*   Handle<Object> tweak_buf = args[1].As<Object>(); */
/*   const unsigned char *tweak= (unsigned char *) node::Buffer::Data(tweak_buf); */

/*   int results = secp256k1_ec_privkey_tweak_mul(sk, tweak); */
/*   if(results == 0){ */
/*     return NanThrowError("invalid key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)sk, 32)); */
/*   } */
/* } */

/* NAN_METHOD(Pubkey_Tweak_Add){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> pk_buf = args[0].As<Object>(); */
/*   unsigned char *pk = (unsigned char *) node::Buffer::Data(pk_buf); */
/*   int pk_len = node::Buffer::Length(args[0]); */

/*   Handle<Object> tweak_buf = args[1].As<Object>(); */
/*   const unsigned char *tweak= (unsigned char *) node::Buffer::Data(tweak_buf); */

/*   int results = secp256k1_ec_pubkey_tweak_add(pk, pk_len, tweak); */
/*   if(results == 0){ */
/*     return NanThrowError("invalid key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)pk, pk_len)); */
/*   } */
/* } */

/* NAN_METHOD(Pubkey_Tweak_Mul){ */
/*   NanScope(); */

/*   //the first argument should be the private key as a buffer */
/*   Handle<Object> pk_buf = args[0].As<Object>(); */
/*   unsigned char *pk = (unsigned char *) node::Buffer::Data(pk_buf); */
/*   int pk_len = node::Buffer::Length(args[0]); */

/*   Handle<Object> tweak_buf = args[1].As<Object>(); */
/*   const unsigned char *tweak= (unsigned char *) node::Buffer::Data(tweak_buf); */

/*   int results = secp256k1_ec_pubkey_tweak_mul(pk, pk_len, tweak); */
/*   if(results == 0){ */
/*     return NanThrowError("invalid key"); */
/*   }else{ */
/*     NanReturnValue(NanNewBufferHandle((char *)pk, pk_len)); */
/*   } */
/* } */

void Init(Handle<Object> exports) {

  /* secp256k1_start(SECP256K1_START_SIGN | SECP256K1_START_VERIFY); */
  /* exports->Set(NanNew("seckeyVerify"), NanNew<FunctionTemplate>(Seckey_Verify)->GetFunction()); */
  /* exports->Set(NanNew("sign"), NanNew<FunctionTemplate>(Sign)->GetFunction()); */
  /* exports->Set(NanNew("signAsync"), NanNew<FunctionTemplate>(Sign_Async)->GetFunction()); */
  /* exports->Set(NanNew("signCompact"), NanNew<FunctionTemplate>(Sign_Compact)->GetFunction()); */
  /* exports->Set(NanNew("signCompactAsync"), NanNew<FunctionTemplate>(Sign_Compact_Async)->GetFunction()); */
  /* exports->Set(NanNew("recoverCompact"), NanNew<FunctionTemplate>(Recover_Compact)->GetFunction()); */
  /* exports->Set(NanNew("recoverCompactAsync"), NanNew<FunctionTemplate>(Recover_Compact_Async)->GetFunction()); */
  /* exports->Set(NanNew("verify"), NanNew<FunctionTemplate>(Verify)->GetFunction()); */
  /* exports->Set(NanNew("verifyAsync"), NanNew<FunctionTemplate>(Verify_Async)->GetFunction()); */
  /* exports->Set(NanNew("secKeyVerify"), NanNew<FunctionTemplate>(Seckey_Verify)->GetFunction()); */
  /* exports->Set(NanNew("pubKeyVerify"), NanNew<FunctionTemplate>(Pubkey_Verify)->GetFunction()); */
  /* exports->Set(NanNew("pubKeyCreate"), NanNew<FunctionTemplate>(Pubkey_Create)->GetFunction()); */
  /* exports->Set(NanNew("pubKeyDecompress"), NanNew<FunctionTemplate>(Pubkey_Decompress)->GetFunction()); */
  /* exports->Set(NanNew("privKeyExport"), NanNew<FunctionTemplate>(Privkey_Export)->GetFunction()); */
  /* exports->Set(NanNew("privKeyImport"), NanNew<FunctionTemplate>(Privkey_Import)->GetFunction()); */
  /* exports->Set(NanNew("privKeyTweakAdd"), NanNew<FunctionTemplate>(Privkey_Tweak_Add)->GetFunction()); */
  /* exports->Set(NanNew("privKeyTweakMul"), NanNew<FunctionTemplate>(Privkey_Tweak_Mul)->GetFunction()); */
  /* exports->Set(NanNew("pubKeyTweakAdd"), NanNew<FunctionTemplate>(Privkey_Tweak_Add)->GetFunction()); */
  /* exports->Set(NanNew("pubKeyTweakMul"), NanNew<FunctionTemplate>(Privkey_Tweak_Mul)->GetFunction()); */
}

NODE_MODULE(secp256k1, Init)
