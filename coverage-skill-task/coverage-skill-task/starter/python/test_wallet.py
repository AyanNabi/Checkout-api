from wallet import currency_symbol, format_amount, parse_amount


def test_currency_symbol_known():
    assert currency_symbol("USD") == "$"


def test_currency_symbol_unknown():
    assert currency_symbol("XYZ") == "XYZ"


def test_format_amount():
    assert format_amount(1234) == "12.34"
    assert format_amount(5) == "0.05"


def test_parse_amount_basic():
    assert parse_amount("12.34") == 1234
    assert parse_amount("12") == 1200
