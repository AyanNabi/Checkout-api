const test = require("node:test");
const assert = require("node:assert");
const { currencySymbol, formatAmount, parseAmount } = require("./wallet");

test("currency symbol known", () => assert.strictEqual(currencySymbol("USD"), "$"));
test("currency symbol unknown", () => assert.strictEqual(currencySymbol("XYZ"), "XYZ"));
test("format amount", () => {
  assert.strictEqual(formatAmount(1234), "12.34");
  assert.strictEqual(formatAmount(5), "0.05");
});
test("parse amount basic", () => {
  assert.strictEqual(parseAmount("12.34"), 1234);
  assert.strictEqual(parseAmount("12"), 1200);
});
