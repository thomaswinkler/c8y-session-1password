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

Setup aliases in your shell profile to use `c8y-session-1password` with `go-c8y-cli`:

<details open>
<summary><strong>bash / zsh</strong></summary>

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

</details>

<details>
<summary><strong>PowerShell</strong></summary>

**Interactive session picker (searches all vaults):**
```powershell
$(c8y sessions login --from-cmd "c8y-session-1password list --reveal" --shell auto) | Invoke-Expression
```

**Direct session access (searches all vaults for item):**
```powershell
$(c8y sessions login --from-cmd "c8y-session-1password --item Production --reveal" --shell auto) | Invoke-Expression
```

**Direct session access from specific vault:**
```powershell
$(c8y sessions login --from-cmd "c8y-session-1password --vault Employee --item Production --reveal" --shell auto) | Invoke-Expression
```

**Using environment variables:**
```powershell
$env:C8YOP_VAULT="Employee"
$env:C8YOP_ITEM="Production"
$(c8y sessions login --from-cmd "c8y-session-1password" --reveal --shell auto) | Invoke-Expression
```

**Searching multiple vaults:**
```powershell
$env:C8YOP_VAULT="Employee,Shared"
$(c8y sessions login --from-cmd "c8y-session-1password list --reveal" --shell auto) | Invoke-Expression
```

</details>

## 1Password Setup

### Enable 1Password CLI Access

**Important:** Before using the `c8y-session-1password`, you must enable [1Password CLI](https://developer.1password.com/docs/cli/) in the 1Password desktop application:
1. Open the 1Password desktop app or [install](https://1password.com/downloads/) 
2. Go to **Settings** → **Developer** (or **Preferences** → **Advanced** on older versions)
3. Enable **"Integrate with 1Password CLI"**
4. Restart the 1Password app if prompted

### Authentication

The 1Password CLI supports multiple authentication methods:

#### Interactive Authentication (Desktop/Personal Use)

For personal use with the 1Password desktop app:

```bash
op signin
```

Signin requires the 1Password desktop application to be running and CLI integration enabled.

#### Service Account Authentication (Automated/CI/CD)

For automated environments using 1Password Service Accounts:

<details open>
<summary><strong>bash/zsh</strong></summary>

```bash
export OP_SERVICE_ACCOUNT_TOKEN="your-service-account-token"
```

</details>

<details>
<summary><strong>PowerShell</strong></summary>

```powershell
$env:OP_SERVICE_ACCOUNT_TOKEN="your-service-account-token"
```

</details>

No desktop app required. Ideal for CI/CD pipelines and server environments.

#### 1Password Connect (Self-Hosted)

For organizations using 1Password Connect Server:

<details>
<summary><strong>bash/zsh</strong></summary>

```bash
export OP_CONNECT_HOST="https://your-connect-server"
export OP_CONNECT_TOKEN="your-connect-token"
```

</details>

<details>
<summary><strong>PowerShell</strong></summary>

```powershell
$env:OP_CONNECT_HOST="https://your-connect-server"
$env:OP_CONNECT_TOKEN="your-connect-token"
```

</details>

### Item Structure

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

Items with multiple URLs in the 1Password URLs section will create separate sessions in the picker, one for each URL:
- **Primary URLs** are listed first in the picker
- **URL labels** are used to distinguish between environments (e.g., "Production", "Staging")
- **Fallback behavior**: If no URLs section exists, falls back to Website/URL fields

## Security

By default, both commands obfuscate sensitive information (passwords, TOTP secrets) to prevent accidental exposure:

- **`list` command**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values
- **Root command (direct access)**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values
- **Interactive mode from root**: Obfuscates sensitive information by default (shows `***`)
  - Use `--reveal` to show actual values

This approach prioritizes security by requiring explicit use of `--reveal` when you need to see sensitive credentials.

## Environment Configuration

For automated environments or when working with specific projects, set these in your shell profile:

<details open>
<summary><strong>bash / zsh</strong></summary>

```bash
# Default configuration (searches specific vaults)
export C8YOP_VAULT="Employee,Shared"
export C8YOP_TAGS="c8y"

# Alternative: search all vaults (omit C8YOP_VAULT)
export C8YOP_TAGS="c8y"

# Project-specific configuration (optional)
export C8YOP_ITEM="MyProject-Dev"
```

</details>

<details>
<summary><strong>PowerShell</strong></summary>

```powershell
# Default configuration (searches specific vaults)
$env:C8YOP_VAULT="Employee,Shared"
$env:C8YOP_TAGS="c8y"

# Alternative: search all vaults (omit C8YOP_VAULT)
$env:C8YOP_TAGS="c8y"

# Project-specific configuration (optional)
$env:C8YOP_ITEM="MyProject-Dev"
```

For persistent environment variables in PowerShell, add to your PowerShell profile:
```powershell
# Add to $PROFILE (run `notepad $PROFILE` to edit)
$env:C8YOP_VAULT="Employee,Shared"
$env:C8YOP_TAGS="c8y"
```

</details>

## Debugging and Logging

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

## Development

**Prerequisites:** Go 1.21+, golangci-lint, 1Password CLI

```bash
make build    # Build binary
make test     # Run tests  
make lint     # Run linting
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
