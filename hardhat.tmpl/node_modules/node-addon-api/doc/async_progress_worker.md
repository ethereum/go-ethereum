# AsyncProgressWorker

`Napi::AsyncProgressWorker` is an abstract class which implements `Napi::AsyncWorker`
while extending `Napi::AsyncWorker` internally with `Napi::ThreadSafeFunction` for
moving work progress reports from worker thread(s) to event loop threads.

Like `Napi::AsyncWorker`, once created, execution is requested by calling
`Napi::AsyncProgressWorker::Queue`. When a thread is available for execution
the `Napi::AsyncProgressWorker::Execute` method will be invoked. During the
execution, `Napi::AsyncProgressWorker::ExecutionProgress::Send` can be used to
indicate execution process, which will eventually invoke `Napi::AsyncProgressWorker::OnProgress`
on the JavaScript thread to safely call into JavaScript. Once `Napi::AsyncProgressWorker::Execute`
completes either `Napi::AsyncProgressWorker::OnOK` or `Napi::AsyncProgressWorker::OnError`
will be invoked. Once the `Napi::AsyncProgressWorker::OnOK` or `Napi::AsyncProgressWorker::OnError`
methods are complete the `Napi::AsyncProgressWorker` instance is destructed.

For the most basic use, only the `Napi::AsyncProgressWorker::Execute` and
`Napi::AsyncProgressWorker::OnProgress` method must be implemented in a subclass.

## Methods

[`Napi::AsyncWorker`][] provides detailed descriptions for most methods.

### Execute

This method is used to execute some tasks outside of the **event loop** on a libuv
worker thread. Subclasses must implement this method and the method is run on
a thread other than that running the main event loop. As the method is not
running on the main event loop, it must avoid calling any methods from node-addon-api
or running any code that might invoke JavaScript. Instead, once this method is
complete any interaction through node-addon-api with JavaScript should be implemented
in the `Napi::AsyncProgressWorker::OnOK` method and/or `Napi::AsyncProgressWorker::OnError`
which run on the main thread and are invoked when the `Napi::AsyncProgressWorker::Execute`
method completes.

```cpp
virtual void Napi::AsyncProgressWorker::Execute(const ExecutionProgress& progress) = 0;
```

### OnOK

This method is invoked when the computation in the `Execute` method ends.
The default implementation runs the `Callback` optionally provided when the
`AsyncProgressWorker` class was created. The `Callback` will by default receive no
arguments. Arguments to the callback can be provided by overriding the `GetResult()`
method.

```cpp
virtual void Napi::AsyncProgressWorker::OnOK();
```

### OnProgress

This method is invoked when the computation in the `Napi::AsyncProgressWorker::ExecutionProcess::Send`
method was called during worker thread execution.

```cpp
virtual void Napi::AsyncProgressWorker::OnProgress(const T* data, size_t count)
```

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Function& callback);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Function& callback, const char* resource_name);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Function& callback, const char* resource_name, const Napi::Object& resource);
```

- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`:  Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Object& receiver, const Napi::Function& callback);
```

- `[in] receiver`: The `this` object passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Object& receiver, const Napi::Function& callback, const char* resource_name);
```

- `[in] receiver`: The `this` object passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`:  Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncWork` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(const Napi::Object& receiver, const Napi::Function& callback, const char* resource_name, const Napi::Object& resource);
```

- `[in] receiver`: The `this` object to be passed to the called function.
- `[in] callback`: The function which will be called when an asynchronous
operations ends. The given function is called from the main event loop thread.
- `[in] resource_name`:  Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncWork` instance which can later be queued for execution by
calling `Napi::AsyncWork::Queue`.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(Napi::Env env);
```

- `[in] env`: The environment in which to create the `Napi::AsyncProgressWorker`.

Returns an `Napi::AsyncProgressWorker` instance which can later be queued for execution by calling
`Napi::AsyncProgressWorker::Queue`.

Available with `NAPI_VERSION` equal to or greater than 5.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(Napi::Env env, const char* resource_name);
```

