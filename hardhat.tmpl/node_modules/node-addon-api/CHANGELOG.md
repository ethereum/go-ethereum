# node-addon-api Changelog

## 2020-07-01 Version 2.0.2, @NickNaso

### Notable changes:

#### API

- `Napi::ObjectWrap`: avoid double-free on old Node.js.
- `Napi::ObjectWrap`: remove wrap only on failure.
- `Napi::ObjectWrap`: gracefully handle constructor exceptions
- `Napi::ObjectWrap`: call `napi_remove_wrap()` in destructor.

#### TEST

- Updated `Napi::BigInt` test for recent change in core.

### Commmits

* [[`5abf60257d`](https://github.com/nodejs/node-addon-api/commit/5abf60257d)] - Merge pull request #723 from gabrielschulhof/backport-4e885069-pr-475 (Nicola Del Gobbo)
* [[`470f130666`](https://github.com/nodejs/node-addon-api/commit/470f130666)] - **objectwrap**: avoid double-free on old Node.js (Gabriel Schulhof)
* [[`81e2eac7ba`](https://github.com/nodejs/node-addon-api/commit/81e2eac7ba)] - **test**: update BigInt test for recent change in core (Michael Dawson) [#649](https://github.com/nodejs/node-addon-api/pull/649)
* [[`204f07252c`](https://github.com/nodejs/node-addon-api/commit/204f07252c)] - **objectwrap**: remove wrap only on failure (Gabriel Schulhof)
* [[`a552a384dd`](https://github.com/nodejs/node-addon-api/commit/a552a384dd)] - **src**: call `napi\_remove\_wrap()` in `ObjectWrap` dtor (Anna Henningsen) [#475](https://github.com/nodejs/node-addon-api/pull/475)
* [[`1a51067438`](https://github.com/nodejs/node-addon-api/commit/1a51067438)] - **objectwrap**: gracefully handle constructor exceptions (Gabriel Schulhof)

## 2020-06-02 Version 2.0.1, @NickNaso

### Notable changes:

#### API

- Fix memory corruption vulnerability

### Commmits

* [[`265fea9edd`](https://github.com/nodejs/node-addon-api/commit/265fea9edd)] - **napi**: fix memory corruption vulnerability (Tobias Nießen )

## 2019-11-21 Version 2.0.0, @NickNaso

### Notable changes:

#### API

- Added `Napi::AsyncProgressWorker` api.
- Added error checking on `Napi::ThreadSafeFunction::GetContext`.
- Added copy constructor to `Napi::ThreadSafeFunction`.
- Added `Napi::ThreadSafeFunction::Ref` and `Napi::ThreadSafeFunction::Unref` to `Napi::ThreadSafeFunction`.
- Added `Napi::Object::AddFinalizer` method.
- Use `napi_add_finalizer()` to attach data when building against N-API 5.
- Added `Napi::Date` api.
- Added `Napi::ObjectWrap::Finalize` method.

#### Documentation

- Added documentation for `Napi::AsyncProgressWorker`.
- Improve `Napi::AsyncWorker` documentation.
- Added documentation for `Napi::Object::AddFinalizer` method.
- Improved documentation for `Napi::ThreadSafeFunction`.
- Improved documentation about the usage of CMake as build tool.
- Some minor corrections all over the documentation.

#### TEST

- Added test cases for `Napi::AsyncProgressWorker` api.
- Added test cases for `Napi::Date` api.
- Added test cases for new features added to `Napi::ThreadSafeFunction`.

### Commmits

* [[`c881168d49`](https://github.com/nodejs/node-addon-api/commit/c881168d49)] - **tsfn**: add error checking on GetContext (#583) (Kevin Eady) [#583](https://github.com/nodejs/node-addon-api/pull/583)
* [[`24d75dd82f`](https://github.com/nodejs/node-addon-api/commit/24d75dd82f)] - Merge pull request #588 from NickNaso/add-asyncprogress-worker-readme (Nicola Del Gobbo)
* [[`aa79e37b62`](https://github.com/nodejs/node-addon-api/commit/aa79e37b62)] - Merge pull request #587 from timrach/patch-1 (Nicola Del Gobbo)
* [[`df75e08c2b`](https://github.com/nodejs/node-addon-api/commit/df75e08c2b)] - **tsfn**: support direct calls to underlying napi\_tsfn (Kevin Eady) [#58](https://github.com/nodejs/node-addon-api/pull/58)
* [[`2298dfae58`](https://github.com/nodejs/node-addon-api/commit/2298dfae58)] - **doc**: Added AsyncProgressWorker to readme (NickNaso)
* [[`b3609d33b6`](https://github.com/nodejs/node-addon-api/commit/b3609d33b6)] - Fix return type and declaration of setter callback (Tim Rach)
* [[`295e560f55`](https://github.com/nodejs/node-addon-api/commit/295e560f55)] - **test**: improve guards for experimental features (legendecas)
* [[`2e71842f63`](https://github.com/nodejs/node-addon-api/commit/2e71842f63)] - **tsfn**: Implement copy constructor (Kevin Eady) [#546](https://github.com/nodejs/node-addon-api/pull/546)
* [[`650562cab9`](https://github.com/nodejs/node-addon-api/commit/650562cab9)] - **src**: implement AsyncProgressWorker (legendecas) [#529](https://github.com/nodejs/node-addon-api/pull/529)
* [[`bdfd14101f`](https://github.com/nodejs/node-addon-api/commit/bdfd14101f)] - **src**: attach data with napi\_add\_finalizer (Gabriel Schulhof) [#577](https://github.com/nodejs/node-addon-api/pull/577)
* [[`9e955a802b`](https://github.com/nodejs/node-addon-api/commit/9e955a802b)] - **doc**: change node.js to Node.js per guideline (#579) (Tobias Nießen) [#579](https://github.com/nodejs/node-addon-api/pull/579)
* [[`b42e21e3a9`](https://github.com/nodejs/node-addon-api/commit/b42e21e3a9)] - **build**: move node/6 to travis allowed failures and add node/13 (#573) (Gabriel Schulhof)
* [[`8d6132f609`](https://github.com/nodejs/node-addon-api/commit/8d6132f609)] - **doc**: improve AsyncWorker docs (#571) (legendecas) [#571](https://github.com/nodejs/node-addon-api/pull/571)
* [[`bc8fc23627`](https://github.com/nodejs/node-addon-api/commit/bc8fc23627)] - **test**: do not run TSFN tests on NAPI\_VERSION \< 4 (legendecas) [#576](https://github.com/nodejs/node-addon-api/pull/576)
* [[`bcc1d58fc4`](https://github.com/nodejs/node-addon-api/commit/bcc1d58fc4)] - implement Object::AddFinalizer (Gabriel Schulhof)
* [[`e9a4bcd52a`](https://github.com/nodejs/node-addon-api/commit/e9a4bcd52a)] - **doc**: updates Make.js doc to current best practices (Jim Schlight) [#558](https://github.com/nodejs/node-addon-api/pull/558)
* [[`b513d1aa7a`](https://github.com/nodejs/node-addon-api/commit/b513d1aa7a)] - **doc**: fix return type of ArrayBuffer::Data (Tobias Nießen) [#552](https://github.com/nodejs/node-addon-api/pull/552)
* [[`34c11cf0a4`](https://github.com/nodejs/node-addon-api/commit/34c11cf0a4)] - **src**: disallow copying, double close of scopes (legendecas) [#566](https://github.com/nodejs/node-addon-api/pull/566)
* [[`ce139a05e8`](https://github.com/nodejs/node-addon-api/commit/ce139a05e8)] - **src**: make failure of closing scopes fatal (legendecas) [#566](https://github.com/nodejs/node-addon-api/pull/566)
* [[`740c79823e`](https://github.com/nodejs/node-addon-api/commit/740c79823e)] - **src**: add Env() to AsyncContext (Rolf Timmermans) [#568](https://github.com/nodejs/node-addon-api/pull/568)
* [[`ea9ce1c801`](https://github.com/nodejs/node-addon-api/commit/ea9ce1c801)] - **tsfn**: add wrappers for Ref and Unref (Kevin Eady) [#561](https://github.com/nodejs/node-addon-api/pull/561)
* [[`2e1769e1a3`](https://github.com/nodejs/node-addon-api/commit/2e1769e1a3)] - **error**: remove unnecessary if condition (legendecas) [#562](https://github.com/nodejs/node-addon-api/pull/562)
* [[`828f223a87`](https://github.com/nodejs/node-addon-api/commit/828f223a87)] - **doc**: fix spelling in ObjectWrap doc (#563) (Tobias Nießen) [#563](https://github.com/nodejs/node-addon-api/pull/563)
* [[`dd9fa8a4a8`](https://github.com/nodejs/node-addon-api/commit/dd9fa8a4a8)] - **doc**: move Arunesh and Taylor to Emeritus (#540) (Michael Dawson) [#540](https://github.com/nodejs/node-addon-api/pull/540)
* [[`cf8b8415df`](https://github.com/nodejs/node-addon-api/commit/cf8b8415df)] - **doc**: add Kevin to the list of collaborators (#539) (Michael Dawson) [#539](https://github.com/nodejs/node-addon-api/pull/539)
* [[`5d6aeae7b5`](https://github.com/nodejs/node-addon-api/commit/5d6aeae7b5)] - **build**: enable travis for fast PR check (legendecas)
* [[`6192e705cd`](https://github.com/nodejs/node-addon-api/commit/6192e705cd)] - **src**: add napi\_date (Mathias Küsel) [#497](https://github.com/nodejs/node-addon-api/pull/497)
* [[`7b1ee96d52`](https://github.com/nodejs/node-addon-api/commit/7b1ee96d52)] - **doc**: update prebuild\_tools.md (Nurbol Alpysbayev) [#527](https://github.com/nodejs/node-addon-api/pull/527)
* [[`0b4f3a5b8c`](https://github.com/nodejs/node-addon-api/commit/0b4f3a5b8c)] - **tsfn**: fix crash on releasing tsfn (legendecas) [#532](https://github.com/nodejs/node-addon-api/pull/532)
* [[`c3c8814d2f`](https://github.com/nodejs/node-addon-api/commit/c3c8814d2f)] - implement virutal ObjectWrap::Finalize (Michael Price) [#515](https://github.com/nodejs/node-addon-api/pull/515)

## 2019-07-23 Version 1.7.1, @NickNaso

### Notable changes:

#### API

- Fixed compilation problems that happen on Node.js with N-API version less than 4.

### Commmits

* [[`c20bcbd069`](https://github.com/nodejs/node-addon-api/commit/c20bcbd069)] - Merge pull request #518 from NickNaso/master (Nicola Del Gobbo)
* [[`6720d57253`](https://github.com/nodejs/node-addon-api/commit/6720d57253)] - Create the native threadsafe\_function for test only for N-API greater than 3. (NickNaso)
* [[`37b6c185ad`](https://github.com/nodejs/node-addon-api/commit/37b6c185ad)] - Fix compilation breakage on 1.7.0 (NickNaso)

## 2019-07-23 Version 1.7.0, @NickNaso

### Notable changes:

#### API

- Added `Napi::ThreadSafeFunction` api.
- Added `Napi::AsyncWorker::GetResult()` method to `Napi::AsyncWorker`.
- Added `Napi::AsyncWorker::Destroy()()` method to `Napi::AsyncWorker`.
- Use full namespace on macros that create the errors.

#### Documentation

- Added documentation about contribution philosophy.
- Added documentation for `Napi::ThreadSafeFunction`.
- Some minor corrections all over the documentation.

#### TEST

- Added test case for bool operator.
- Fixed test case for `Napi::ObjectWrap`.

### Commmits

* [[`717c9ab163`](https://github.com/nodejs/node-addon-api/commit/717c9ab163)] - **AsyncWorker**: add GetResult() method (Kevin Eady) [#512](https://github.com/nodejs/node-addon-api/pull/512)
* [[`d9d991bbc9`](https://github.com/nodejs/node-addon-api/commit/d9d991bbc9)] - **doc**: add ThreadSafeFunction to main README (#513) (Kevin Eady) [#513](https://github.com/nodejs/node-addon-api/pull/513)
* [[`ac6000d0fd`](https://github.com/nodejs/node-addon-api/commit/ac6000d0fd)] - **doc**: fix minor typo (Yohei Kishimoto) [#510](https://github.com/nodejs/node-addon-api/pull/510)
* [[`e9fa1eaa86`](https://github.com/nodejs/node-addon-api/commit/e9fa1eaa86)] - **doc**: document ThreadSafeFunction (#494) (Kevin Eady) [#494](https://github.com/nodejs/node-addon-api/pull/494)
* [[`cab3b1e2a2`](https://github.com/nodejs/node-addon-api/commit/cab3b1e2a2)] - **doc**: ClassPropertyDescriptor example (Ross Weir) [#507](https://github.com/nodejs/node-addon-api/pull/507)
* [[`c32d7dbdcf`](https://github.com/nodejs/node-addon-api/commit/c32d7dbdcf)] - **macros**: create errors fully namespaced (Gabriel Schulhof) [#506](https://github.com/nodejs/node-addon-api/pull/506)
* [[`0a90df2fcb`](https://github.com/nodejs/node-addon-api/commit/0a90df2fcb)] - Implement ThreadSafeFunction class (Jinho Bang)
* [[`1fb540eeb5`](https://github.com/nodejs/node-addon-api/commit/1fb540eeb5)] - Use curly brackets to include node\_api.h (NickNaso) [#493](https://github.com/nodejs/node-addon-api/pull/493)
* [[`b2b08122ea`](https://github.com/nodejs/node-addon-api/commit/b2b08122ea)] - **AsyncWorker**: make callback optional (Kevin Eady) [#489](https://github.com/nodejs/node-addon-api/pull/489)
* [[`a0cac77c82`](https://github.com/nodejs/node-addon-api/commit/a0cac77c82)] - Added test for bool operator (NickNaso) [#490](https://github.com/nodejs/node-addon-api/pull/490)
* [[`ab7d8fcc48`](https://github.com/nodejs/node-addon-api/commit/ab7d8fcc48)] - **src**: fix objectwrap test case (Michael Dawson) [#495](https://github.com/nodejs/node-addon-api/pull/495)
* [[`3b6b9eb88a`](https://github.com/nodejs/node-addon-api/commit/3b6b9eb88a)] - **AsyncWorker**: introduce Destroy() method (Gabriel Schulhof) [#488](https://github.com/nodejs/node-addon-api/pull/488)
* [[`f633fbd95d`](https://github.com/nodejs/node-addon-api/commit/f633fbd95d)] - string.md: Document existing New(env, value, length) APIs (Tux3) [#486](https://github.com/nodejs/node-addon-api/pull/486)
* [[`aaea55eda9`](https://github.com/nodejs/node-addon-api/commit/aaea55eda9)] - Little fix on code example (Nicola Del Gobbo) [#470](https://github.com/nodejs/node-addon-api/pull/470)
* [[`e1cf9a35a1`](https://github.com/nodejs/node-addon-api/commit/e1cf9a35a1)] - Use `Value::IsEmpty` to check for empty value (NickNaso) [#478](https://github.com/nodejs/node-addon-api/pull/478)
* [[`3ad5dfc7d9`](https://github.com/nodejs/node-addon-api/commit/3ad5dfc7d9)] - Fix link (Alba Mendez) [#481](https://github.com/nodejs/node-addon-api/pull/481)
* [[`a3b4d99c45`](https://github.com/nodejs/node-addon-api/commit/a3b4d99c45)] - **doc**: Add contribution philosophy doc (Hitesh Kanwathirtha)
* [[`36863f087b`](https://github.com/nodejs/node-addon-api/commit/36863f087b)] - **doc**: refer to TypedArray and ArrayBuffer from Array (Gabriel "_|Nix|_" Schulhof) [#465](https://github.com/nodejs/node-addon-api/pull/465)

## 2019-04-03 Version 1.6.3, @NickNaso

### Notable changes:

#### API

- Added `SuppressDestruct` method to `Napi::AsyncWorker`.
- Added new build targets for debug.
- Exposed macros that throw errors.
- Fixed memory leaks caused by callback data when a napi error occurs.
- Fixed missing `void *data` usage in `Napi::PropertyDescriptors`.

#### Documentation

- Some minor corrections all over the documentation.

### Commmits

* [[`83b41c2fe4`](https://github.com/nodejs/node-addon-api/commit/83b41c2fe4)] - Document adding -fvisibility=hidden flag for macOS users (Nicola Del Gobbo) [#460](https://github.com/nodejs/node-addon-api/pull/460)
* [[`1ed7ad8769`](https://github.com/nodejs/node-addon-api/commit/1ed7ad8769)] - **doc**: correct return type of Int32Value to int32\_t (Bill Gallafent) [#459](https://github.com/nodejs/node-addon-api/pull/459)
* [[`b0f6b601aa`](https://github.com/nodejs/node-addon-api/commit/b0f6b601aa)] - **src**: add AsyncWorker destruction suppression (Gabriel Schulhof) [#407](https://github.com/nodejs/node-addon-api/pull/407)
* [[`72b1975cff`](https://github.com/nodejs/node-addon-api/commit/72b1975cff)] - **doc**: fix links to the Property Descriptor docs (Ryuichi Okumura) [#458](https://github.com/nodejs/node-addon-api/pull/458)
* [[`fcfc612728`](https://github.com/nodejs/node-addon-api/commit/fcfc612728)] - **build**: new build targets for debug purposes (Jinho Bang) [#186](https://github.com/nodejs/node-addon-api/pull/186)
* [[`c629553cd7`](https://github.com/nodejs/node-addon-api/commit/c629553cd7)] - **doc**: minor doc corrections and clarifications (Bruce A. MacNaughton) [#426](https://github.com/nodejs/node-addon-api/pull/426)
* [[`7b87e0b999`](https://github.com/nodejs/node-addon-api/commit/7b87e0b999)] - **doc**: update number.md (Bernardo Heynemann) [#436](https://github.com/nodejs/node-addon-api/pull/436)
* [[`fcf173d2a1`](https://github.com/nodejs/node-addon-api/commit/fcf173d2a1)] - **src**: expose macros that throw errors (Gabriel Schulhof) [#448](https://github.com/nodejs/node-addon-api/pull/448)
* [[`b409a2f987`](https://github.com/nodejs/node-addon-api/commit/b409a2f987)] - **package**: add npm search keywords (Sam Roberts) [#452](https://github.com/nodejs/node-addon-api/pull/452)
* [[`0bc7987806`](https://github.com/nodejs/node-addon-api/commit/0bc7987806)] - **doc**: fix references to Weak and Persistent (Jake Barnes) [#428](https://github.com/nodejs/node-addon-api/pull/428)
* [[`ad6f569f85`](https://github.com/nodejs/node-addon-api/commit/ad6f569f85)] - **doc**: dix typo (Abhishek Kumar Singh) [#435](https://github.com/nodejs/node-addon-api/pull/435)
* [[`28df833a49`](https://github.com/nodejs/node-addon-api/commit/28df833a49)] - Merge pull request #441 from jschlight/master (Jim Schlight)
* [[`4921e74d83`](https://github.com/nodejs/node-addon-api/commit/4921e74d83)] - Rearranges names to be alphabetical (Jim Schlight)
* [[`48220335b0`](https://github.com/nodejs/node-addon-api/commit/48220335b0)] - Membership review update (Jim Schlight)
* [[`44f0695533`](https://github.com/nodejs/node-addon-api/commit/44f0695533)] - Merge pull request #394 from NickNaso/create\_release (Nicola DelGobbo)
* [[`fa49d68416`](https://github.com/nodejs/node-addon-api/commit/fa49d68416)] - **doc**: fix some `Finalizer` signatures (Philipp Renoth) [#414](https://github.com/nodejs/node-addon-api/pull/414)
* [[`020ac4a628`](https://github.com/nodejs/node-addon-api/commit/020ac4a628)] - **src**: make `Object::GetPropertyNames()` const (Philipp Renoth)[#415](https://github.com/nodejs/node-addon-api/pull/415)
* [[`91eaa6f4cb`](https://github.com/nodejs/node-addon-api/commit/91eaa6f4cb)] - **src**: fix callbackData leaks on error napi status (Philipp Renoth) [#417](https://github.com/nodejs/node-addon-api/pull/417)
* [[`0b40275752`](https://github.com/nodejs/node-addon-api/commit/0b40275752)] - **src**: fix noexcept control flow issues (Philipp Renoth) [#420](https://github.com/nodejs/node-addon-api/pull/420)
* [[`c1ff2936f9`](https://github.com/nodejs/node-addon-api/commit/c1ff2936f9)] - **src**: fix missing void\*data usage in PropertyDescriptors (Luciano Martorella) [#374](https://github.com/nodejs/node-addon-api/pull/374)

## 2018-11-29 Version 1.6.2, @NickNaso

### Notable changes:

#### API

- Fixed selection logic for version 6.x.

### Commmits

* [[`07a0fc4e95`](https://github.com/nodejs/node-addon-api/commit/07a0fc4e95)] - **src**: fix selection logic for 6.x (Michael Dawson) [#402](https://github.com/nodejs/node-addon-api/pull/402)

## 2018-11-14 Version 1.6.1, @NickNaso

### Notable changes:

#### Documentation

- Updated links for examples to point to node-addon-examples repo.
- Fixed typos on some parts of documentation.

#### API

- Removed unused member on `Napi::CallbackScope`.
- Enabled `Napi::CallbackScope` only with N-API v3.

### Commmits

* [[`e7cd292a74`](https://github.com/nodejs/node-addon-api/commit/e7cd292a74)] - **src**: remove unused CallbackScope member (Gabriel Schulhof) [#391](https://github.com/nodejs/node-addon-api/pull/391)
* [[`d47399fe25`](https://github.com/nodejs/node-addon-api/commit/d47399fe25)] - **src**: guard CallbackScope with N-API v3 (Michael Dawson) [#395](https://github.com/nodejs/node-addon-api/pull/395)
* [[`29a0262ab9`](https://github.com/nodejs/node-addon-api/commit/29a0262ab9)] - **doc**: fix typo (Dongjin Na) [#385](https://github.com/nodejs/node-addon-api/pull/385)
* [[`b6dc15b88d`](https://github.com/nodejs/node-addon-api/commit/b6dc15b88d)] - **doc**: make links point to node-addon-examples repo (Nicola Del Gobbo) [#389](https://github.com/nodejs/node-addon-api/pull/389)

## 2018-11-02 Version 1.6.0, @NickNaso

### Notable changes:

#### Documentation

- Improved documentation about ABI stability.

#### API

- Add `Napi::CallbackScope` class that help to have the equivalent of the scope
associated with a callback in place when making certain N-API calls

#### TEST

- Added tests for `Napi::Array` class.
- Added tests for `Napi::ArrayBuffer` class.

### Commmits

* [[`8ce605c657`](https://github.com/nodejs/node-addon-api/commit/8ce605c657)] - **build**: avoid using package-lock.json (Jaeseok Yoon) [#359](https://github.com/nodejs/node-addon-api/pull/359)
* [[`fa3a6150b3`](https://github.com/nodejs/node-addon-api/commit/fa3a6150b3)] - **src**: use MakeCallback() -\> Call() in AsyncWorker (Jinho Bang) [#361](https://github.com/nodejs/node-addon-api/pull/361)
* [[`2342415463`](https://github.com/nodejs/node-addon-api/commit/2342415463)] - **test**: create test objects in the stack instead of the heap (Dongjin Na) [#371](https://github.com/nodejs/node-addon-api/pull/371)
* [[`67b7db0a6f`](https://github.com/nodejs/node-addon-api/commit/67b7db0a6f)] - **test**: write tests for Array class (Jaeseok Yoon) [#363](https://github.com/nodejs/node-addon-api/pull/363)
* [[`729f6dc4ee`](https://github.com/nodejs/node-addon-api/commit/729f6dc4ee)] - **test**: add arraybuffer tests (Dongjin Na) [#369](https://github.com/nodejs/node-addon-api/pull/369)
* [[`405f3e5b5b`](https://github.com/nodejs/node-addon-api/commit/405f3e5b5b)] - **src**: implement CallbackScope class (Jinho Bang) [#362](https://github.com/nodejs/node-addon-api/pull/362)
* [[`015d95312f`](https://github.com/nodejs/node-addon-api/commit/015d95312f)] - **doc**: fix Napi::Reference link (Gentilhomme) [#365](https://github.com/nodejs/node-addon-api/pull/365)
* [[`fd65078e3c`](https://github.com/nodejs/node-addon-api/commit/fd65078e3c)] - README.md: link to new ABI stability guide (Gabriel Schulhof) [#367](https://github.com/nodejs/node-addon-api/pull/367)
* [[`ffebf9ba9a`](https://github.com/nodejs/node-addon-api/commit/ffebf9ba9a)] - Updates for release 1.5.0 (NickNaso)

## 2018-10-03 Version 1.5.0, @NickNaso

### Notable changes:

#### Documentation

- Completed the documentation to cover all the API surface.
- Numerous fixes to make documentation more consistent in all of its parts.

#### API

- Add `Napi::AsyncContext` class to handle asynchronous operation.
- Add `Napi::BigInt` class to work with BigInt type.
- Add `Napi::VersionManagement` class to retrieve the versions of Node.js and N-API.
- Fix potential memory leaks.
- DataView feature is enabled by default
- Add descriptor for Symbols
- Add new methods on `Napi::FunctionReference`.
- Add the possibility to retrieve the environment on `Napi::Promise::Deferred`

#### TOOL

- Add tool to check if a native add-on is built using N-API

#### TEST

- Start to increase the test coverage
- Fix in the test suite to better handle the experimental features that are not
yet backported in the previous Node.js version.

### Commits

* [[`2009c019af`](https://github.com/nodejs/node-addon-api/commit/2009c019af)] - Merge pull request #292 from devsnek/feature/bigint (Gus Caplan)
* [[`e44aca985e`](https://github.com/nodejs/node-addon-api/commit/e44aca985e)] - add bigint class (Gus Caplan)
* [[`a3951ab973`](https://github.com/nodejs/node-addon-api/commit/a3951ab973)] - Add documentation for Env(). (Rolf Timmermans) [#318](https://github.com/nodejs/node-addon-api/pull/318)
* [[`a6f7a6ad51`](https://github.com/nodejs/node-addon-api/commit/a6f7a6ad51)] - Add Env() to Promise::Deferred. (Rolf Timmermans)
* [[`0097e96b92`](https://github.com/nodejs/node-addon-api/commit/0097e96b92)] - Fixed broken links for Symbol and String (NickNaso)
* [[`b0ecd38d76`](https://github.com/nodejs/node-addon-api/commit/b0ecd38d76)] - Fix Code of conduct link properly (#323) (Jake Yoon) [#323](https://github.com/nodejs/node-addon-api/pull/323)
* [[`223474900f`](https://github.com/nodejs/node-addon-api/commit/223474900f)] - **doc**: update Version management (Dongjin Na) [#360](https://github.com/nodejs/node-addon-api/pull/360)
* [[`4f76262a10`](https://github.com/nodejs/node-addon-api/commit/4f76262a10)] - **doc**: some fix on `Napi::Boolean` documentation (NickNaso) [#354](https://github.com/nodejs/node-addon-api/pull/354)
* [[`78374f72d2`](https://github.com/nodejs/node-addon-api/commit/78374f72d2)] - **doc**: number documentation (NickNaso) [#356](https://github.com/nodejs/node-addon-api/pull/356)
* [[`51ffe453f8`](https://github.com/nodejs/node-addon-api/commit/51ffe453f8)] - **doc**: doc cleanup (NickNaso) [#353](https://github.com/nodejs/node-addon-api/pull/353)
* [[`fc11c944b2`](https://github.com/nodejs/node-addon-api/commit/fc11c944b2)] - **doc**: major doc cleanup (NickNaso) [#335](https://github.com/nodejs/node-addon-api/pull/335)
* [[`100d0a7cb2`](https://github.com/nodejs/node-addon-api/commit/100d0a7cb2)] - **doc**: first pass on objectwrap documentation (NickNaso) [#321](https://github.com/nodejs/node-addon-api/pull/321)
* [[`c7d54180ff`](https://github.com/nodejs/node-addon-api/commit/c7d54180ff)] - **doc**: the Napi::ObjectWrap example does not compile (Arnaud Botella) [#339](https://github.com/nodejs/node-addon-api/pull/339)
* [[`7cdd78726a`](https://github.com/nodejs/node-addon-api/commit/7cdd78726a)] - **doc**: added cpp highlight for string.md (Jaeseok Yoon) [#329](https://github.com/nodejs/node-addon-api/pull/329)
* [[`8ed29f547c`](https://github.com/nodejs/node-addon-api/commit/8ed29f547c)] - **doc**: add blurb about ABI stability (Gabriel Schulhof) [#326](https://github.com/nodejs/node-addon-api/pull/326)
* [[`757eb1f5a3`](https://github.com/nodejs/node-addon-api/commit/757eb1f5a3)] - **doc**: add function and function reference doc (NickNaso) [#299](https://github.com/nodejs/node-addon-api/pull/299)
* [[`2885c18591`](https://github.com/nodejs/node-addon-api/commit/2885c18591)] - **doc**: Create changelog for release 1.4.0 (Nicola Del Gobbo)
* [[`917bd60baa`](https://github.com/nodejs/node-addon-api/commit/917bd60baa)] - **src**: remove TODOs by fixing memory leaks (Gabriel Schulhof) [#343](https://github.com/nodejs/node-addon-api/pull/343)
* [[`dfcb93945f`](https://github.com/nodejs/node-addon-api/commit/dfcb93945f)] - **src**: implement AsyncContext class (Jinho Bang) [#252](https://github.com/nodejs/node-addon-api/pull/252)
* [[`211ed38d0d`](https://github.com/nodejs/node-addon-api/commit/211ed38d0d)] - **src**: make 'nothing' target a static library (Gabriel Schulhof) [#348](https://github.com/nodejs/node-addon-api/pull/348)
* [[`97c4ab5cf2`](https://github.com/nodejs/node-addon-api/commit/97c4ab5cf2)] - **src**: add Call and MakeCallback that accept cargs (NickNaso) [#344](https://github.com/nodejs/node-addon-api/pull/344)
* [[`b6e2d92c09`](https://github.com/nodejs/node-addon-api/commit/b6e2d92c09)] - **src**: enable DataView feature by default (Jinho) [#331](https://github.com/nodejs/node-addon-api/pull/331)
* [[`0a00e7c97b`](https://github.com/nodejs/node-addon-api/commit/0a00e7c97b)] - **src**: implement missing descriptor defs for symbols (Philipp Renoth) [#280](https://github.com/nodejs/node-addon-api/pull/280)
* [[`38e01b7e3b`](https://github.com/nodejs/node-addon-api/commit/38e01b7e3b)] - **src**: first pass on adding version management apis (NickNaso) [#325](https://github.com/nodejs/node-addon-api/pull/325)
* [[`79ee8381d2`](https://github.com/nodejs/node-addon-api/commit/79ee8381d2)] - **src**: fix compile failure in test (Michael Dawson) [#345](https://github.com/nodejs/node-addon-api/pull/345)
* [[`4d92a6066f`](https://github.com/nodejs/node-addon-api/commit/4d92a6066f)] - **src**: Add ObjectReference test case (Anisha Rohra) [#212](https://github.com/nodejs/node-addon-api/pull/212)
* [[`779560f397`](https://github.com/nodejs/node-addon-api/commit/779560f397)] - **test**: add operator overloading tests in Number (Your Name) [#355](https://github.com/nodejs/node-addon-api/pull/355)
* [[`73fed84ceb`](https://github.com/nodejs/node-addon-api/commit/73fed84ceb)] - **test**: add ability to control experimental tests (Michael Dawson) [#350](https://github.com/nodejs/node-addon-api/pull/350)
* [[`14c69abd46`](https://github.com/nodejs/node-addon-api/commit/14c69abd46)] - **test**: write tests for Boolean class (Jaeseok Yoon) [#328](https://github.com/nodejs/node-addon-api/pull/328)
* [[`2ad47a83b1`](https://github.com/nodejs/node-addon-api/commit/2ad47a83b1)] - **test**: explicitly cast to uint32\_t in test (Gabriel Schulhof) [#341](https://github.com/nodejs/node-addon-api/pull/341)
* [[`622ffaea76`](https://github.com/nodejs/node-addon-api/commit/622ffaea76)] - **test**: Tighten up compiler warnings (Mikhail Cheshkov) [#315](https://github.com/nodejs/node-addon-api/pull/315)
* [[`fd3c37b0f2`](https://github.com/nodejs/node-addon-api/commit/fd3c37b0f2)] - **tools**: add tool to check for N-API modules (Gabriel Schulhof) [#346](https://github.com/nodejs/node-addon-api/pull/346)

## 2018-07-19 Version 1.4.0, @NickNaso

### Notable changes:

#### Documentation

- Numerous additions to the documentation, filling out coverage
  of API surface

#### API

- Add resource parameters to AsyncWorker constructor
- Add memory management feature

### Commits

* [[`7dc5ac8bc3`](https://github.com/nodejs/node-addon-api/commit/7dc5ac8bc3)] - **doc**: update metadata for release (Nicola Del Gobbo)
* [[`d68e86adb4`](https://github.com/nodejs/node-addon-api/commit/d68e86adb4)] - **doc**: Added documentation for PropertyDescriptor (Anisha Rohra) [#309](https://github.com/nodejs/node-addon-api/pull/309)
* [[`968a5f2000`](https://github.com/nodejs/node-addon-api/commit/968a5f2000)] - **doc**: Add documentation for ObjectReference.md (Anisha Rohra) [#307](https://github.com/nodejs/node-addon-api/pull/307)
* [[`908cdc314c`](https://github.com/nodejs/node-addon-api/commit/908cdc314c)] - **doc**: add `TypedArray` and `TypedArrayOf` (Kyle Farnung) [#305](https://github.com/nodejs/node-addon-api/pull/305)
* [[`2ff776ffe3`](https://github.com/nodejs/node-addon-api/commit/2ff776ffe3)] - backport node::Persistent (Gabriel Schulhof) [#300](https://github.com/nodejs/node-addon-api/pull/300)
* [[`98161970c9`](https://github.com/nodejs/node-addon-api/commit/98161970c9)] - Backport perf, crash and exception handling fixes (Gabriel Schulhof) [#295](https://github.com/nodejs/node-addon-api/pull/295)
* [[`dd1191e086`](https://github.com/nodejs/node-addon-api/commit/dd1191e086)] - **test**: fix asyncworker test so it runs on 6.x (Michael Dawson) [#298](https://github.com/nodejs/node-addon-api/pull/298)
* [[`11697fcecd`](https://github.com/nodejs/node-addon-api/commit/11697fcecd)] - **doc**: ArrayBuffer and Buffer documentation (Kyle Farnung) [#256](https://github.com/nodejs/node-addon-api/pull/256)
* [[`605aa2babf`](https://github.com/nodejs/node-addon-api/commit/605aa2babf)] - Add memory management feature (NickNaso) [#286](https://github.com/nodejs/node-addon-api/pull/286)
* [[`86be13a611`](https://github.com/nodejs/node-addon-api/commit/86be13a611)] - **doc**: Fix HandleScope docs (Ben Berman) [#287](https://github.com/nodejs/node-addon-api/pull/287)
* [[`90f92c4dc0`](https://github.com/nodejs/node-addon-api/commit/90f92c4dc0)] - **doc**: Update broken links in README.md (Hitesh Kanwathirtha) [#290](https://github.com/nodejs/node-addon-api/pull/290)
* [[`c2a620dc11`](https://github.com/nodejs/node-addon-api/commit/c2a620dc11)] - **doc**: Clarify positioning versus N-API (Michael Dawson) [#288](https://github.com/nodejs/node-addon-api/pull/288)
* [[`6cff890ee5`](https://github.com/nodejs/node-addon-api/commit/6cff890ee5)] - **doc**: Fix typo in docs (Ben Berman) [#284](https://github.com/nodejs/node-addon-api/pull/284)
* [[`7394bfd154`](https://github.com/nodejs/node-addon-api/commit/7394bfd154)] - **doc**: Fix typo in docs (Ben Berman) [#285](https://github.com/nodejs/node-addon-api/pull/285)
* [[`12b2cdeed3`](https://github.com/nodejs/node-addon-api/commit/12b2cdeed3)] - fix test files (Kyle Farnung) [#257](https://github.com/nodejs/node-addon-api/pull/257)
* [[`9ab6607242`](https://github.com/nodejs/node-addon-api/commit/9ab6607242)] - **doc**: Update Doc Version Number (joshgarde) [#277](https://github.com/nodejs/node-addon-api/pull/277)
* [[`e029a076c6`](https://github.com/nodejs/node-addon-api/commit/e029a076c6)] - **doc**: First pass at basic Node Addon API docs (Hitesh Kanwathirtha) [#268](https://github.com/nodejs/node-addon-api/pull/268)
* [[`74ff79717e`](https://github.com/nodejs/node-addon-api/commit/74ff79717e)] - **doc**: fix link to async\_worker.md (Michael Dawson)
* [[`5a63f45eda`](https://github.com/nodejs/node-addon-api/commit/5a63f45eda)] - **doc**: First step of error and async doc (NickNaso) [#272](https://github.com/nodejs/node-addon-api/pull/272)
* [[`9d38f61afb`](https://github.com/nodejs/node-addon-api/commit/9d38f61afb)] - **doc**: New Promise and Reference docs (Jim Schlight) [#243](https://github.com/nodejs/node-addon-api/pull/243)
* [[`43ff9fa836`](https://github.com/nodejs/node-addon-api/commit/43ff9fa836)] - **doc**: Updated Object documentation (Anisha Rohra) [#254](https://github.com/nodejs/node-addon-api/pull/254)
* [[`b197f7cc8b`](https://github.com/nodejs/node-addon-api/commit/b197f7cc8b)] - **doc**: minor typos (Nick Soggin) [#248](https://github.com/nodejs/node-addon-api/pull/248)
* [[`4b8918b352`](https://github.com/nodejs/node-addon-api/commit/4b8918b352)] - Add resource parameters to AsyncWorker constructor (Jinho Bang) [#253](https://github.com/nodejs/node-addon-api/pull/253)
* [[`1ecf7c19b6`](https://github.com/nodejs/node-addon-api/commit/1ecf7c19b6)] - **doc**: fix wrong link in readme (miloas) [#255](https://github.com/nodejs/node-addon-api/pull/255)
* [[`a750ed1932`](https://github.com/nodejs/node-addon-api/commit/a750ed1932)] - **release**: updates to metadata for next release (Michael Dawson)

## 2018-05-08 Version 1.3.0, @mhdawson

### Notable changes:

#### Documentation
- Added documentation for Scopes
- Added documentation for migration from NAN
- Update documentation to better explain the use of NODE_ADDON_API

#### API
- Implement data manipulation methods for dataview
- Use built-in N-API on Node.js >= 6.14.2
- Value
  - Added IsExternal()
  - IsObject() allow functions
- String
  - Fixed initialization of std::string to nullptr

#### Tests
- Fix test failures on linuxOne and AIX
- Added basic tests for Scopes
- Fix MSVC warning C4244 in tests

### Commits

* [386c2aeb74] - test: remove dep on later C++ feature (Michael Dawson) https://github.com/nodejs/node-addon-api/pull/267
* [10697734da] - Use built-in N-API on Node.js >= 6.14.2 (Gabriel Schulhof)
* [75086da273] - test: add basic tests and doc for scopes (Michael Dawson) https://github.com/nodejs/node-addon-api/pull/250
* [341dbd25d5] - doc: update blurb explaining NODE_ADDON_API (Gabriel Schulhof) https://github.com/nodejs/node-addon-api/pull/251
* [cf6c93e4ee] - don't try to escape null (Michael Dawson) https://github.com/nodejs/node-addon-api/pull/245
* [15e4b35fc2] - test: fix MSVC warning C4244 in tests (Kyle Farnung) https://github.com/nodejs/node-addon-api/pull/236
* [7f3ca03b8e] - Create a doc for migration (Sampson Gao) https://github.com/nodejs/node-addon-api/pull/118
* [0a2177debe] - Fix test failures on linuxOne and AIX (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/232
* [d567f4b6b5] - Added Napi::Value::IsExternal() (Eric Bickle) https://github.com/nodejs/node-addon-api/pull/227
* [1b0f0e004a] - Update node-gyp.md (Michele Campus) https://github.com/nodejs/node-addon-api/pull/226
* [faf19c4f7a] - Fixed initialization of std::string to nullptr (Eric Bickle) https://github.com/nodejs/node-addon-api/pull/228
* [9c4d321b57] - Implement data manipulation methods for dataview (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/218
* [5a39fdca6f] - n-api: throw RangeError napi_create_typedarray() (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/216
* [1376377202] - Make IsObject() allow functions (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/217
* [673b59d319] - src: Initial implementation of DataView class (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/205
* [0a899bf1c5] - doc: update indication of latest version (Michael Dawson) https://github.com/nodejs/node-addon-api/pull/211
* [17c74e5a5e] - n-api: RangeError in napi_create_dataview() (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/214
* [4058a29989] - n-api: fix memory leak in napi_async_destroy() (Jinho Bang) https://github.com/nodejs/node-addon-api/pull/213


