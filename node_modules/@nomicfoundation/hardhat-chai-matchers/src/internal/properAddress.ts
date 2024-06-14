export function supportProperAddress(Assertion: Chai.AssertionStatic) {
  Assertion.addProperty("properAddress", function (this: any) {
    const subject = this._obj;
    this.assert(
      /^0x[0-9a-fA-F]{40}$/.test(subject),
      `Expected "${subject}" to be a proper address`,
      `Expected "${subject}" NOT to be a proper address`,
      "proper address (eg.: 0x1234567890123456789012345678901234567890)",
      subject
    );
  });
}
