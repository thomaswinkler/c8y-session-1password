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

def test_auth():
    """Test authentication with the native messaging host"""
    print("Testing authentication...")
    
    # Change to project root directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(os.path.dirname(script_dir))
    os.chdir(project_root)
    
    # Start the native messaging host (from project root)
    cmd = ['./bin/c8y-session-1password']
    process = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    
    try:
        # Send auth test message
        message = {"type": "test_auth"}
        send_chrome_message_to_process(process, message)
        
        # Read response
        response = read_chrome_response(process)
        print(f"Received response: {response}")
        
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
            print(f"Return code: {process.returncode}")
            if stderr:
                print(f"STDERR: {stderr.decode('utf-8')}")
        except subprocess.TimeoutExpired:
            print("Process timed out")
            process.kill()

if __name__ == "__main__":
    test_auth()
