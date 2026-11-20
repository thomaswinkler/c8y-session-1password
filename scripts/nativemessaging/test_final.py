#!/usr/bin/env python3
"""
Final comprehensive test of the Chrome Native Messaging implementation.
This script demonstrates all the key features working together.
"""

import json
import struct
import subprocess
import sys
import os
import time

def send_chrome_message_to_process(process, data):
    """Send a message using Chrome Native Messaging format"""
    json_data = json.dumps(data).encode('utf-8')
    length = len(json_data)
    
    # Write length prefix (4 bytes, little-endian)
    length_bytes = struct.pack('<I', length)
    process.stdin.write(length_bytes)
    
    # Write JSON data
    process.stdin.write(json_data)
    process.stdin.flush()
    
    return length

def read_chrome_response(process):
    """Read a response using Chrome Native Messaging format"""
    # Read 4-byte length prefix
    length_bytes = process.stdout.read(4)
    if len(length_bytes) != 4:
        return None
    
    length = struct.unpack('<I', length_bytes)[0]
    
    # Read JSON data
    json_bytes = process.stdout.read(length)
    if len(json_bytes) != length:
        return None
    
    return json.loads(json_bytes.decode('utf-8'))

def test_complete_chrome_messaging():
    """Complete test of Chrome Native Messaging functionality"""
    print("🚀 Chrome Native Messaging - Complete Test Suite")
    print("=" * 60)
    
    # Change to project root directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(script_dir))
    os.chdir(project_root)
    
    # Start the native messaging host
    cmd = ['./bin/c8y-session-1password', 'chrome-extension://test/']
    process = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    try:
        print("\n📋 Test 1: Authentication Test")
        print("-" * 30)
        msg_size = send_chrome_message_to_process(process, {'type': 'test_auth'})
        print(f"→ Sent auth request ({msg_size} bytes)")
        
        auth_response = read_chrome_response(process)
        if auth_response and auth_response.get('success'):
            print("✅ Authentication: PASSED")
            print(f"   Response: {auth_response}")
        else:
            print("❌ Authentication: FAILED")
            print(f"   Response: {auth_response}")
            return False
        
        print("\n📋 Test 2: Single Session Query")
        print("-" * 30)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'sap-apm-gf'
        })
        print(f"→ Sent session query ({msg_size} bytes)")
        
        session_response = read_chrome_response(process)
        if isinstance(session_response, dict) and 'name' in session_response:
            print("✅ Single Session Query: PASSED")
            print(f"   Found: {session_response['name']}")
            print(f"   Host: {session_response.get('host', 'N/A')}")
            print(f"   Username: {session_response.get('username', 'N/A')}")
            print(f"   Has Password: {'Yes' if session_response.get('password') else 'No'}")
        else:
            print("❌ Single Session Query: FAILED")
            print(f"   Response: {session_response}")
            return False
        
        print("\n📋 Test 3: Multiple Sessions Query")
        print("-" * 30)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'dtm-test'
        })
        print(f"→ Sent multi-session query ({msg_size} bytes)")
        
        multi_response = read_chrome_response(process)
        if isinstance(multi_response, list) and len(multi_response) > 0:
            print("✅ Multiple Sessions Query: PASSED")
            print(f"   Found {len(multi_response)} sessions:")
            for i, session in enumerate(multi_response[:5]):  # Show first 5
                print(f"   {i+1}. {session.get('name', 'Unknown')}")
            if len(multi_response) > 5:
                print(f"   ... and {len(multi_response)-5} more sessions")
        else:
            print("❌ Multiple Sessions Query: FAILED")
            print(f"   Response type: {type(multi_response)}")
            return False
        
        print("\n📋 Test 4: Empty Result Query")
        print("-" * 30)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'nonexistent-session-12345'
        })
        print(f"→ Sent empty result query ({msg_size} bytes)")
        
        empty_response = read_chrome_response(process)
        if isinstance(empty_response, dict) and 'error' in empty_response:
            print("✅ Empty Result Query: PASSED")
            print(f"   Correctly returned error: {empty_response.get('error', 'Unknown error')}")
        else:
            print("❌ Empty Result Query: FAILED")
            print(f"   Expected error response, got: {empty_response}")
            return False
        
        print("\n🎉 All Tests PASSED!")
        print("\n📊 Summary")
        print("-" * 30)
        print("✅ Chrome Native Messaging Protocol implemented correctly")
        print("✅ Persistent connection with message loop working")
        print("✅ Authentication testing functional")
        print("✅ Session filtering and querying working")
        print("✅ Single and multiple session responses working")
        print("✅ Error handling for empty results working")
        print("✅ Binary protocol with 4-byte length prefixes working")
        print("✅ JSON message parsing and response generation working")
        
        print("\n🚀 Chrome Extension Native Messaging is READY!")
        
        return True
        
    except Exception as e:
        print(f"❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        return False
        
    finally:
        # Clean shutdown
        if process.stdin and not process.stdin.closed:
            process.stdin.close()
        
        # Give the process a moment to finish
        try:
            process.wait(timeout=2)
        except subprocess.TimeoutExpired:
            process.terminate()

if __name__ == "__main__":
    success = test_complete_chrome_messaging()
    sys.exit(0 if success else 1)
