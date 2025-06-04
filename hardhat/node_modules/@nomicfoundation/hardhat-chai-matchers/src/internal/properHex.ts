export function supportProperHex(Assertion: Chai.AssertionStatic) {
  Assertion.addMethod("properHex", function (this: any, length: number) {
    const subject = this._obj;
    const isNegated = this.__flags.negate === true;

    const isHex = (a: string) => /^0x[0-9a-fA-F]*$/.test(a);
    if (!isHex(subject)) {
      this.assert(
        isNegated, // trick to make this assertion always fail
        `Expected "${subject}" to be a proper hex string, but it contains invalid (non-hex) characters`,
        `Expected "${subject}" NOT to be a proper hex string, but it contains only valid hex characters`
      );
    }

    this.assert(
      subject.length === length + 2,
      `Expected "${subject}" to be a hex string of length ${
        length + 2
      } (the provided ${length} plus 2 more for the "0x" prefix), but its length is ${
        subject.length
      }`,
      `Expected "${subject}" NOT to be a hex string of length ${
        length + 2
      } (the provided ${length} plus 2 more for the "0x" prefix), but its length is ${
        subject.length
      }`
    );
  });
}
