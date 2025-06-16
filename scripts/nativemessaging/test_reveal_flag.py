#!/usr/bin/env python3
"""
Test the reveal flag functionality in Chrome Native Messaging.
This script tests that passwords are hidden by default and revealed when requested.
"""

import json
import struct
import subprocess
import sys
import os

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

def test_reveal_flag():
    """Test the reveal flag functionality"""
    print("🔐 Testing Chrome Native Messaging - Reveal Flag")
    print("=" * 50)
    
    # Change to project root directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(script_dir))
    os.chdir(project_root)
    
    # Start the native messaging host
    cmd = ['./bin/c8y-session-1password']
    process = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    try:
        print("\n📋 Test 1: Session Query WITHOUT reveal flag (default)")
        print("-" * 50)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'sap-apm-gf'
        })
        print(f"→ Sent session query without reveal flag ({msg_size} bytes)")
        
        response_no_reveal = read_chrome_response(process)
        if isinstance(response_no_reveal, dict) and 'name' in response_no_reveal:
            password = response_no_reveal.get('password', '')
            if password == '***':
                print("✅ Password correctly hidden (***)")
            else:
                print(f"❌ Password should be hidden but got: {password}")
                return False
            
            print(f"   Session: {response_no_reveal['name']}")
            print(f"   Host: {response_no_reveal.get('host', 'N/A')}")
            print(f"   Username: {response_no_reveal.get('username', 'N/A')}")
            print(f"   Password: {password}")
        else:
            print("❌ Failed to get session response")
            print(f"   Response: {response_no_reveal}")
            return False
        
        print("\n📋 Test 2: Session Query WITH reveal=true")
        print("-" * 50)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'sap-apm-gf',
            'reveal': True
        })
        print(f"→ Sent session query with reveal=true ({msg_size} bytes)")
        
        response_with_reveal = read_chrome_response(process)
        if isinstance(response_with_reveal, dict) and 'name' in response_with_reveal:
            password = response_with_reveal.get('password', '')
            if password and password != '***':
                print("✅ Password correctly revealed")
                print(f"   Password length: {len(password)} characters")
            else:
                print(f"❌ Password should be revealed but got: {password}")
                return False
            
            print(f"   Session: {response_with_reveal['name']}")
            print(f"   Host: {response_with_reveal.get('host', 'N/A')}")
            print(f"   Username: {response_with_reveal.get('username', 'N/A')}")
            print(f"   Password: {'*' * len(password)} (length: {len(password)})")
        else:
            print("❌ Failed to get session response")
            print(f"   Response: {response_with_reveal}")
            return False
        
        print("\n📋 Test 3: Session Query WITH reveal=false (explicit)")
        print("-" * 50)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'sap-apm-gf',
            'reveal': False
        })
        print(f"→ Sent session query with reveal=false ({msg_size} bytes)")
        
        response_explicit_false = read_chrome_response(process)
        if isinstance(response_explicit_false, dict) and 'name' in response_explicit_false:
            password = response_explicit_false.get('password', '')
            if password == '***':
                print("✅ Password correctly hidden with explicit reveal=false")
            else:
                print(f"❌ Password should be hidden but got: {password}")
                return False
            
            print(f"   Session: {response_explicit_false['name']}")
            print(f"   Password: {password}")
        else:
            print("❌ Failed to get session response")
            return False
        
        print("\n📋 Test 4: Multiple Sessions Query WITHOUT reveal")
        print("-" * 50)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'dtm-test'
        })
        print(f"→ Sent multi-session query without reveal ({msg_size} bytes)")
        
        multi_response_no_reveal = read_chrome_response(process)
        if isinstance(multi_response_no_reveal, list) and len(multi_response_no_reveal) > 0:
            print(f"✅ Found {len(multi_response_no_reveal)} sessions")
            
            # Check that all passwords are hidden
            all_hidden = True
            for i, session in enumerate(multi_response_no_reveal[:3]):
                password = session.get('password', '')
                if password != '***':
                    print(f"❌ Session {i+1} password should be hidden but got: {password}")
                    all_hidden = False
                else:
                    print(f"   {i+1}. {session.get('name', 'Unknown')} - Password: ***")
            
            if all_hidden:
                print("✅ All passwords correctly hidden in multi-session response")
            else:
                return False
        else:
            print("❌ Failed to get multi-session response")
            return False
        
        print("\n📋 Test 5: Multiple Sessions Query WITH reveal=true")
        print("-" * 50)
        msg_size = send_chrome_message_to_process(process, {
            'vaults': [], 
            'tags': ['c8y'], 
            'search': 'dtm-test',
            'reveal': True
        })
        print(f"→ Sent multi-session query with reveal=true ({msg_size} bytes)")
        
        multi_response_with_reveal = read_chrome_response(process)
        if isinstance(multi_response_with_reveal, list) and len(multi_response_with_reveal) > 0:
            print(f"✅ Found {len(multi_response_with_reveal)} sessions")
            
            # Check that all passwords are revealed
            all_revealed = True
            for i, session in enumerate(multi_response_with_reveal[:3]):
                password = session.get('password', '')
                if not password or password == '***':
                    print(f"❌ Session {i+1} password should be revealed but got: {password}")
                    all_revealed = False
                else:
                    print(f"   {i+1}. {session.get('name', 'Unknown')} - Password: {'*' * len(password)} (length: {len(password)})")
            
            if all_revealed:
                print("✅ All passwords correctly revealed in multi-session response")
            else:
                return False
        else:
            print("❌ Failed to get multi-session response")
            return False
        
        print("\n🎉 All Reveal Flag Tests PASSED!")
        print("\n📊 Summary")
        print("-" * 30)
        print("✅ Default behavior hides passwords (no reveal flag)")
        print("✅ reveal=true reveals passwords")
        print("✅ reveal=false explicitly hides passwords")
        print("✅ Single session responses respect reveal flag")
        print("✅ Multiple session responses respect reveal flag")
        print("✅ Chrome Extension can control password visibility")
        
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
    success = test_reveal_flag()
    sys.exit(0 if success else 1)
