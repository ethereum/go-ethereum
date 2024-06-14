# AsyncWorker

`Napi::AsyncWorker` is an abstract class that you can subclass to remove many of
the tedious tasks of moving data between the event loop and worker threads. This
class internally handles all the details of creating and executing an asynchronous
operation.

Once created, execution is requested by calling `Napi::AsyncWorker::Queue`. When
a thread is available for execution the `Napi::AsyncWorker::Execute` method will
be invoked. Once `Napi::AsyncWorker::Execute` completes either
`Napi::AsyncWorker::OnOK` or `Napi::AsyncWorker::OnError` will be invoked. Once
the `Napi::AsyncWorker::OnOK` or `Napi::AsyncWorker::OnError` methods are
complete the `Napi::AsyncWorker` instance is destructed.

For the most basic use, only the `Napi::AsyncWorker::Execute` method must be
implemented in a subclass.

## Methods

### Env

Requests the environment in which the async worker has been initially created.

```cpp
Napi::Env Napi::AsyncWorker::Env() const;
```

Returns the environment in which the async worker has been created.

### Queue

Requests that the work be queued for execution.

```cpp
void Napi::AsyncWorker::Queue();
```

### Cancel

Cancels queued work if it has not yet been started. If it has already started
executing, it cannot be cancelled. If cancelled successfully neither
`OnOK` nor `OnError` will be called.

```cpp
void Napi::AsyncWorker::Cancel();
```

### Receiver

```cpp
Napi::ObjectReference& Napi::AsyncWorker::Receiver();
```

Returns the persistent object reference of the receiver object set when the async
worker was created.

### Callback

```cpp
Napi::FunctionReference& Napi::AsyncWorker::Callback();
```

Returns the persistent function reference of the callback set when the async
worker was created. The returned function reference will receive the results of
the computation that happened in the `Napi::AsyncWorker::Execute` method, unless
the default implementation of `Napi::AsyncWorker::OnOK` or
`Napi::AsyncWorker::OnError` is overridden.

### SuppressDestruct

```cpp
void Napi::AsyncWorker::SuppressDestruct();
```

Prevents the destruction of the `Napi::AsyncWorker` instance upon completion of
the `Napi::AsyncWorker::OnOK` callback.

### SetError

Sets the error message for the error that happened during the execution. Setting
an error message will cause the `Napi::AsyncWorker::OnError` method to be
invoked instead of `Napi::AsyncWorker::OnOK` once the
`Napi::AsyncWorker::Execute` method completes.

```cpp
void Napi::AsyncWorker::SetError(const std::string& error);
```

- `[in] error`: The reference to the string that represent the message of the error.

### Execute

This method is used to execute some tasks outside of the **event loop** on a libuv
worker thread. Subclasses must implement this method and the method is run on
a thread other than that running the main event loop. As the method is not
running on the main event loop, it must avoid calling any methods from node-addon-api
or running any code that might invoke JavaScript. Instead, once this method is
complete any interaction through node-addon-api with JavaScript should be implemented
in the `Napi::AsyncWorker::OnOK` method and `Napi::AsyncWorker::OnError` which run
on the main thread and are invoked when the `Napi::AsyncWorker::Execute` method completes.

```cpp
virtual void Napi::AsyncWorker::Execute() = 0;
```

### OnOK

This method is invoked when the computation in the `Execute` method ends.
The default implementation runs the `Callback` optionally provided when the
`AsyncWorker` class was created. The `Callback` will by default receive no
arguments. The arguments to the `Callback` can be provided by overriding the
`GetResult()` method.

```cpp
virtual void Napi::AsyncWorker::OnOK();
```
### GetResult

This method returns the arguments passed to the `Callback` invoked by the default
`OnOK()` implementation. The default implementation returns an empty vector,
providing no arguments to the `Callback`.

```cpp
virtual std::vector<napi_value> Napi::AsyncWorker::GetResult(Napi::Env env);
```

### OnError

This method is invoked after `Napi::AsyncWorker::Execute` completes if an error
occurs while `Napi::AsyncWorker::Execute` is running and C++ exceptions are
enabled or if an error was set through a call to `Napi::AsyncWorker::SetError`.
The default implementation calls the `Callback` provided when the `Napi::AsyncWorker`
class was created, passing in the error as the first parameter.

```cpp
virtual void Napi::AsyncWorker::OnError(const Napi::Error& e);
```

### Destroy

