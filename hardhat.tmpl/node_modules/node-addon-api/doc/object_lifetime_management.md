# Object lifetime management

A handle may be created when any new node-addon-api Value and
its subclasses is created or returned.

As the methods and classes within the node-addon-api are used,
handles to objects in the heap for the underlying
VM may be created. A handle may be created when any new
node-addon-api Value or one of its subclasses is created or returned.
These handles must hold the objects 'live' until they are no
longer required by the native code, otherwise the objects could be
collected by the garbage collector before the native code was
finished using them.

As handles are created they are associated with a
'scope'. The lifespan for the default scope is tied to the lifespan
of the native method call. The result is that, by default, handles
remain valid and the objects associated with these handles will be
held live for the lifespan of the native method call.

In many cases, however, it is necessary that the handles remain valid for
either a shorter or longer lifespan than that of the native method.
The sections which follow describe the node-addon-api classes and
methods that than can be used to change the handle lifespan from
the default.

## Making handle lifespan shorter than that of the native method

It is often necessary to make the lifespan of handles shorter than
the lifespan of a native method. For example, consider a native method
that has a loop which creates a number of values and does something
with each of the values, one at a time:

```C++
for (int i = 0; i < LOOP_MAX; i++) {
  std::string name = std::string("inner-scope") + std::to_string(i);
  Napi::Value newValue = Napi::String::New(info.Env(), name.c_str());
  // do something with newValue
};
```

This would result in a large number of handles being created, consuming
substantial resources. In addition, even though the native code could only
use the most recently created value, all of the previously created
values would also be kept alive since they all share the same scope.

To handle this case, node-addon-api provides the ability to establish
a new 'scope' to which newly created handles will be associated. Once those
handles are no longer required, the scope can be deleted and any handles
associated with the scope are invalidated. The `Napi::HandleScope`
and `Napi::EscapableHandleScope` classes are provided by node-addon-api for
creating additional scopes.

node-addon-api only supports a single nested hierarchy of scopes. There is
only one active scope at any time, and all new handles will be associated
with that scope while it is active. Scopes must be deleted in the reverse
order from which they are opened. In addition, all scopes created within
a native method must be deleted before returning from that method. Since
`Napi::HandleScopes` are typically stack allocated the compiler will take care of
deletion, however, care must be taken to create the scope in the right
place such that you achieve the desired lifetime.

Taking the earlier example, creating a `Napi::HandleScope` in the innner loop
would ensure that at most a single new value is held alive throughout the
execution of the loop:

```C
for (int i = 0; i < LOOP_MAX; i++) {
  Napi::HandleScope scope(info.Env());
  std::string name = std::string("inner-scope") + std::to_string(i);
  Napi::Value newValue = Napi::String::New(info.Env(), name.c_str());
  // do something with neValue
};
```

When nesting scopes, there are cases where a handle from an
inner scope needs to live beyond the lifespan of that scope. node-addon-api
provides the `Napi::EscapableHandleScope` with the `Escape` method
in order to support this case. An escapable scope
allows one object to be 'promoted' so that it 'escapes' the
current scope and the lifespan of the handle changes from the current
scope to that of the outer scope. The `Escape` method can only be called
once for a given `Napi::EscapableHandleScope`.
