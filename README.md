# c8y-session-1password

[![Test](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/test.yml/badge.svg)](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/test.yml)
[![Release](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/release.yml/badge.svg)](https://github.com/thomaswinkler/c8y-session-1password/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/thomaswinkler/c8y-session-1password)](https://goreportcard.com/report/github.com/thomaswinkler/c8y-session-1password)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A session provider for [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) that integrates with 1Password for storing and retrieving Cumulocity session credentials.

## Installation

go-c8y-cli can get the credentials for a login from another program, a session provider. `c8y-session-1password` is such a program: it reads your sessions from 1Password, lets you select one, and passes it to go-c8y-cli.

You can install it in two ways:

- **As go-c8y-cli extension:** go-c8y-cli installs and updates it for you. You select a session with `set-session --from-extension onepassword`. Requires go-c8y-cli 2.57.0 or later.
- **As standalone binary:** you install and update the binary yourself, and configure go-c8y-cli to run it. Use it with older go-c8y-cli versions, or to run `c8y-session-1password` on its own, e.g. in scripts.

Both select sessions the same way. In both cases, the [1Password CLI](#1password-setup) must be installed.

### As go-c8y-cli extension (recommended)

```bash
c8y extensions install thomaswinkler/c8y-session-1password
```

Update it with `c8y extensions update session-1password`.

### Standalone binary

```bash
go install github.com/thomaswinkler/c8y-session-1password@latest
```

Or download an archive for your platform from [GitHub Releases](https://github.com/thomaswinkler/c8y-session-1password/releases). The binary is in the `bin` folder of the archive. Copy it to a folder in your `PATH`.

## Usage with go-c8y-cli

### With the extension

```bash
# Select a session from 1Password
set-session --from-extension onepassword

# Pass arguments to c8y-session-1password after --
set-session --from-extension onepassword -- --vault Employee Production

# Use the provider by default for set-session
c8y settings update session.provider.type extension
c8y settings update session.provider.extension onepassword
set-session
```

`set-session` is part of the go-c8y-cli shell integration (`c8y cli install`). The provider uses your environment, for example `C8YOP_VAULT` and `C8YOP_TAGS`.

### Without the extension

Use the standalone binary as external session provider, e.g. with go-c8y-cli versions before 2.57.0:

```bash
# Use the provider by default for set-session
c8y settings update session.provider.type external
c8y settings update session.provider.command "c8y-session-1password --reveal"
set-session

# Or log in once
eval "$(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto)"
```

```powershell
$(c8y sessions login --from-cmd "c8y-session-1password --reveal" --shell auto) | Invoke-Expression
```

`--reveal` is required, as go-c8y-cli needs the password (and TOTP code) to log in.

### Selecting a session

Without arguments, an interactive picker shows all sessions. A filter argument selects the sessions to show:

```bash
set-session --from-extension onepassword -- production
c8y-session-1password --reveal production
```

The filter matches the session name, item name, URL, and username (case-insensitive, also parts), and the item ID (exact match). If exactly one session matches, it's selected without the picker.

To select an item directly, use `--item` with the item name or ID, or `--uri` with an `op://` URI (see [URI output](#uri-output)). A URI can also be given instead of the filter, e.g. `c8y-session-1password --reveal 'op://vault123/abc123'`. `--vault` limits the search to one or more vaults.

### Use with the c8y MCP server

With go-c8y-cli 2.58.0 or later, the session picker of `c8y mcp serve` shows the sessions from 1Password, grouped by vault. When the token expires, the server gets the session from 1Password again, including a new TOTP code.

If `onepassword` is your default provider (see [With the extension](#with-the-extension)), the picker shows your session files and the sessions from 1Password, e.g. for Claude Code:

```bash
claude mcp add --scope user c8y -- c8y mcp serve
```

Use `--sessionProvider` to choose the sources and their order. `file` stands for your session files, and arguments after the provider name select the sessions from 1Password:

```bash
# only sessions from 1Password
claude mcp add --scope user c8y -- c8y mcp serve --sessionProvider onepassword

# sessions from the Employee vault, then the session files
claude mcp add --scope user c8y -- c8y mcp serve --sessionProvider "onepassword --vault Employee,file"
```

Check which sessions the picker shows with `c8y sessions list --from-extension onepassword`.

By default, the agent can only read data (session mode `prod`). To allow it to create and update (`qual`), or also delete data (`dev`), set the mode of the session with a `mode` field in the 1Password item (see [Optional Fields](#optional-fields)), or with `--mode` for all sessions without the field:

```bash
claude mcp add --scope user c8y-dev -- c8y mcp serve --sessionProvider "onepassword --mode dev --vault Development"
```

`c8y mcp serve --mode` limits the mode of all sessions, e.g. `--mode qual` doesn't allow deleting data in a `dev` session.

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
- `uri` - The `op://vault/item` URI with the session URL as `target_url` parameter

### Output Format Examples

#### JSON output (default)

The default output format is JSON, which includes all session details as required by `go-c8y-cli` [sessions login](https://goc8ycli.netlify.app/docs/cli/c8y/sessions/c8y_sessions_login/) command.

```json
{
  "sessionUri": "op://vault123/abc123",
  "name": "Production",
  "host": "https://myinstance.cumulocity.com",
  "username": "myuser",
  "password": "***",
  "tenant": "t123456",
  "itemId": "abc123",
  "itemName": "Production",
  "vaultId": "vault123",
  "vaultName": "Employee",
  "tags": [
    "c8y"
  ]
}
```

#### URI output

```
op://vault123/abc123?target_url=https%3A%2F%2Fmyinstance.cumulocity.com
```

Use for example to store the selected session URI in an environment variable for further use in scripts or commands. The `target_url` parameter is optional and holds the selected url to use for the session. This is required for 1Password items with multiple URLs.

Please note that `target_url` is NOT supported by 1Password CLI `op` command, but can be used in scripts or other tools that support this format. `c8y-session-1password --uri` accepts it to select the session of that URL. Quote the URI in the shell, as `?` is a wildcard in zsh:

```bash
c8y-session-1password --reveal --uri 'op://vault123/abc123?target_url=https%3A%2F%2Fmyinstance.cumulocity.com'
```

<details open>
<summary><strong>bash / zsh</strong></summary>

```bash
# Store URI in environment variable
export C8Y_SESSION=$(c8y-session-1password --output uri)
echo "Selected session: $C8Y_SESSION"

# Use the URI with the 1Password CLI. Remove the target_url parameter, which op doesn't support
op item get "${C8Y_SESSION%%\?*}" --fields label=password
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

### List sessions

`--list` prints all matching sessions instead of selecting one, without the interactive picker. Other tools can use it to show the sessions in their own session picker. The output never includes passwords or TOTP secrets, so `--list` can't be used with `--reveal`.

```bash
# JSON array of all matching sessions (an empty array if none match)
c8y-session-1password --list --vault Employee

# One URI per line, which selects the session with --uri
c8y-session-1password --list --output uri production
```

## 1Password Setup

### Enable 1Password CLI Access

> [!IMPORTANT] 
> Before using the `c8y-session-1password`, you must enable [1Password CLI](https://developer.1password.com/docs/cli/) in the 1Password desktop app:
> 1. Open the 1Password desktop app or [install](https://1password.com/downloads/) 
> 2. Go to **Settings** → **Developer** (or **Preferences** → **Advanced** on older versions)
> 3. Enable **"Integrate with 1Password CLI"**
> 4. Restart the 1Password app if prompted

> [!NOTE]
> If you cannot run the `op` command, make sure the 1Password CLI is installed and available in your PATH. You can download it from [1Password CLI](https://developer.1password.com/docs/cli/get-started/). You might also need to run `unalias op` to remove any existing aliases that might conflict with the 1Password CLI command.

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

Items need the following for `c8y-session-1password` to find them and create sessions:

- **Category**: `Login`
- **Title**: Item and session name (used for display and filtering)
- **Tags**: Must include the `c8y` tag (case-insensitive, required for discovery)
- `username`: Cumulocity username
- `password`: Cumulocity password
- At least one URL (see below)

You can choose any tags. Make sure to pass using `--tags` flag or set `C8YOP_TAGS` environment variable to configure the tags to be used for search.

### URL Configuration and Multi Session Support

There is flexibility in how you configure URLs for Cumulocity instances. If there is more than one URL found in an item, a separate session will be created and shown for each URL.

The `sessionUri` of these sessions includes the URL as `target_url` parameter, e.g. `op://Employee/abc123?target_url=https%3A%2F%2Fmyinstance.cumulocity.com`, so it identifies the session and can be used with `--uri`. Sessions of items with one URL use `op://vault/item`, which the 1Password CLI also supports.

URLs should be absolute URLs pointing to the Cumulocity instance, e.g. `https://myinstance.cumulocity.com`. The url will be used to identify the session in the picker. A path, query, or fragment is removed, so a URL copied from the browser, e.g. `https://myinstance.cumulocity.com/apps/cockpit/index.html#/`, also works. URLs of the same instance in one item result in one session. The label of the field, also if customized, is not used for the session or its presentation in the picker.

The following fields will be searched for URLs:

- Create one or more `website` fields
- Create custom fields with **Type**: `URL`
- Create custom fields with **Label**: `website` or **Label**: `url`

**URL detection**: Searches websites first, then falls back to custom fields with:
- Field **Type** = `URL` (case-sensitive)
- Field **Label** = `website` or `url` (case-insensitive)

### Optional Fields

- `tenant`: If configured in the 1Password item, its value will be passed as tenant id to `go-c8y-cli`. As `go-c8y-cli` determines the tenant id automatically, this is only needed if you really want to override the tenant id.
- `mode`: The session mode of `go-c8y-cli`, `prod` (read only), `qual` (also create and update) or `dev` (also delete). It applies to all URLs of the item, and takes precedence over `--mode`. Invalid values are ignored with a warning.
- One-time password: For tenants with two-factor authentication (TOTP), add a one-time password field to the item. With `--reveal`, the output includes the current code (`totp`) and the secret (`totpSecret`), and `go-c8y-cli` uses the code to log in.

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
- ❌ Multiple sessions not created: Check URLs section or multiple URL custom fields. URLs of the same instance result in one session
- ❌ Make sure the item is in the vaults you are searching and you have access to the vault
- ❌ If using `--item`, ensure the item name matches exactly (case-sensitive)

Enable DEBUG logging to see detailed output and troubleshoot issues with fetching items or sessions

## Security

By default, all sensitive information (passwords, TOTP secrets) is obfuscated in the output to prevent accidental exposure:

- Passwords, TOTP codes, and TOTP secrets show as `***` by default
- Use `--reveal` flag to show actual values when needed
- `--list` never outputs passwords, TOTP codes, or TOTP secrets, also not with `--reveal`

This approach prioritizes security by requiring explicit use of `--reveal` when you need to see sensitive credentials.

## Environment Configuration

The following environment variables can be set to customize the behavior of `c8y-session-1password`:

- `C8YOP_VAULT` - Default vault(s) to search (comma-separated: `"Employee,Shared"`, optional - if not provided, searches all vaults)
- `C8YOP_TAGS` - Filter tags (defaults to `"c8y"`)
- `C8YOP_ITEM` - Default item name or ID
- `C8YOP_LOG_LEVEL` - Logging level (`debug`, `info`, `warn`, `error`; defaults to `warn`)

Flags take precedence over the environment variables. `--uri` takes precedence over `--vault`, `--item`, `C8YOP_VAULT`, and `C8YOP_ITEM`, as the URI identifies exactly one session.

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
# Flags: --debug (debug level) or -v/--verbose (info level)
c8y-session-1password --debug

# go-c8y-cli passes -v and --debug on to the session provider
set-session --from-extension onepassword --debug

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
- `info` - General information
- `warn`, `warning` - Warning messages (default)
- `error` - Error messages only

Debug logging is particularly useful for:
- Troubleshooting 1Password CLI connectivity
- Understanding which vaults and items are being searched
- Performance analysis of bulk vs individual item fetching

## Chrome Extension

`c8y-session-1password` can also be used as native messaging host of a Chrome extension. See [Chrome Native Messaging](docs/chrome-native-messaging.md).

## Development

**Prerequisites:** Go 1.21+, golangci-lint, 1Password CLI

```bash
make build                # Build binary into bin/
make test                 # Run tests
make lint                 # Run linting
make package              # Build the release archives into dist/
make validate-extension   # Validate the go-c8y-cli extension (needs c8y 2.57.0 or later)
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
