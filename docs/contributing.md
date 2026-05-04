# Contributing

We welcome contributions to Lucifer! This document outlines the process for contributing to the project.

## Getting Started

1. Fork the repository on GitHub
2. Clone your fork locally
3. Set up your development environment (see [Getting Started](getting-started.md))
4. Create a feature branch for your changes

## Development Workflow

### 1. Choose an Issue

- Check the [GitHub Issues](https://github.com/your-org/lucifer/issues) for open tasks
- Comment on the issue to indicate you're working on it
- For new features, create an issue first to discuss the approach

### 2. Code Development

- Follow Go coding standards and conventions
- Write comprehensive tests for new features
- Update documentation for any user-facing changes
- Ensure all tests pass before submitting

### 3. Testing

Run the test suite:

```bash
# Test all components
go test ./...

# Test specific package
go test ./chaos-engineering/orchestrator

# Run with coverage
go test -cover ./...
```

### 4. Documentation

- Update relevant documentation in `docs/`
- Add code comments for complex logic
- Update CLI help text if commands change

### 5. Pull Request

- Create a pull request with a clear title and description
- Reference the issue number in the PR description
- Ensure CI checks pass
- Request review from maintainers

## Code Standards

### Go Code

- Use `gofmt` to format code
- Follow effective Go guidelines
- Use meaningful variable and function names
- Add comments for exported functions

### Commit Messages

- Use clear, descriptive commit messages
- Start with a verb (Add, Fix, Update, etc.)
- Reference issue numbers when applicable

Example:
```
Fix S3 access deny revert issue (#123)

- Add proper error handling for bucket policy deletion
- Update documentation with permission requirements
- Add test case for permission denied scenario
```

### Branch Naming

- Use descriptive branch names
- Prefix with issue number when applicable

Examples:
- `feature/s3-chaos-improvements`
- `fix/123-s3-permission-issue`
- `docs/update-cli-reference`

## Component-Specific Guidelines

### Orchestrator

- Keep API backward compatible
- Add proper error handling
- Update API documentation for new endpoints

### Agents

- Minimize resource usage
- Handle network interruptions gracefully
- Log errors appropriately

### CLI

- Maintain consistent command structure
- Provide helpful error messages
- Support both interactive and scripted usage

### Security Audit

- Add comprehensive test cases for new rules
- Document rule logic clearly
- Consider performance impact of new scans

## Testing Strategy

### Unit Tests

- Test individual functions and methods
- Mock external dependencies
- Cover error conditions

### Integration Tests

- Test component interactions
- Use test containers for external services
- Validate end-to-end workflows

### Chaos Testing

- Test chaos experiments in isolated environments
- Verify revert functionality
- Monitor for unexpected side effects

## Security Considerations

- Never commit credentials or secrets
- Validate all user inputs
- Follow principle of least privilege
- Report security issues privately

## Documentation

- Keep README files up to date
- Update API documentation for changes
- Add examples for new features
- Maintain changelog

## Community

- Be respectful and inclusive
- Help other contributors
- Participate in code reviews
- Share knowledge and best practices

## Recognition

Contributors are recognized in:
- GitHub contributor statistics
- CHANGELOG entries
- Release notes
- Project documentation

Thank you for contributing to Lucifer! 🎉