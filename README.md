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

**Interactive session picker (searches all vaults):**
```bash
eval $(c8y sessions login --from-cmd "c8y-session-1password list --reveal" --shell auto)
```

**Direct session access (searches all vaults for item):**
```bash
eval $(c8y sessions login --from-cmd "c8y-session-1password --item Production --reveal" --shell auto)
```

**Direct session access from specific vault:**
```bash
eval $(c8y sessions login --from-cmd "c8y-session-1password --vault Employee --item Production --reveal" --shell auto)
```

**Using environment variables:**
```bash
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="Production"
eval $(c8y sessions login --from-cmd "c8y-session-1password" --reveal --shell auto)
```

**Searching multiple vaults:**
```bash
export C8YOP_VAULT="Employee,Shared"
eval $(c8y sessions login --from-cmd "c8y-session-1password list --reveal" --shell auto)
```

## 1Password Setup

Structure your 1Password login items with:
- **Title**: Session name
- **Username**: Cumulocity username  
- **Password**: Cumulocity password
- **Website**: Cumulocity instance URL (or use URLs section for multiple environments)
- **Tags**: Add `c8y` tag (required for filtering)
- **Custom Field** (optional): `Tenant` for explicit tenant specification
- **One-Time Password** (optional): TOTP secret for 2FA

Keep item names short and consistent. Same for tags.

### Multiple URLs Support

Items with multiple URLs in the 1Password URLs section will create separate sessions for each URL:
- **Primary URLs** are listed first in the picker
- **URL labels** are used to distinguish between environments (e.g., "Production", "Staging")
- **Fallback behavior**: If no URLs section exists, falls back to Website/URL fields

## Usage

### Environment Variables
- `C8YOP_VAULT` - Default vault(s) to search (comma-separated: `"Employee,Shared"`, optional - if not provided, searches all vaults)
- `C8YOP_TAGS` - Filter tags (defaults to `"c8y"`)
- `C8YOP_ITEM` - Default item name or ID
- `C8YOP_LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`; defaults to `info`)
- `LOG_LEVEL` - Alternative to `C8YOP_LOG_LEVEL` (C8YOP_LOG_LEVEL takes precedence)

### Command Line Options
```bash
# Interactive picker from all vaults (passwords obfuscated by default)
c8y-session-1password list

# Interactive picker from specific vault(s) (passwords obfuscated by default)
c8y-session-1password list --vault "Employee" --tags "c8y,prod"

# Interactive picker with revealed passwords
c8y-session-1password list --vault "Employee" --tags "c8y,prod" --reveal

# Direct access from all vaults (vault is optional)
c8y-session-1password --item "Production"

# Direct access from specific vault (passwords obfuscated by default)
c8y-session-1password --vault "Employee" --item "Production"
c8y-session-1password --uri "op://Employee/Production"

# Direct access with revealed passwords
c8y-session-1password --vault "Employee" --item "Production" --reveal
c8y-session-1password --uri "op://Employee/Production" --reveal
```

### Security Features

By default, both commands obfuscate sensitive information (passwords, TOTP secrets) to prevent accidental exposure:

- **`list` command**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values
- **Root command (direct access)**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values
- **Interactive mode from root**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values

This approach prioritizes security by requiring explicit use of `--reveal` when you need to see sensitive credentials.

### Debugging and Logging

Enable debug logging to troubleshoot 1Password integration issues:

```bash
# Enable debug logging (recommended - consistent with other C8YOP_ variables)
export C8YOP_LOG_LEVEL=debug
c8y-session-1password list

# Alternative using LOG_LEVEL
export LOG_LEVEL=debug
c8y-session-1password list

# Or inline
C8YOP_LOG_LEVEL=debug c8y-session-1password --item "Production"
LOG_LEVEL=debug c8y-session-1password --item "Production"
```

Available log levels:
- `debug` - Detailed operational information (fetching items, API calls)
- `info` - General information (default)
- `warn`, `warning` - Warning messages
- `error` - Error messages only

Debug logging is particularly useful for:
- Troubleshooting 1Password CLI connectivity
- Understanding which vaults and items are being searched
- Performance analysis of bulk vs individual item fetching

## Environment Configuration

For automated environments or when working with specific projects, set these in your shell profile:

```bash
# Default configuration (searches specific vaults)
export C8YOP_VAULT="Employee,Shared"
export C8YOP_TAGS="c8y"

# Alternative: search all vaults (omit C8YOP_VAULT)
export C8YOP_TAGS="c8y"

# Project-specific configuration (optional)
export C8YOP_ITEM="MyProject-Dev"
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
