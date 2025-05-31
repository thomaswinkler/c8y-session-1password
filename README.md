# c8y-session-1password

A session provider for [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) that integrates with 1Password for storing and retrieving Cumulocity IoT session credentials.

## Features

- Interactive session picker with search and filtering
- Integration with 1Password CLI
- Support for TOTP/2FA codes
- Vault and tag-based filtering
- Environment variable configuration
- Compatible with `c8y sessions login --from-cmd` workflow

## Prerequisites

- [1Password CLI (op)](https://developer.1password.com/docs/cli/) installed and configured
- [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) installed
- Active 1Password session (run `op signin` first)

## Installation

```bash
go install github.com/thomaswinkler/c8y-session-1password@latest
```

Or build from source:

```bash
git clone https://github.com/thomaswinkler/c8y-session-1password.git
cd c8y-session-1password
go build -o c8y-session-1password .
```

## Usage

### Basic Usage

Interactive session selection:

```bash
c8y-session-1password list
```

Direct item retrieval:

```bash
# Using vault and item flags
c8y-session-1password --vault "Employee" --item "Production"

# Using op:// URI
c8y-session-1password --uri "op://Employee/Production"

# Using environment variables
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="Production"
c8y-session-1password
```

### Integration with go-c8y-cli

Use with `c8y sessions login` for seamless session switching:

```bash
# Interactive selection
eval $(c8y sessions login --from-cmd "c8y-session-1password list" --shell auto)

# Direct item retrieval
eval $(c8y sessions login --from-cmd "c8y-session-1password --vault Employee --item Production" --shell auto)
```

### Command Line Options

#### Root Command
For direct item retrieval:
```bash
c8y-session-1password --help
```

Available flags:
- `--vault` - Vault name or ID to search in (supports multiple vaults: `"Employee,Shared"`)
- `--item` - Specific item ID or name to retrieve
- `--uri` - op://vault/item URI to retrieve specific item

#### List Command
For interactive session selection:
```bash
c8y-session-1password list --help
```

Available flags:
- `--vault` - Vault name or ID to search in (supports multiple vaults: `"Employee,Shared"`)
- `--tags` - Comma-separated tags to filter by

### Environment Variables

Configure default values using environment variables:

- `C8YOP_VAULT` - Default vault to search in (can be vault name, ID, or comma-separated list: `"Employee,Shared"`)
- `C8YOP_TAGS` - Default tags to filter by (comma-separated)
- `C8YOP_ITEM` - Default item to retrieve (item ID or name)

**Compatibility Note:** For backward compatibility, `CYOP_*` variants are also supported as fallbacks:
- `CYOP_VAULT` - Fallback for C8YOP_VAULT
- `CYOP_TAGS` - Fallback for C8YOP_TAGS
- `CYOP_ITEM` - Fallback for C8YOP_ITEM

Example:

```bash
export C8YOP_VAULT="Development"
export C8YOP_TAGS="c8y,staging"
export C8YOP_ITEM="My Cumulocity Session"
c8y-session-1password list
```

Or using the compatibility format:

```bash
export CYOP_VAULT="Development"
export CYOP_TAGS="c8y,staging"
export CYOP_ITEM="My Cumulocity Session"
c8y-session-1password
```

### Direct Item Access

You can retrieve a specific item directly without the interactive picker using several methods:

#### Method 1: Using vault and item flags
```bash
c8y-session-1password --vault "Employee" --item "Cumulocity Production"
```

#### Method 2: Using op:// URI
```bash
c8y-session-1password --uri "op://Employee/Cumulocity Production"
```

#### Method 3: Using environment variables
```bash
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="Cumulocity Production"
c8y-session-1password
```

All arguments are optional if the required environment variables are set, making it easy to configure in CI/CD environments.

## Session URI Format

The tool generates session URIs in the format:
```
op://VaultName/ItemName
```

- **VaultName**: The name of the 1Password vault containing the item
- **ItemName**: The title/name of the 1Password item

This format makes the URIs human-readable while internally using item UUIDs for reliable access.

### Example Output
```json
{
  "sessionUri": "op://Employee/Cumulocity Production",
  "name": "Cumulocity Production", 
  "host": "https://mycompany.cumulocity.com",
  "username": "john.doe",
  "tenant": "mycompany",
  "totp": "123456",
  "itemId": "abc123def456hij789klm012nop345qr",
  "itemName": "Cumulocity Production",
  "vaultId": "vault123",
  "vaultName": "Employee",
  "tags": ["c8y", "production"]
}
```

## Multi-Vault Support

The tool supports searching across multiple 1Password vaults by providing comma-separated vault names. This is particularly useful when you have credentials stored across different vaults (e.g., Employee and Shared vaults) and want to avoid specifying different vault names when switching between sessions.

### How it works

When multiple vaults are specified:
- **For listing**: Items from all specified vaults are included in the results
- **For direct item access**: Vaults are searched in the order specified until the item is found

### Usage Examples

#### Interactive selection across multiple vaults
```bash
c8y-session-1password list --vault "Employee,Shared"
```

#### Direct item access with vault fallback
```bash
# Will search Employee first, then Shared if not found
c8y-session-1password --vault "Employee,Shared" --item "Production Environment"
```

#### Environment variable configuration
```bash
export C8YOP_VAULT="Employee,Shared,Personal"
export C8YOP_ITEM="My Session"
c8y-session-1password  # Searches vaults in order until found
```

This feature eliminates the need to remember which vault contains which credentials, providing a seamless experience across your 1Password organization.

## 1Password Item Structure

For optimal compatibility, structure your 1Password login items as follows:

### Required Fields
- **Title**: Descriptive name for the session
- **Username**: Your Cumulocity username (can include tenant as `tenant/username`)
- **Password**: Your Cumulocity password
- **Website**: URL of your Cumulocity instance

### Optional Fields
- **Tenant**: Custom field for explicit tenant specification
- **One-Time Password**: TOTP secret for 2FA

### Tags
- Add the `c8y` tag (or your preferred tag) to identify Cumulocity sessions
- Use additional tags for environment categorization (e.g., `production`, `staging`)

### Example Item Structure
```
Title: My C8Y Instance
Username: myuser@company.com
Password: ••••••••
Website: https://mycompany.cumulocity.com
Tags: c8y, production
Custom Fields:
  - Tenant: mycompany
```

## Examples

### Basic interactive session selection
```bash
c8y-session-1password list
```

### Filter by specific vault
```bash
c8y-session-1password list --vault "Work Credentials"
```

### Filter by multiple vaults
```bash
c8y-session-1password list --vault "Employee,Shared"
```

### Filter by multiple tags
```bash
c8y-session-1password list --tags "c8y,production"
```

### Get specific item by ID or name
```bash
c8y-session-1password --item "Cumulocity Production"
c8y-session-1password --item "abc123def456hij789klm012nop345qr"
```

### Get specific item using op:// URI
```bash
c8y-session-1password --uri "op://Employee/Cumulocity Production"
```

### Get item from multiple vaults (searches in order)
```bash
c8y-session-1password --vault "Employee,Shared" --item "Cumulocity Production"
```

### Using environment variables for automation
```bash
# Set up environment for CI/CD
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="Cumulocity Production"

# Now you can call without any arguments
c8y-session-1password
```

### Using multiple vaults for fallback
```bash
# Set up multiple vaults to search in order
export C8YOP_VAULT="Employee,Shared,Personal"
export C8YOP_ITEM="Cumulocity Production"

# Will search Employee first, then Shared, then Personal
c8y-session-1password
```

### Integration with go-c8y-cli using environment variables
```bash
# Set up your session details
export C8YOP_VAULT="Employee"
export C8YOP_ITEM="My Production Environment"

# Use with c8y sessions login
eval $(c8y sessions login --from-cmd "c8y-session-1password" --shell auto)
```

### Shell alias for quick access
Add to your shell profile:
```bash
alias c8y-login='eval $(c8y sessions login --from-cmd "c8y-session-1password list" --shell auto)'
alias c8y-prod='eval $(c8y sessions login --from-cmd "c8y-session-1password --uri \"op://Employee/Production Environment\"" --shell auto)'
```

Then simply run:
```bash
c8y-login    # Interactive picker
c8y-prod     # Direct to production environment
```

## Development

### Building
```bash
go build -o c8y-session-1password .
```

### Testing
```bash
go test ./...
```

## Related Projects

- [go-c8y-cli](https://github.com/reubenmiller/go-c8y-cli) - Cumulocity IoT CLI tool
- [c8y-session-bitwarden](https://github.com/reubenmiller/c8y-session-bitwarden) - Bitwarden integration

## License

MIT License - see [LICENSE](LICENSE) file for details.
