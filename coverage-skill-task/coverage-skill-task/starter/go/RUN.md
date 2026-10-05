# Go starter

```bash
go test -cover ./...
go test -coverprofile=c.out ./... && go tool cover -func=c.out
```

Baseline: **38.5% of statements**.
Put new tests in this folder as `*_test.go` in package `wallet`.
