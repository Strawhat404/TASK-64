#!/bin/bash
# Generate self-signed TLS certificates for local development
# Usage: ./generate.sh

CERT_DIR="$(dirname "$0")"

openssl req -x509 -newkey rsa:4096 -keyout "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -days 365 -nodes -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

echo "Generated server.crt and server.key in $CERT_DIR"
echo "To start the server with TLS, ensure these files are present."