- `[in] env`: The environment in which to create the `Napi::AsyncProgressWorker`.
- `[in] resource_name`: Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncProgressWorker::Queue`.

Available with `NAPI_VERSION` equal to or greater than 5.

### Constructor

Creates a new `Napi::AsyncProgressWorker`.

```cpp
explicit Napi::AsyncProgressWorker(Napi::Env env, const char* resource_name, const Napi::Object& resource);
```

- `[in] env`: The environment in which to create the `Napi::AsyncProgressWorker`.
- `[in] resource_name`:  Null-terminated string that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the async_hooks API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible async_hooks.

Returns a `Napi::AsyncProgressWorker` instance which can later be queued for execution by
calling `Napi::AsyncProgressWorker::Queue`.

Available with `NAPI_VERSION` equal to or greater than 5.

### Destructor

Deletes the created work object that is used to execute logic asynchronously and
release the internal `Napi::ThreadSafeFunction`, which will be aborted to prevent
unexpected upcoming thread safe calls.

```cpp
virtual Napi::AsyncProgressWorker::~AsyncProgressWorker();
```

# AsyncProgressWorker::ExecutionProcess

A bridge class created before the worker thread execution of `Napi::AsyncProgressWorker::Execute`.

## Methods

### Send

`Napi::AsyncProgressWorker::ExecutionProcess::Send` takes two arguments, a pointer
to a generic type of data, and a `size_t` to indicate how many items the pointer is
pointing to.

The data pointed to will be copied to internal slots of `Napi::AsyncProgressWorker` so
after the call to `Napi::AsyncProgressWorker::ExecutionProcess::Send` the data can
be safely released.

Note that `Napi::AsyncProgressWorker::ExecutionProcess::Send` merely guarantees
**eventual** invocation of `Napi::AsyncProgressWorker::OnProgress`, which means
multiple send might be coalesced into single invocation of `Napi::AsyncProgressWorker::OnProgress`
with latest data.

```cpp
void Napi::AsyncProgressWorker::ExecutionProcess::Send(const T* data, size_t count) const;
```

## Example

The first step to use the `Napi::AsyncProgressWorker` class is to create a new class that
inherits from it and implement the `Napi::AsyncProgressWorker::Execute` abstract method.
Typically input to the worker will be saved within the class' fields generally
passed in through its constructor.

During the worker thread execution, the first argument of `Napi::AsyncProgressWorker::Execute`
can be used to report the progress of the execution.

When the `Napi::AsyncProgressWorker::Execute` method completes without errors the
`Napi::AsyncProgressWorker::OnOK` function callback will be invoked. In this function the
results of the computation will be reassembled and returned back to the initial
JavaScript context.

`Napi::AsyncProgressWorker` ensures that all the code in the `Napi::AsyncProgressWorker::Execute`
function runs in the background out of the **event loop** thread and at the end
the `Napi::AsyncProgressWorker::OnOK` or `Napi::AsyncProgressWorker::OnError` function will be
called and are executed as part of the event loop.

The code below shows a basic example of the `Napi::AsyncProgressWorker` implementation:

```cpp
#include<napi.h>

#include <chrono>
#include <thread>

use namespace Napi;

class EchoWorker : public AsyncProgressWorker<uint32_t> {
    public:
        EchoWorker(Function& callback, std::string& echo)
        : AsyncProgressWorker(callback), echo(echo) {}

        ~EchoWorker() {}
    // This code will be executed on the worker thread
    void Execute(const ExecutionProgress& progress) {
        // Need to simulate cpu heavy task
        for (uint32_t i = 0; i < 100; ++i) {
          progress.Send(&i, 1)
          std::this_thread::sleep_for(std::chrono::seconds(1));
        }
    }

    void OnOK() {
        HandleScope scope(Env());
        Callback().Call({Env().Null(), String::New(Env(), echo)});
    }

    void OnProgress(const uint32_t* data, size_t /* count */) {
        HandleScope scope(Env());
        Callback().Call({Env().Null(), Env().Null(), Number::New(Env(), *data)});
    }

    private:
        std::string echo;
};
```

The `EchoWorker`'s constructor calls the base class' constructor to pass in the
callback that the `Napi::AsyncProgressWorker` base class will store persistently. When
the work on the `Napi::AsyncProgressWorker::Execute` method is done the
`Napi::AsyncProgressWorker::OnOk` method is called and the results are return back to
JavaScript when the stored callback is invoked with its associated environment.

The following code shows an example of how to create and use an `Napi::AsyncProgressWorker`

```cpp
#include <napi.h>

// Include EchoWorker class
// ..

use namespace Napi;

Value Echo(const CallbackInfo& info) {
    // We need to validate the arguments here
    Function cb = info[1].As<Function>();
    std::string in = info[0].As<String>();
    EchoWorker* wk = new EchoWorker(cb, in);
    wk->Queue();
    return info.Env().Undefined();
}
```

The implementation of a `Napi::AsyncProgressWorker` can be used by creating a
new instance and passing to its constructor the callback to execute when the
asynchronous task ends and other data needed for the computation. Once created,
the only other action needed is to call the `Napi::AsyncProgressWorker::Queue`
method that will queue the created worker for execution.

[`Napi::AsyncWorker`]: ./async_worker.md
