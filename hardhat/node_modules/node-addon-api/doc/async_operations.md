# Asynchronous operations

Node.js native add-ons often need to execute long running tasks and to avoid
blocking the **event loop** they have to run them asynchronously from the
**event loop**.
In the Node.js model of execution the event loop thread represents the thread
where JavaScript code is executing. The Node.js guidance is to avoid blocking
other work queued on the event loop thread. Therefore, we need to do this work on
another thread.

All this means that native add-ons need to leverage async helpers from libuv as
part of their implementation. This allows them to schedule work to be executed
asynchronously so that their methods can return in advance of the work being
completed.

Node Addon API provides an interface to support functions that cover
the most common asynchronous use cases. There is an abstract classes to implement
asynchronous operations:

- **[`Napi::AsyncWorker`](async_worker.md)**

These class helps manage asynchronous operations through an abstraction
of the concept of moving data between the **event loop** and **worker threads**.

Also, the above class may not be appropriate for every scenario. When using any
other asynchronous mechanism, the following API is necessary to ensure an
asynchronous operation is properly tracked by the runtime:

- **[AsyncContext](async_context.md)**

- **[CallbackScope](callback_scope.md)**
