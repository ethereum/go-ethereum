# Missing Price Data

Two possible reasons for this:
- [Default API key](https://github.com/cgewecke/eth-gas-reporter/blob/23fc57687b4e190c7e28571a14773d96cdbf7d63/lib/config.js#L12) reached a usage cap.
- The tests ran too quickly and price data couldn't be fetched before returning the test. You can manually slow down your tests with a dummy

To slow down unit tests, you can add a dummy test like this:
```js
// Wait so the reporter has time to fetch and return prices from APIs.
// https://github.com/cgewecke/eth-gas-reporter/issues/254
describe("eth-gas-reporter workaround", () => {
  it("should kill time", (done) => {
    setTimeout(done, 2000);
  });
});
```