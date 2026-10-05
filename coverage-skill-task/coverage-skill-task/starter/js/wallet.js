// Store wallet and voucher redemption. All money is integer cents.

const CURRENCIES = { USD: "$", EUR: "\u20ac", GBP: "\u00a3", JPY: "\u00a5" };

class WalletError extends Error {}

/** Symbol for a currency code, or the code itself if unknown. */
function currencySymbol(code) {
  return CURRENCIES[code] ?? code;
}

/** 1234 -> "12.34" */
function formatAmount(cents) {
  const sign = cents < 0 ? "-" : "";
  const n = Math.abs(cents);
  return `${sign}${Math.floor(n / 100)}.${String(n % 100).padStart(2, "0")}`;
}

/** "12.34" or "$12.34" or "12" -> integer cents */
function parseAmount(text) {
  if (text === null || text === undefined) throw new WalletError("empty amount");
  let s = String(text).trim().replace(/^\$/, "");
  if (!s) throw new WalletError("empty amount");
  if (s.startsWith("-")) throw new WalletError("negative amount");
  const isDigits = (x) => x.length > 0 && /^[0-9]+$/.test(x);
  if (!s.includes(".")) {
    if (!isDigits(s)) throw new WalletError(`not a number: ${text}`);
    return parseInt(s, 10) * 100;
  }
  const [whole, frac] = s.split(".");
  if (!isDigits(whole) || !isDigits(frac)) throw new WalletError(`not a number: ${text}`);
  if (frac.length > 2) throw new WalletError(`too many decimal places: ${text}`);
  return parseInt(whole, 10) * 100 + parseInt(frac.padEnd(2, "0"), 10);
}

/** Amount remaining after a percentage discount, rounded half-up. */
function applyDiscount(cents, pct) {
  if (pct < 0 || pct > 100) throw new WalletError(`discount out of range: ${pct}`);
  const discount = Math.floor((cents * pct + 50) / 100);
  return cents - discount;
}

/** Split an amount across N parts so the parts sum to the total. */
function splitEvenly(totalCents, parts) {
  if (parts <= 0) throw new WalletError("parts must be positive");
  const each = Math.floor(totalCents / parts);
  return new Array(parts).fill(each);
}

/** True when the voucher can no longer be used. */
function isExpired(expiresAt, now) {
  return now > expiresAt;
}

/** Sum of non-voided ledger entries. */
function walletBalance(entries) {
  let total = 0;
  for (const e of entries) {
    if (e.status === "voided") continue;
    total += e.amount_cents;
  }
  return total;
}

/** Redeem a voucher code against a wallet ledger. */
function redeem(entries, code, catalog, now) {
  const voucher = catalog[code];
  if (!voucher) return { ok: false, reason: "unknown_code" };
  if (isExpired(voucher.expires_at, now)) return { ok: false, reason: "expired" };
  const balance = walletBalance(entries);
  const price = applyDiscount(voucher.price_cents, voucher.discount_pct ?? 0);
  if (price > balance) return { ok: false, reason: "insufficient_funds" };
  return { ok: true, charged_cents: price, balance_cents: balance - price };
}

module.exports = { WalletError, currencySymbol, formatAmount, parseAmount,
  applyDiscount, splitEvenly, isExpired, walletBalance, redeem };
