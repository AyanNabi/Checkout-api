"""Store wallet and voucher redemption. All money is integer cents."""

CURRENCIES = {"USD": "$", "EUR": "\u20ac", "GBP": "\u00a3", "JPY": "\u00a5"}


class WalletError(Exception):
    pass


def currency_symbol(code):
    """Symbol for a currency code, or the code itself if unknown."""
    return CURRENCIES.get(code, code)


def format_amount(cents):
    """1234 -> '12.34'."""
    sign = "-" if cents < 0 else ""
    cents = abs(cents)
    return "%s%d.%02d" % (sign, cents // 100, cents % 100)


def parse_amount(text):
    """'12.34' or '$12.34' or '12' -> integer cents."""
    if text is None:
        raise WalletError("empty amount")
    s = str(text).strip().lstrip("$")
    if not s:
        raise WalletError("empty amount")
    if s.startswith("-"):
        raise WalletError("negative amount")
    if "." not in s:
        if not s.isdigit():
            raise WalletError("not a number: %s" % text)
        return int(s) * 100
    whole, _, frac = s.partition(".")
    if not whole.isdigit() or not frac.isdigit():
        raise WalletError("not a number: %s" % text)
    if len(frac) > 2:
        raise WalletError("too many decimal places: %s" % text)
    return int(whole) * 100 + int(frac.ljust(2, "0"))


def apply_discount(cents, pct):
    """Amount remaining after a percentage discount, rounded half-up."""
    if pct < 0 or pct > 100:
        raise WalletError("discount out of range: %s" % pct)
    discount = int((cents * pct + 50) // 100)
    return cents - discount


def split_evenly(total_cents, parts):
    """Split an amount across N parts so the parts sum to the total."""
    if parts <= 0:
        raise WalletError("parts must be positive")
    each = total_cents // parts
    return [each for _ in range(parts)]


def is_expired(expires_at, now):
    """True when the voucher can no longer be used."""
    return now > expires_at


def wallet_balance(entries):
    """Sum of non-voided ledger entries."""
    total = 0
    for e in entries:
        if e.get("status") == "voided":
            continue
        total += e["amount_cents"]
    return total


def redeem(entries, code, catalog, now):
    """Redeem a voucher code against a wallet ledger."""
    voucher = catalog.get(code)
    if voucher is None:
        return {"ok": False, "reason": "unknown_code"}
    if is_expired(voucher["expires_at"], now):
        return {"ok": False, "reason": "expired"}
    balance = wallet_balance(entries)
    price = apply_discount(voucher["price_cents"], voucher.get("discount_pct", 0))
    if price > balance:
        return {"ok": False, "reason": "insufficient_funds"}
    return {"ok": True, "charged_cents": price, "balance_cents": balance - price}
