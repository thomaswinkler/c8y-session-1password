# c8y-session-1password

[![Test](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/test.yml/badge.svg)](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/test.yml)
[![Release](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/release.yml/badge.svg)](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/thomaswinkler/c8y-session-1password)](https://goreportcard.com/report/github.com/thomaswinkler/c8y-session-1password)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A session provider for [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) that integrates with 1Password for storing and retrieving Cumulocity IoT session credentials.

## Installation

### Using Go Install

```bash
go install github.com/thomaswinkler/c8y-session-1password@latest
```

Or download from [GitHub Releases](https://github.com/thomaswinkler/c8y-session-1password/releases).

## Integration with go-c8y-cli

**Interactive session picker:**
```bash
eval $(c8y sessions login --from-cmd "c8y-session-1password list" --shell auto)
```

**Direct session access:**
```bash
eval $(c8y sessions login --from-cmd "c8y-session-1password --vault Employee --item Production" --shell auto)
```

**Using environment variables:**
```bash
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="Production"
eval $(c8y sessions login --from-cmd "c8y-session-1password" --shell auto)
```

## 1Password Setup

Structure your 1Password login items with:
- **Title**: Session name
- **Username**: Cumulocity username  
- **Password**: Cumulocity password
- **Website**: Cumulocity instance URL
- **Tags**: Add `c8y` tag (required for filtering)
- **Custom Field** (optional): `Tenant` for explicit tenant specification
- **One-Time Password** (optional): TOTP secret for 2FA

## Configuration

### Environment Variables
- `C8YOP_VAULT` - Default vault(s) to search (comma-separated: `"Employee,Shared"`)
- `C8YOP_TAGS` - Filter tags (defaults to `"c8y"`)
- `C8YOP_ITEM` - Default item name or ID

### Command Line Options
```bash
# Interactive picker
c8y-session-1password list --vault "Employee" --tags "c8y,prod"

# Direct access
c8y-session-1password --vault "Employee" --item "Production"
c8y-session-1password --uri "op://Employee/Production"
```

## Shell Integration

### Recommended Aliases

Add these aliases to your shell profile (`~/.zshrc`, `~/.bashrc`, etc.):

```bash
# Quick session login with interactive picker
alias c8y-op='eval $(c8y sessions login --from-cmd "c8y-session-1password list" --shell auto)'

# Add other aliases as needed
alias c8y-xyz-session='eval $(c8y sessions login --from-cmd "c8y-session-1password --vault Shared --item xyz" --shell auto)'
```

### Environment Configuration

For automated environments or when working with specific projects, set these in your shell profile:

```bash
# Default configuration
export C8YOP_VAULT="Employee,Shared"
export C8YOP_TAGS="c8y"

# Project-specific configuration (optional)
export C8YOP_ITEM="MyProject-Dev"
```

### Usage Examples

```bash
# Interactive session selection
c8y-op

# Quick environment switching  
c8y-xyz-session

# One-time vault override
eval $(c8y sessions login --from-cmd "c8y-session-1password list --vault Personal" --shell auto)
```

## Development

**Prerequisites:** Go 1.21+, golangci-lint, 1Password CLI

```bash
make build    # Build binary
make test     # Run tests  
make lint     # Run linting
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
