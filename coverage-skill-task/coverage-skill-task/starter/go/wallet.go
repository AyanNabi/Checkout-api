// Package wallet handles store wallets and voucher redemption.
// All money is integer cents.
package wallet

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var currencies = map[string]string{"USD": "$", "EUR": "\u20ac", "GBP": "\u00a3", "JPY": "\u00a5"}

var ErrWallet = errors.New("wallet error")

type Entry struct {
	AmountCents int
	Status      string
}

type Voucher struct {
	PriceCents  int
	ExpiresAt   int64
	DiscountPct int
}

type Result struct {
	OK            bool
	Reason        string
	ChargedCents  int
	BalanceCents  int
}

// CurrencySymbol returns the symbol for a code, or the code itself if unknown.
func CurrencySymbol(code string) string {
	if s, ok := currencies[code]; ok {
		return s
	}
	return code
}

// FormatAmount turns 1234 into "12.34".
func FormatAmount(cents int) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ParseAmount turns "12.34", "$12.34" or "12" into integer cents.
func ParseAmount(text string) (int, error) {
	s := strings.TrimPrefix(strings.TrimSpace(text), "$")
	if s == "" {
		return 0, fmt.Errorf("%w: empty amount", ErrWallet)
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("%w: negative amount", ErrWallet)
	}
	if !strings.Contains(s, ".") {
		if !isDigits(s) {
			return 0, fmt.Errorf("%w: not a number: %s", ErrWallet, text)
		}
		n, _ := strconv.Atoi(s)
		return n * 100, nil
	}
	parts := strings.SplitN(s, ".", 2)
	whole, frac := parts[0], parts[1]
	if !isDigits(whole) || !isDigits(frac) {
		return 0, fmt.Errorf("%w: not a number: %s", ErrWallet, text)
	}
	if len(frac) > 2 {
		return 0, fmt.Errorf("%w: too many decimal places: %s", ErrWallet, text)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, _ := strconv.Atoi(whole)
	f, _ := strconv.Atoi(frac)
	return w*100 + f, nil
}

// ApplyDiscount returns the amount remaining after a percentage discount,
// rounded half-up.
func ApplyDiscount(cents, pct int) (int, error) {
	if pct < 0 || pct > 100 {
		return 0, fmt.Errorf("%w: discount out of range: %d", ErrWallet, pct)
	}
	discount := (cents*pct + 50) / 100
	return cents - discount, nil
}

// SplitEvenly splits an amount across n parts so the parts sum to the total.
func SplitEvenly(totalCents, parts int) ([]int, error) {
	if parts <= 0 {
		return nil, fmt.Errorf("%w: parts must be positive", ErrWallet)
	}
	each := totalCents / parts
	out := make([]int, parts)
	for i := range out {
		out[i] = each
	}
	return out, nil
}

// IsExpired reports whether the voucher can no longer be used.
func IsExpired(expiresAt, now int64) bool {
	return now > expiresAt
}

// WalletBalance sums the non-voided ledger entries.
func WalletBalance(entries []Entry) int {
	total := 0
	for _, e := range entries {
		if e.Status == "voided" {
			continue
		}
		total += e.AmountCents
	}
	return total
}

// Redeem applies a voucher code against a wallet ledger.
func Redeem(entries []Entry, code string, catalog map[string]Voucher, now int64) Result {
	v, ok := catalog[code]
	if !ok {
		return Result{OK: false, Reason: "unknown_code"}
	}
	if IsExpired(v.ExpiresAt, now) {
		return Result{OK: false, Reason: "expired"}
	}
	balance := WalletBalance(entries)
	price, err := ApplyDiscount(v.PriceCents, v.DiscountPct)
	if err != nil {
		return Result{OK: false, Reason: "bad_voucher"}
	}
	if price > balance {
		return Result{OK: false, Reason: "insufficient_funds"}
	}
	return Result{OK: true, ChargedCents: price, BalanceCents: balance - price}
}
