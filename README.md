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
eval $(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto)
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
eval $(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto)
```

**Searching multiple vaults:**
```bash
export C8YOP_VAULT="Employee,Shared"
eval $(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto)
```

**Add to your shell profile:**
```bash
# Add to ~/.bashrc or ~/.zshrc
set-session-op() {
  eval $(c8y sessions login --from-cmd "c8y-session-1password --reveal $*" --shell auto)
} 

# Advanced: Get URI for a session (useful for direct 1Password CLI commands)
get-session-uri() {
  c8y-session-1password --output uri $*
}

# Remove op alias created by go-c8y-cli to avoid conflicts with 1Password CLI
unalias op
```

Then you can use both functions:
```bash
set-session-op xyz           # Set session in go-c8y-cli
get-session-uri xyz          # Get op:// URI for the session

# Use URI with 1Password CLI directly
op item get $(get-session-uri Production) --fields label=password
```

</details>

<details>
<summary><strong>PowerShell</strong></summary>

**Interactive session picker (searches all vaults):**
```powershell
$(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto) | Invoke-Expression
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
$(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto) | Invoke-Expression
```

</details>

## Output Formats

Control the output format using the `--output` or `-o` flag. This is useful for integrating with other tools or scripts that expect specific formats.

```bash
# Default JSON output (for go-c8y-cli integration)
c8y-session-1password --output json

# URI output (op:// uri format)
c8y-session-1password --output uri
```

**Available output formats:**
- `json` (default) - Full session JSON for go-c8y-cli integration
- `uri` - Only the `op://vault/item` URI for direct 1Password CLI usage

### Output Format Examples

#### JSON output (default)

The default output format is JSON, which includes all session details as required by `go-c8y-cli` [sessions login](https://goc8ycli.netlify.app/docs/cli/c8y/sessions/c8y_sessions_login/) command.

```json
{
  "host": "https://myinstance.cumulocity.com",
  "username": "myuser",
  "password": "***",
  "tenant": "t123456",
  "itemId": "abc123",
  "vaultName": "Employee"
}
```

#### URI output

```
op://Employee/abc123?target_url=https%3A%2F%2Fmyinstance.cumulocity.com
```

Use for example to store the selected session URI in an environment variable for further use in scripts or commands. The `target_url` parameter is optional and holds the selected url to use for the session. This is required for 1Password items with multiple URLs.

Please note that `target_url` is NOT supported by 1Password CLI `op` command, but can be used in scripts or other tools that support this format.

<details open>
<summary><strong>bash / zsh</strong></summary>

```bash
# Store URI in environment variable
export C8Y_SESSION=$(c8y-session-1password --output uri)
echo "Selected session: $C8Y_SESSION"
```
</details>

<details>
<summary><strong>PowerShell</strong></summary>

```powershell
# Store URI in environment variable
$env:C8Y_SESSION = c8y-session-1password --output uri
Write-Host "Selected session: $env:C8Y_SESSION"
```
</details>

## 1Password Setup

### Enable 1Password CLI Access

> [!IMPORTANT] 
> Before using the `c8y-session-1password`, you must enable [1Password CLI](https://developer.1password.com/docs/cli/) in the 1Password desktop app:
> 1. Open the 1Password desktop app or [install](https://1password.com/downloads/) 
> 2. Go to **Settings** → **Developer** (or **Preferences** → **Advanced** on older versions)
> 3. Enable **"Integrate with 1Password CLI"**
> 4. Restart the 1Password app if prompted

> [!NOTE]
> If you can not run `op` command, make sure the 1Password CLI is installed and available in your PATH. You can download it from [1Password CLI](https://developer.1password.com/docs/cli/get-started/). You might also need to run `unalias op` to remove any existing aliases that might conflict with the 1Password CLI command.

### Authentication

The 1Password CLI supports multiple authentication methods:

#### Interactive Authentication (Desktop/Personal Use)

For personal use with the 1Password desktop app:

```bash
op signin
```

Signin requires the 1Password desktop app to be running and CLI integration enabled.

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

## 1Password Items

Create 1Password **Login** items with these exact field configurations:

### Required Fields

The following fields are required for the `c8y-session-1password` to function correctly:

- `title`: Item and session name (used for display and filtering)
- `username`: Cumulocity username
- `password`: Cumulocity password
- `tags`: Must include `c8y` tag (case-insensitive, required for discovery)

You can choose any tags. Make sure to pass using `--tags` flag or set `C8YOP_TAGS` environment variable to configure the tags to be used for search.

### URL Configuration and Multi Session Support

There is flexibility in how you configure URLs for Cumulocity instances. If there is more than one URL found in an item, a separate session will be created and shown for each URL.

URLs should be absolute URLs pointing to the Cumulocity instance, e.g. `https://myinstance.cumulocity.com`. The url will be used to identify the session in the picker. The label of the field, also if customized, is not used for the session or it's presentation in the picker.

The following fields will be searched for URLs:

- Create one or more `website` fields
- Create custom fields with **Type**: `URL`
- Create custom fields with **Label**: `website` or **Label**: `url`

**URL detection**: Searches websites first, then falls back to custom fields with:
- Field **Type** = `URL` (case-sensitive)
- Field **Label** = `website` or `url` (case-insensitive)

### Optional Fields

Currently, only `tenant` field is supported as an optional field. If configured in the 1Password item, it's value will be passed as tenant id to `go-c8y-cli`. As `go-c8y-cli` determines the tenant id automatically, this is only needed if you really want to override the tenant id.

### Summary and Troubleshooting

When creating 1Password items, ensure:
- ✅ **Category** is `Login` (not Secure Note or other types)
- ✅ **Tags** include `c8y` or pass tags using `--tags` or set `C8YOP_TAGS` env variable
- ✅ **Username** and **Password** fields are populated
- ✅ At least one URL is configured (`website` field, or URL custom fields)
- ✅ **Title** is descriptive for easy identification in the picker

Make sure item names and tags are short and descriptive to make it easy to find the right session in the interactive picker.

**Common Issues:**
- ❌ Item not appearing: Check category is "Login" and has `c8y` tag as well as at least one URL
- ❌ Multiple sessions not created: Check URLs section or multiple URL custom fields
- ❌ Make sure the item is in the vaults you are searching and you have access to the vault
- ❌ If using `--item`, ensure the item name matches exactly (case-sensitive)

Enable DEBUG logging to see detailed output and troubleshoot issues with fetching items or sessions

## Security

By default, all sensitive information (passwords, TOTP secrets) is obfuscated in the output to prevent accidental exposure:

- Passwords, TOTP codes, and TOTP secrets show as `***` by default
- Use `--reveal` flag to show actual values when needed

This approach prioritizes security by requiring explicit use of `--reveal` when you need to see sensitive credentials.

## Environment Configuration

The following environment variables can be set to customize the behavior of `c8y-session-1password`:

- `C8YOP_VAULT` - Default vault(s) to search (comma-separated: `"Employee,Shared"`, optional - if not provided, searches all vaults)
- `C8YOP_TAGS` - Filter tags (defaults to `"c8y"`)
- `C8YOP_ITEM` - Default item name or ID
- `C8YOP_LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`; defaults to `info`)

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
# Enable debug logging
export C8YOP_LOG_LEVEL=debug
c8y-session-1password

# Alternative using LOG_LEVEL
export LOG_LEVEL=debug
c8y-session-1password

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
