#!/bin/bash

# Color Compatibility Test Script
# This script helps test color rendering across different terminals

echo "=== Color Compatibility Test ==="
echo "Terminal: $TERM"
echo "Testing c8y-session-1password color rendering..."
echo ""

# Build the application
echo "Building application..."
go build -o c8y-session-1password . || {
    echo "❌ Build failed"
    exit 1
}

echo "✓ Build successful"
echo ""

# Run the debug colors command
echo "Running color test..."
./c8y-session-1password debug-colors

echo ""
echo "=== No-Color Mode Test ==="
echo "Testing --no-color flag..."
./c8y-session-1password debug-colors --no-color 2>/dev/null || {
    echo "Note: debug-colors doesn't support --no-color flag"
    echo "To test no-color mode with real data, use:"
    echo "  ./c8y-session-1password --no-color [filter]"
}

echo ""
echo "=== Test Instructions ==="
echo ""
echo "Please verify the colors appear correctly:"
echo "1. Title should have green text (light mode) or yellow text (dark mode)"
echo "2. Selected item should have blue text (#056AD6) with no background"
echo "3. Description border (│) should be darker blue (light mode) or light blue (dark mode)"
echo "4. Status message should be blue/light-blue text"
echo "5. No-color mode should disable all colors"
echo ""
echo "If colors don't appear correctly, please report:"
echo "- Terminal application name and version"
echo "- \$TERM environment variable: $TERM"
echo "- Color profile detected (shown above)"
echo ""
echo "Test this script in different terminals:"
echo "- VS Code integrated terminal"
echo "- Terminal.app"
echo "- iTerm2"
echo "- Any other terminal you use"
