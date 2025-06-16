#!/usr/bin/env python3
"""
Test script to verify username search functionality in c8y-session-1password.
This script tests that filtering by username works correctly.
"""

import json
import subprocess
import sys
import os

# Get the path to the binary
script_dir = os.path.dirname(os.path.abspath(__file__))
binary_path = os.path.join(os.path.dirname(os.path.dirname(script_dir)), 'bin', 'c8y-session-1password')

def test_username_search():
    """Test searching by username"""
    print("Testing username search functionality...")
    
    # Test 1: Search for a username that should exist
    print("\n=== Test 1: Search by username (thomas) ===")
    test_message = {
        "vaults": [],
        "tags": ["c8y"],
        "search": "thomas",
        "reveal": False
    }
    
    result = send_message(test_message)
    if result:
        if isinstance(result, list):
            print(f"Found {len(result)} sessions matching 'thomas':")
            for i, session in enumerate(result, 1):
                print(f"  {i}. {session.get('name', 'Unknown')} - {session.get('username', 'No username')}")
        else:
            print(f"Single session found: {result.get('name', 'Unknown')} - {result.get('username', 'No username')}")
    else:
        print("No sessions found")
    
    # Test 2: Search by partial username
    print("\n=== Test 2: Search by partial username (winkler) ===")
    test_message = {
        "vaults": [],
        "tags": ["c8y"],
        "search": "winkler",
        "reveal": False
    }
    
    result = send_message(test_message)
    if result:
        if isinstance(result, list):
            print(f"Found {len(result)} sessions matching 'winkler':")
            for i, session in enumerate(result, 1):
                print(f"  {i}. {session.get('name', 'Unknown')} - {session.get('username', 'No username')}")
        else:
            print(f"Single session found: {result.get('name', 'Unknown')} - {result.get('username', 'No username')}")
    else:
        print("No sessions found")
    
    # Test 3: Search for non-existent username
    print("\n=== Test 3: Search by non-existent username (nonexistent) ===")
    test_message = {
        "vaults": [],
        "tags": ["c8y"],
        "search": "nonexistent",
        "reveal": False
    }
    
    result = send_message(test_message)
    if result:
        print(f"Unexpected result found: {result}")
    else:
        print("No sessions found (expected)")

def send_message(message):
    """Send a message to the native messaging binary and return the response"""
    try:
        # Convert message to JSON bytes
        message_json = json.dumps(message)
        message_bytes = message_json.encode('utf-8')
        
        # Prepare the message with length prefix (native messaging format)
        message_length = len(message_bytes)
        length_bytes = message_length.to_bytes(4, byteorder='little')
        
        # Start the process
        process = subprocess.Popen(
            [binary_path],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        
        # Send the message
        full_message = length_bytes + message_bytes
        stdout, stderr = process.communicate(input=full_message, timeout=30)
        
        if process.returncode != 0:
            print(f"Process failed with return code {process.returncode}")
            if stderr:
                print(f"Error: {stderr.decode('utf-8')}")
            return None
        
        if not stdout:
            print("No response received")
            return None
        
        # Parse the response (skip the 4-byte length prefix)
        if len(stdout) < 4:
            print("Response too short")
            return None
        
        response_length = int.from_bytes(stdout[:4], byteorder='little')
        response_json = stdout[4:4+response_length]
        
        try:
            response = json.loads(response_json.decode('utf-8'))
            return response
        except json.JSONDecodeError as e:
            print(f"Failed to parse JSON response: {e}")
            print(f"Raw response: {response_json}")
            return None
            
    except subprocess.TimeoutExpired:
        print("Request timed out")
        process.kill()
        return None
    except Exception as e:
        print(f"Error sending message: {e}")
        return None

if __name__ == "__main__":
    if not os.path.exists(binary_path):
        print(f"Binary not found at {binary_path}")
        print("Please run 'make build' first")
        sys.exit(1)
    
    test_username_search()
