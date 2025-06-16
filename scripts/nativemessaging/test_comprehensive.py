#!/usr/bin/env python3

import json
import struct
import subprocess
import sys
import os

def send_chrome_message_to_process(process, data):
    """Send a message using Chrome Native Messaging format"""
    json_data = json.dumps(data).encode('utf-8')
    length = len(json_data)
    
    print(f"Sending message: {data}")
    print(f"Message length: {length} bytes")
    
    # Write length prefix (4 bytes, little-endian)
    length_bytes = struct.pack('<I', length)
    process.stdin.write(length_bytes)
    
    # Write JSON data
    process.stdin.write(json_data)
    process.stdin.flush()

def read_chrome_response(process):
    """Read a response using Chrome Native Messaging format"""
    # Read 4-byte length prefix
    length_bytes = process.stdout.read(4)
    if len(length_bytes) != 4:
        print(f"Failed to read length prefix, got {len(length_bytes)} bytes")
        return None
    
    length = struct.unpack('<I', length_bytes)[0]
    print(f"Response length: {length}")
    
    # Read JSON data
    json_bytes = process.stdout.read(length)
    if len(json_bytes) != length:
        print(f"Failed to read JSON data, got {len(json_bytes)} bytes, expected {length}")
        return None
    
    return json.loads(json_bytes.decode('utf-8'))

def test_multiple_messages():
    """Test multiple messages with the native messaging host"""
    print("Testing multiple messages with persistent connection...")
    
    # Change to project root directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(script_dir))
    os.chdir(project_root)
    
    # Start the native messaging host (from project root)
    cmd = ['./bin/c8y-session-1password']
    process = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    try:
        # Test 1: Authentication test
        print("\n=== Test 1: Authentication ===")
        auth_message = {"type": "test_auth"}
        send_chrome_message_to_process(process, auth_message)
        
        auth_response = read_chrome_response(process)
        print(f"Auth response: {auth_response}")
        
        # Test 2: Session query with search filter
        print("\n=== Test 2: Session query with search ===")
        session_message = {"vaults": [], "tags": ["c8y"], "search": "dtm-test"}
        send_chrome_message_to_process(process, session_message)
        
        session_response = read_chrome_response(process)
        if isinstance(session_response, list):
            print(f"Session response: Found {len(session_response)} sessions")
            for i, session in enumerate(session_response[:3]):  # Show first 3
                print(f"  {i+1}. {session.get('name', 'Unknown')}")
        else:
            print(f"Session response: {session_response}")
        
        # Test 3: Session query for single match
        print("\n=== Test 3: Single session match ===")
        single_message = {"vaults": [], "tags": ["c8y"], "search": "sap-apm-gf"}
        send_chrome_message_to_process(process, single_message)
        
        single_response = read_chrome_response(process)
        if isinstance(single_response, dict) and 'name' in single_response:
            print(f"Single session: {single_response['name']}")
            print(f"Host: {single_response.get('host', 'N/A')}")
            print(f"Username: {single_response.get('username', 'N/A')}")
        else:
            print(f"Single response: {single_response}")
        
    except Exception as e:
        print(f"Error: {e}")
        import traceback
        traceback.print_exc()
    finally:
        # Close stdin to signal end of communication
        if process.stdin and not process.stdin.closed:
            process.stdin.close()
        
        # Wait for process to complete and get stderr
        try:
            _, stderr = process.communicate(timeout=10)
            print(f"\nReturn code: {process.returncode}")
            if stderr:
                print(f"STDERR output:\n{stderr.decode('utf-8')}")
        except subprocess.TimeoutExpired:
            print("Process timed out")
            process.kill()

if __name__ == "__main__":
    test_multiple_messages()