This method is invoked when the instance must be deallocated. If
`SuppressDestruct()` was not called then this method will be called after either
`OnError()` or `OnOK()` complete. The default implementation of this method
causes the instance to delete itself using the `delete` operator. The method is
provided so as to ensure that instances allocated by means other than the `new`
operator can be deallocated upon work completion.

```cpp
virtual void Napi::AsyncWorker::Destroy();
```

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Function& callback);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by calling
`Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Function& callback, const char* resource_name);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Function& callback, const char* resource_name, const Napi::Object& resource);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Object& receiver, const Napi::Function& callback);
```

- `[in] receiver`: The `this` object passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Object& receiver, const Napi::Function& callback, const char* resource_name);
```

- `[in] receiver`: The `this` object passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncWork` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(const Napi::Object& receiver, const Napi::Function& callback, const char* resource_name, const Napi::Object& resource);
```

- `[in] receiver`: The `this` object passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncWork` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.


### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(Napi::Env env);
```

- `[in] env`: The environment in which to create the `Napi::AsyncWorker`.

Returns an `Napi::AsyncWorker` instance which can later be queued for execution by calling
`Napi::AsyncWorker::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(Napi::Env env, const char* resource_name);
```

- `[in] env`: The environment in which to create the `Napi::AsyncWorker`.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by
calling `Napi::AsyncWorker::Queue`.

### Constructor

Creates a new `Napi::AsyncWorker`.

```cpp
explicit Napi::AsyncWorker(Napi::Env env, const char* resource_name, const Napi::Object& resource);
```

- `[in] env`: The environment in which to create the `Napi::AsyncWorker`.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncWorker` instance which can later be queued for execution by
calling `Napi::AsyncWorker::Queue`.

### Destructor

Deletes the created work object that is used to execute logic asynchronously.

```cpp
virtual Napi::AsyncWorker::~AsyncWorker();
```

## Operator

```cpp
Napi::AsyncWorker::operator napi_async_work() const;
```

Returns the N-API napi_async_work wrapped by the `Napi::AsyncWorker` object. This
can be used to mix usage of the C N-API and node-addon-api.

## Example

The first step to use the `Napi::AsyncWorker` class is to create a new class that
inherits from it and implement the `Napi::AsyncWorker::Execute` abstract method.
Typically input to your worker will be saved within class' fields generally
passed in through its constructor.

When the `Napi::AsyncWorker::Execute` method completes without errors the
`Napi::AsyncWorker::OnOK` function callback will be invoked. In this function the
results of the computation will be reassembled and returned back to the initial
JavaScript context.

`Napi::AsyncWorker` ensures that all the code in the `Napi::AsyncWorker::Execute`
function runs in the background out of the **event loop** thread and at the end
the `Napi::AsyncWorker::OnOK` or `Napi::AsyncWorker::OnError` function will be
called and are executed as part of the event loop.

The code below shows a basic example of `Napi::AsyncWorker` the implementation:

```cpp
#include<napi.h>

#include <chrono>
#include <thread>

use namespace Napi;

class EchoWorker : public AsyncWorker {
    public:
        EchoWorker(Function& callback, std::string& echo)
        : AsyncWorker(callback), echo(echo) {}

        ~EchoWorker() {}
    // This code will be executed on the worker thread
    void Execute() {
        // Need to simulate cpu heavy task
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }

    void OnOK() {
        HandleScope scope(Env());
        Callback().Call({Env().Null(), String::New(Env(), echo)});
    }

    private:
        std::string echo;
};
```

The `EchoWorker`'s contructor calls the base class' constructor to pass in the
callback that the `Napi::AsyncWorker` base class will store persistently. When
the work on the `Napi::AsyncWorker::Execute` method is done the
`Napi::AsyncWorker::OnOk` method is called and the results return back to
JavaScript invoking the stored callback with its associated environment.

The following code shows an example of how to create and use an `Napi::AsyncWorker`.

```cpp
#include<napi.h>

// Include EchoWorker class
// ..

use namespace Napi;

Value Echo(const CallbackInfo& info) {
    // You need to validate the arguments here.
    Function cb = info[1].As<Function>();
    std::string in = info[0].As<String>();
    EchoWorker* wk = new EchoWorker(cb, in);
    wk->Queue();
    return info.Env().Undefined();
```

Using the implementation of a `Napi::AsyncWorker` is straight forward. You only
need to create a new instance and pass to its constructor the callback you want to
execute when your asynchronous task ends and other data you need for your
computation. Once created the only other action you have to do is to call the
`Napi::AsyncWorker::Queue` method that will queue the created worker for execution.
