# Testing Guidelines for goncat

## Overview

This document outlines the testing approach and guidelines for the goncat project. We focus on comprehensive unit testing to ensure code quality, maintainability, and reliability.

## Unit Testing Philosophy

- **Focus on unit tests**: Test individual functions and methods in isolation
- **No integration tests in unit tests**: Integration tests are separate (see `tests/` directory)
- **Table-driven tests**: Use table-driven test patterns for multiple test cases
- **Test behavior, not implementation**: Focus on inputs, outputs, and edge cases
- **Isolation**: Mock external dependencies when necessary

## Test File Structure

### Naming Convention
- Test files should be named `*_test.go`
- Place test files in the same package as the code being tested
- Example: `pkg/config/config.go` → `pkg/config/config_test.go`

### Test Function Naming
- Use descriptive names: `Test<FunctionName>` or `Test<FunctionName>_<Scenario>`
- Examples:
  - `TestParseTransport`
  - `TestValidatePort_InvalidRange`
  - `TestNew_WithNilContext`

## Test Structure Pattern

### Table-Driven Tests (Preferred)

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string      // test case description
        input    InputType   // input parameters
        expected OutputType  // expected output
        wantErr  bool        // whether error is expected
    }{
        {
            name:     "valid input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    invalidInput,
            expected: zeroValue,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName(%v) error = %v, wantErr %v", 
                    tt.input, err, tt.wantErr)
                return
            }
            
            if !tt.wantErr && result != tt.expected {
                t.Errorf("FunctionName(%v) = %v, want %v", 
                    tt.input, result, tt.expected)
            }
        })
    }
}
```

### Simple Test Pattern

For simple functions with few cases:

```go
func TestSimpleFunction(t *testing.T) {
    result := SimpleFunction(input)
    
    if result != expected {
        t.Errorf("SimpleFunction(%v) = %v, want %v", input, result, expected)
    }
}
```

## What to Test

### Priority 1: Public APIs
- All exported functions and methods
- Constructors (New functions)
- Interface implementations
- Public struct methods

### Priority 2: Edge Cases
- Nil inputs
- Empty strings/slices
- Boundary values (min, max, 0, -1)
- Invalid inputs

### Priority 3: Error Handling
- Functions that return errors
- Error message content (when important)
- Proper cleanup on errors

## What NOT to Test

- Private/unexported functions (test through public APIs)
- Trivial getters/setters (unless they have logic)
- Third-party library behavior
- Integration behavior (belongs in integration tests)

## Platform-Specific Testing

For platform-specific code (e.g., `_windows.go`, `_unix.go`):

```go
//go:build windows
// +build windows

package mypackage

func TestWindowsSpecificFunction(t *testing.T) {
    // Test Windows-specific behavior
}
```

## Running Tests

### Run all tests
```bash
go test ./...
```

### Run tests for specific package
```bash
go test ./pkg/config
```

### Run with verbose output
```bash
go test -v ./...
```

### Run with coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run specific test
```bash
go test -run TestFunctionName ./pkg/config
```

## Best Practices

1. **Keep tests simple**: Each test should verify one behavior
2. **Use meaningful test names**: Describe what is being tested
3. **Avoid test interdependence**: Tests should not depend on execution order
4. **Clean up resources**: Use `defer` for cleanup operations
5. **Test error paths**: Don't just test happy paths
6. **Use subtests**: Use `t.Run()` for table-driven tests to get better error reporting
7. **Avoid sleeps**: Don't use `time.Sleep()` in tests; use channels or sync primitives
8. **Mock external dependencies**: Use interfaces to enable mocking

## Code Coverage Goals

- Aim for >80% coverage for new code
- Critical packages (crypto, config, validation) should have >90% coverage
- Focus on meaningful coverage, not just line coverage

## Examples

See existing tests for reference:
- `cmd/shared/parsers_test.go` - Table-driven test example

## Documentation Standards (go doc)

All tests should be written for well-documented code. Ensure:
- Package-level comments describing the package purpose
- Exported functions have doc comments starting with the function name
- Doc comments are complete sentences
- Complex logic has explanatory comments
