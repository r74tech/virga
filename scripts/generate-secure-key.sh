#!/bin/bash

# Script to generate secure encryption keys for Virga C2

echo "Virga C2 - Secure Key Generator"
echo "==============================="
echo

# Function to generate a secure key
generate_key() {
    local key_length=$1
    local key_name=$2
    
    echo "Generating $key_name..."
    
    # Generate random bytes and encode to base64
    key=$(openssl rand -base64 $key_length)
    
    echo "$key_name: $key"
    echo
    
    # Also show the hex version
    hex_key=$(openssl rand -hex $((key_length * 3 / 4)))
    echo "Hex format: $hex_key"
    echo
}

# Generate 32-byte key for AES-256
echo "=== AES-256 Encryption Key ==="
generate_key 32 "32-byte key (recommended)"

echo "=== Alternative Key Lengths ==="
generate_key 16 "16-byte key"
generate_key 64 "64-byte key (will be hashed to 32)"

echo "=== Usage Instructions ==="
echo "1. Copy one of the generated keys above"
echo "2. Set it as an environment variable:"
echo "   export VIRGA_ENCRYPTION_KEY='<your-key-here>'"
echo
echo "3. Or add it directly to your config file:"
echo "default: configs/server.yaml"
echo "   encryption:"
echo "     key: <your-key-here>"
echo "     type: aes-256"
echo
echo "=== Security Notes ==="
echo "- NEVER use the default key in production"
echo "- Use a different key for each deployment"
echo "- Store keys securely (e.g., in a secret manager)"
echo "- Rotate keys periodically"
echo "- The key will be normalized to 32 bytes using SHA-256 if not exactly 32 bytes"
echo
echo "=== Verifying Key Randomness ==="
# Check if ent is installed
if command -v ent >/dev/null 2>&1; then
    echo "Key entropy test:"
    test_key=$(openssl rand -base64 32)
    echo "$test_key" | ent | grep -E "Entropy|Chi" || true
else
    echo "Note: Install 'ent' (entropy calculator) for randomness verification"
    echo "  macOS: brew install ent"
    echo "  Linux: apt-get install ent / yum install ent"
fi