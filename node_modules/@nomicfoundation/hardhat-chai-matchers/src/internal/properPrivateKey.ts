export function supportProperPrivateKey(Assertion: Chai.AssertionStatic) {
  Assertion.addProperty("properPrivateKey", function (this: any) {
    const subject = this._obj;
    this.assert(
      /^0x[0-9a-fA-F]{64}$/.test(subject),
      `Expected "${subject}" to be a proper private key`,
      `Expected "${subject}" NOT to be a proper private key`,
      "proper private key (eg.: 0x1010101010101010101010101010101010101010101010101010101010101010)",
      subject
    );
  });
}
