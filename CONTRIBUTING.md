# Contributing to Zenqo

Thanks for your interest in contributing to Zenqo! Here's how you can help.

## Getting Started

```bash
git clone https://github.com/zenqos/zenqo.git
cd zenqo
go test ./...
```

## Development

### Project Structure

```
core/           # Framework core — handlers, routing, validation, errors
internal/       # Internal packages (encoding, logging)
middleware/     # Built-in middleware (CORS, secure headers)
cmd/zenqo/     # CLI tool
examples/       # Example applications
```

### Running Tests

```bash
# All tests
go test ./... -v -count=1

# With race detection
go test ./... -race

# Coverage
go test ./core/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# Benchmarks
go test -bench=. -benchmem ./core/
```

## How to Contribute

### Reporting Bugs

Open an issue using the [Bug Report](https://github.com/zenqos/zenqo/issues/new?template=bug_report.md) template. Include a minimal reproducible example.

### Suggesting Features

Open an issue using the [Feature Request](https://github.com/zenqos/zenqo/issues/new?template=feature_request.md) template. Describe the proposed API.

### Submitting Code

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Write tests for your changes
4. Make sure all tests pass (`go test ./... -race`)
5. Commit with a clear message following the convention below
6. Open a Pull Request

### Commit Convention

```
type :: scope - description
```

Types: `feat`, `fix`, `refactor`, `style`, `docs`, `chore`, `test`

Examples:
```
feat :: core - add rate limiting middleware
fix :: validate - handle nil pointer in nested struct
docs :: readme - add authentication example
test :: controller - add adapt() edge case tests
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Exported symbols must have GoDoc comments
- Tests go in `_test.go` files in the same package
- Keep dependencies minimal — the framework only depends on `chi`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
