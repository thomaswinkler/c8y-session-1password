#!/usr/bin/env python3
# test_native_messaging.py
import json
import struct
import subprocess
import sys
import os

def test_native_host():
    # Change to project root directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(script_dir))
    os.chdir(project_root)
    
    # Test data
    test_request = {"vaults": [], "tags": ["c8y"], "search": ""}
    json_data = json.dumps(test_request).encode('utf-8')
    
    # Chrome native messaging format: 4-byte length + JSON
    message_length = struct.pack('<I', len(json_data))
    full_message = message_length + json_data
    
    print(f"Sending message: {test_request}")
    print(f"Message length: {len(json_data)} bytes")
    
    # Start the native host (from project root)
    process = subprocess.Popen(
        ['./bin/c8y-session-1password'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE
    )
    
    # Send the message
    stdout, stderr = process.communicate(input=full_message, timeout=10)
    
    print(f"Return code: {process.returncode}")
    print(f"STDERR: {stderr.decode() if stderr else 'None'}")
    
    if stdout:
        try:
            # Read response length
            if len(stdout) >= 4:
                response_length = struct.unpack('<I', stdout[:4])[0]
                print(f"Response length: {response_length}")
                
                if len(stdout) >= 4 + response_length:
                    response_data = stdout[4:4+response_length]
                    response = json.loads(response_data.decode('utf-8'))
                    print(f"Response: {json.dumps(response, indent=2)}")
                else:
                    print("Response truncated")
            else:
                print("No valid response received")
        except Exception as e:
            print(f"Error parsing response: {e}")
            print(f"Raw stdout: {stdout}")
    else:
        print("No response received")

if __name__ == "__main__":
    test_native_host()