#!/bin/bash
set -e

# Generate private key
openssl genrsa -out /tmp/server.key 2048

# Generate self-signed certificate
openssl req -new -x509 -days 365 -key /tmp/server.key \
    -out /tmp/server.crt \
    -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=localhost"

# Combine key and certificate
cat /tmp/server.key /tmp/server.crt > /tmp/server.pem

rm /tmp/server.key
rm /tmp/server.crt
# Set proper permissions
chmod 600 /tmp/server.pem

exec socat OPENSSL-LISTEN:443,reuseaddr,fork,cert=/tmp/server.pem,verify=0 TCP:127.0.0.1:22
