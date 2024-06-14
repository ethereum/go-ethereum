export function supportHexEqual(Assertion: Chai.AssertionStatic) {
  Assertion.addMethod("hexEqual", function (this: any, other: string) {
    const subject = this._obj;
    const isNegated = this.__flags.negate === true;

    // check that both values are proper hex strings
    const isHex = (a: string) => /^0x[0-9a-fA-F]*$/.test(a);
    for (const element of [subject, other]) {
      if (!isHex(element)) {
        this.assert(
          isNegated, // trick to make this assertion always fail
          `Expected "${subject}" to be a hex string equal to "${other}", but "${element}" is not a valid hex string`,
          `Expected "${subject}" not to be a hex string equal to "${other}", but "${element}" is not a valid hex string`
        );
      }
    }

    // compare values
    const extractNumeric = (hex: string) => hex.replace(/^0x0*/, "");
    this.assert(
      extractNumeric(subject.toLowerCase()) ===
        extractNumeric(other.toLowerCase()),
      `Expected "${subject}" to be a hex string equal to "${other}"`,
      `Expected "${subject}" NOT to be a hex string equal to "${other}", but it was`,
      `Hex string representing the same number as ${other}`,
      subject
    );
  });
}
