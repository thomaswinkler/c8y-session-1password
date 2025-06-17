# Chrome Extension Native Messaging Support

The `c8y-session-1password` tool now supports Chrome Extension Native Messaging protocol, allowing it to act as a native messaging host for Chrome extensions.

## Features

### Native Messaging Mode
- **Automatic Detection**: When input is piped via stdin, the tool automatically switches to native messaging mode
- **Chrome Protocol**: Implements the full Chrome Native Messaging protocol with 4-byte length prefixes
- **Persistent Connection**: Maintains a persistent connection with message loop for multiple requests
- **Proper Logging**: All logs go to stderr, keeping stdout clean for Chrome communication

### Message Types

#### 1. Authentication Test
```json
{"type": "test_auth"}
```
Response:
```json
{
  "type": "auth_result",
  "success": true
}
```
Or on failure:
```json
{
  "type": "auth_result", 
  "success": false,
  "error": "error message"
}
```

#### 2. Session Query
```json
{
  "vaults": ["vault1", "vault2"],
  "tags": ["c8y", "prod"],
  "search": "filter-text",
  "reveal": true
}
```

**Field Descriptions:**
- `vaults` (optional): Array of vault names/IDs to search. If empty, searches all vaults.
- `tags` (optional): Array of tags to filter by. Defaults to ["c8y"] if not specified.
- `search` (optional): Search term to filter sessions by name, item name, URL, or username.
- `reveal` (optional): Boolean flag to reveal sensitive information. Defaults to `false`.

**Security Note:** By default, passwords and TOTP secrets are obfuscated as `"***"`. Set `reveal: true` to get actual values.
Response (single session):
```json
{
  "sessionUri": "op://vault/item",
  "name": "Session Name",
  "host": "https://example.com",
  "username": "user@example.com",
  "password": "secret",
  "tenant": "t123456",
  "itemId": "item-id",
  "itemName": "Item Name",
  "vaultId": "vault-id", 
  "vaultName": "Vault Name",
  "tags": ["c8y"]
}
```

Response (multiple sessions):
```json
[
  {session1},
  {session2},
  ...
]
```

## Chrome Native Messaging Protocol

The tool implements the full Chrome Native Messaging protocol:

1. **Message Format**: Each message is prefixed with a 4-byte little-endian length header
2. **Persistent Connection**: The tool runs in a message loop, processing multiple messages
3. **Proper Termination**: Connection closes gracefully when Chrome closes the pipe (EOF)
4. **Error Handling**: Invalid messages receive error responses without terminating the connection

## Usage

### From Chrome Extension
```javascript
// Connect to native messaging host
const port = chrome.runtime.connectNative('com.example.c8y_session_1password');

// Test authentication
port.postMessage({type: "test_auth"});

// Query sessions (passwords hidden by default)
port.postMessage({
  vaults: [],
  tags: ["c8y"],
  search: "production"
});

// Query sessions with passwords revealed
port.postMessage({
  vaults: [],
  tags: ["c8y"],
  search: "production",
  reveal: true
});

// Listen for responses
port.onMessage.addListener(function(response) {
  if (response.type === 'auth_result') {
    console.log('Auth result:', response.success);
  } else if (Array.isArray(response)) {
    console.log('Multiple sessions:', response.length);
    response.forEach(session => {
      console.log(`- ${session.name}: ${session.password === '***' ? 'Password hidden' : 'Password revealed'}`);
    });
  } else if (response.name) {
    console.log('Single session:', response.name);
    console.log('Password:', response.password === '***' ? 'Hidden' : 'Revealed');
  }
});
```

### Command Line Testing
```bash
# Test with Python scripts (from scripts/nativemessaging directory)
cd scripts/nativemessaging
python3 test_native_messaging.py      # Basic protocol test
python3 test_auth.py                  # Authentication test
python3 test_comprehensive.py         # Multi-message test
python3 test_reveal_flag.py           # Reveal flag functionality test
```

