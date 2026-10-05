# Python starter

```bash
pip install pytest coverage
python3 -m coverage run -m pytest -q
python3 -m coverage report -m --include=wallet.py
```

Baseline: **46%** (4 tests, 57 statements, 31 missed).
Put new tests in this folder as `test_*.py`.
