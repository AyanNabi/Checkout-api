package wallet

import (
	"errors"
	"testing"
)

func TestCurrencySymbolKnown(t *testing.T) {
	if CurrencySymbol("USD") != "$" {
		t.Fatal("want $")
	}
	if CurrencySymbol("EUR") != "€" {
		t.Fatal("want €")
	}
}

func TestCurrencySymbolUnknown(t *testing.T) {
	if CurrencySymbol("XYZ") != "XYZ" {
		t.Fatal("want XYZ")
	}
}

func TestFormatAmount(t *testing.T) {
	if FormatAmount(1234) != "12.34" {
		t.Fatal("want 12.34")
	}
	if FormatAmount(5) != "0.05" {
		t.Fatal("want 0.05")
	}
	if FormatAmount(-1234) != "-12.34" {
		t.Fatalf("want -12.34, got %s", FormatAmount(-1234))
	}
	if FormatAmount(-5) != "-0.05" {
		t.Fatalf("want -0.05, got %s", FormatAmount(-5))
	}
}

func TestParseAmountBasic(t *testing.T) {
	if n, _ := ParseAmount("12.34"); n != 1234 {
		t.Fatal("want 1234")
	}
	if n, _ := ParseAmount("12"); n != 1200 {
		t.Fatal("want 1200")
	}
	if n, _ := ParseAmount("  $12.3  "); n != 1230 {
		t.Fatalf("want 1230, got %d", n)
	}
}

func TestParseAmountErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"negative", "-12.34"},
		{"invalid whole", "12a.34"},
		{"invalid frac", "12.3a"},
		{"too many decimals", "12.345"},
		{"non-numeric single string", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseAmount(tt.input)
			if err == nil {
				t.Fatalf("expected error for input %q, got nil", tt.input)
			}
			if !errors.Is(err, ErrWallet) {
				t.Fatalf("expected ErrWallet wrapped, got %v", err)
			}
		})
	}
}

func TestApplyDiscount(t *testing.T) {
	// Valid scenarios
	res, err := ApplyDiscount(1000, 20) // 20% off $10.00 -> $8.00
	if err != nil || res != 800 {
		t.Fatalf("expected 800, got %d (err: %v)", res, err)
	}

	// Rounding half-up: 15% off 105 cents = 15.75 cents discount -> rounded 16 -> 89 remaining
	res, err = ApplyDiscount(105, 15)
	if err != nil || res != 89 {
		t.Fatalf("expected 89, got %d", res)
	}

	// Boundary checks
	_, err = ApplyDiscount(1000, -1)
	if err == nil {
		t.Fatal("expected error for negative discount percentage")
	}

	_, err = ApplyDiscount(1000, 101)
	if err == nil {
		t.Fatal("expected error for discount percentage > 100")
	}
}

func TestSplitEvenly_Validation(t *testing.T) {
	_, err := SplitEvenly(100, 0)
	if err == nil {
		t.Fatal("expected error when parts <= 0")
	}

	_, err = SplitEvenly(100, -2)
	if err == nil {
		t.Fatal("expected error when parts < 0")
	}
}

func TestSplitEvenly_PreservesTotal(t *testing.T) {
	// Exact split
	parts, err := SplitEvenly(100, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parts) != 2 || parts[0] != 50 || parts[1] != 50 {
		t.Fatalf("unexpected split: %v", parts)
	}
}

func TestSplitEvenly_RemainderBug(t *testing.T) {
	parts, err := SplitEvenly(100, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sum := 0
	for _, p := range parts {
		sum += p
	}

	if sum != 100 {
		t.Fatalf("Split sum %d does not equal total cents 100 (lost remainder cents)", sum)
	}
}

func TestWalletBalance(t *testing.T) {
	entries := []Entry{
		{AmountCents: 1000, Status: "cleared"},
		{AmountCents: 500, Status: "pending"},
		{AmountCents: 2000, Status: "voided"},
	}

	if bal := WalletBalance(entries); bal != 1500 {
		t.Fatalf("expected 1500, got %d", bal)
	}
}

func TestRedeem(t *testing.T) {
	now := int64(1000)
	catalog := map[string]Voucher{
		"VALID10": {PriceCents: 1000, DiscountPct: 10, ExpiresAt: 2000},
		"EXPIRED": {PriceCents: 1000, DiscountPct: 10, ExpiresAt: 500},
		"BADPCT":  {PriceCents: 1000, DiscountPct: 150, ExpiresAt: 2000},
	}

	entries := []Entry{
		{AmountCents: 2000, Status: "cleared"},
		{AmountCents: 500, Status: "voided"},
	}

	// 1. Unknown voucher code
	res := Redeem(entries, "MISSING", catalog, now)
	if res.OK || res.Reason != "unknown_code" {
		t.Fatalf("unexpected result: %+v", res)
	}

	// 2. Expired voucher
	res = Redeem(entries, "EXPIRED", catalog, now)
	if res.OK || res.Reason != "expired" {
		t.Fatalf("unexpected result: %+v", res)
	}

	// 3. Invalid voucher configuration (discount > 100%)
	res = Redeem(entries, "BADPCT", catalog, now)
	if res.OK || res.Reason != "bad_voucher" {
		t.Fatalf("unexpected result: %+v", res)
	}

	// 4. Insufficient funds
	lowEntries := []Entry{{AmountCents: 100, Status: "cleared"}}
	res = Redeem(lowEntries, "VALID10", catalog, now)
	if res.OK || res.Reason != "insufficient_funds" {
		t.Fatalf("unexpected result: %+v", res)
	}

	// 5. Successful redemption ($10.00 price - 10% discount = $9.00 charge; balance $20.00 -> remaining $11.00)
	res = Redeem(entries, "VALID10", catalog, now)
	if !res.OK || res.ChargedCents != 900 || res.BalanceCents != 1100 {
		t.Fatalf("unexpected result: %+v", res)
	}
}
