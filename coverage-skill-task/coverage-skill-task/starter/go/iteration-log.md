candidate                           | coverage before -> after | kept/discarded | reason
------------------------------------|--------------------------|----------------|--------------------------------------------------
SplitEvenly (invalid parts <= 0)    | 38.5% -> 42.3%           | Kept           | Essential error branch for invalid input.
SplitEvenly (remainder distribution)| 42.3% -> 42.3%           | Kept as bug-finding test| SplitEvenly(100, 3) returned parts totaling 99 instead of 100. The test checks the required invariant that splitting an amount must preserve the total. The failure was investigated and classified as a production bug rather than weakening the test. 
ApplyDiscount (out of range pct)    | 42.3% -> 50.0%           | Kept           | Required validation path (`pct < 0` / `pct > 100`).
ApplyDiscount (rounding logic)      | 50.0% -> 50.0%           | Kept           | Tests half-up rounding boundary.
ParseAmount (empty / whitespace)    | 50.0% -> 53.8%           | Kept           | Error branch coverage.
ParseAmount (negative input error)  | 53.8% -> 57.7%           | Kept           | Error branch coverage.
ParseAmount (invalid format/chars)  | 57.7% -> 69.2%           | Kept           | Validates `isDigits` non-numeric branches.
ParseAmount (fractional cases)      | 69.2% -> 76.9%           | Kept           | Tests single digit and excess decimal rules.
Redeem (unknown/expired/bad/funds)  | 76.9% -> 96.2%           | Kept           | High-value domain business logic branches.
WalletBalance (voided entries)      | 96.2% -> 96.2%           | Kept           | Behavioral test for ledger status filtering.
FormatAmount (negative amounts)     | 96.2% -> 98.5%          | Kept           | Edge branch for negative formatting.