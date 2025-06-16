# Native Messaging Test Scripts

This directory contains Python scripts for testing Chrome Extension Native Messaging functionality.

## Scripts

- **`test_native_messaging.py`** - Basic protocol test for Chrome Native Messaging
- **`test_auth.py`** - Authentication testing script using `op signin`
- **`test_comprehensive.py`** - Comprehensive testing of multiple message types
- **`test_reveal_flag.py`** - Tests reveal flag functionality for sensitive data
- **`test_final.py`** - Final comprehensive test demonstrating all features

## Usage

Run from the project root or from this directory:

```bash
# From project root
python3 scripts/nativemessaging/test_reveal_flag.py
```

## Requirements

- Python 3.6+
- 1Password CLI (`op`) installed and configured
- Built `c8y-session-1password` binary in `bin/` directory

All scripts automatically change to the project root directory before executing tests.

## Documentation

See [Chrome Native Messaging Documentation](../../docs/chrome-native-messaging.md) for complete implementation details.
